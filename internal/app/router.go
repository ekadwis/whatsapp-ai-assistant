package app

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/ai"
	"github.com/verssache/whatsapp-ai-assistant/internal/commands"
	"github.com/verssache/whatsapp-ai-assistant/internal/finance"
	"github.com/verssache/whatsapp-ai-assistant/internal/notes"
	"github.com/verssache/whatsapp-ai-assistant/pkg/formatter"
)

const defaultPendingTTL = 5 * time.Minute

const defaultSystemPrompt = `Kamu adalah asisten keuangan pribadi yang terintegrasi dengan WhatsApp. Kamu membantu user mencatat pengeluaran dan pemasukan.

ATURAN:
1. User menulis dalam Bahasa Indonesia. Jawab dalam Bahasa Indonesia.
2. Jika user menyebut membeli/bayar/beli/spending, klasifikasikan sebagai PENGELUARAN (expense).
3. Jika user menyebut terima/gaji/dapat/earning/transfer masuk, klasifikasikan sebagai PEMASUKAN (income).
4. Parse nominal dari format Indonesia: "16k" = 16000, "1.5jt" = 1500000, "50rb" = 50000, "16.000" = 16000.
5. Jika pesan BUKAN tentang keuangan, jawab sebagai asisten AI yang helpful (general chat).
6. Selalu tentukan kategori yang paling cocok dari daftar yang tersedia.
7. Jika tidak yakin apakah pesan tentang keuangan, tanyakan klarifikasi.

TOOLS YANG TERSEDIA:
- record_transaction: Catat pengeluaran atau pemasukan
- get_report: Buat laporan keuangan (harian/mingguan/bulanan)
- set_budget: Atur budget per kategori
- save_note: Simpan catatan cepat
- edit_transaction: Edit transaksi yang sudah ada (berdasarkan ID)
- delete_transaction: Hapus transaksi (berdasarkan ID)`

type PendingAction struct {
	Transaction *ai.RecordTransactionArgs
	CreatedAt   time.Time
}

type transactionExecResult struct {
	ID          string
	Description string
	Category    string
	Amount      float64
	IsIncome    bool
	When        time.Time
}

type AppRouter struct {
	cmdRouter      *commands.Router
	llmClient      *ai.LLMClient
	financeService *finance.FinanceService
	notesService   *notes.NotesService

	pendingActions sync.Map // map[string]*PendingAction
	pendingTTL     time.Duration
	systemPrompt   string
}

func NewAppRouter(
	cmdRouter *commands.Router,
	llmClient *ai.LLMClient,
	financeService *finance.FinanceService,
	notesService *notes.NotesService,
) *AppRouter {
	return &AppRouter{
		cmdRouter:      cmdRouter,
		llmClient:      llmClient,
		financeService: financeService,
		notesService:   notesService,
		pendingTTL:     defaultPendingTTL,
		systemPrompt:   defaultSystemPrompt,
	}
}

func (r *AppRouter) HandleMessage(ctx context.Context, sender string, text string) string {
	if r == nil {
		return formatter.FormatError("Router belum siap.")
	}

	sender = strings.TrimSpace(sender)
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// 1) Check pending confirmation flow.
	if pendingRaw, ok := r.pendingActions.Load(sender); ok {
		pending, _ := pendingRaw.(*PendingAction)
		if pending == nil || pending.Transaction == nil {
			r.pendingActions.Delete(sender)
		} else if time.Since(pending.CreatedAt) > r.pendingTTL {
			r.pendingActions.Delete(sender)
		} else {
			lower := strings.ToLower(text)
			switch lower {
			case "ya", "yes", "y", "iya", "ok", "oke", "benar":
				r.pendingActions.Delete(sender)
				return r.executeTransaction(ctx, pending.Transaction)
			case "bukan", "tidak", "no", "n", "cancel", "batal":
				r.pendingActions.Delete(sender)
				return "❌ Dibatalkan."
			default:
				// User changed topic; clear stale pending and continue normal routing.
				r.pendingActions.Delete(sender)
			}
		}
	}

	// 2) Commands have priority over LLM.
	if r.cmdRouter != nil {
		if response, matched := r.cmdRouter.Route(ctx, text); matched {
			return response
		}
	}

	// 3) LLM route.
	if r.llmClient == nil {
		return formatter.FormatError("Layanan AI belum siap.")
	}

	llmResp, err := r.llmClient.Chat(ctx, r.systemPrompt, text)
	if err != nil {
		return formatter.FormatError("Maaf, sedang ada gangguan. Coba lagi nanti ya.")
	}

	// 4) Tool calls.
	if len(llmResp.ToolCalls) > 0 {
		return r.handleToolCalls(ctx, sender, llmResp.ToolCalls)
	}

	// 5) General chat response.
	content := strings.TrimSpace(llmResp.Content)
	if content == "" {
		return formatter.FormatError("Tidak ada respons dari AI. Coba lagi ya.")
	}
	return content
}

func (r *AppRouter) handleToolCalls(ctx context.Context, sender string, calls []ai.ToolCall) string {
	_ = sender

	var responses []string
	var txResults []transactionExecResult
	var budgetAlerts []string

	for _, call := range calls {
		switch call.Name {
		case "record_transaction":
			var args ai.RecordTransactionArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data transaksi tidak valid."))
				continue
			}

			result, budgetAlert, err := r.executeTransactionResult(ctx, &args)
			if err != nil {
				responses = append(responses, formatter.FormatError("Gagal mencatat: "+err.Error()))
				continue
			}
			txResults = append(txResults, result)
			if strings.TrimSpace(budgetAlert) != "" {
				budgetAlerts = append(budgetAlerts, budgetAlert)
			}

		case "get_report":
			if r.financeService == nil {
				responses = append(responses, formatter.FormatError("Service laporan belum siap."))
				continue
			}

			var args ai.GetReportArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data laporan tidak valid."))
				continue
			}

			report, err := r.financeService.GenerateReport(ctx, args.Period)
			if err != nil {
				responses = append(responses, formatter.FormatError(err.Error()))
				continue
			}

			switch normalizeReportPeriod(args.Period, report.Period) {
			case "weekly":
				responses = append(responses, formatter.FormatWeeklyReport(report.DateRange, report.TotalIncome, report.TotalExpense, report.Categories))
			case "monthly":
				responses = append(responses, formatter.FormatMonthlyReport(report.DateRange, report.TotalIncome, report.TotalExpense, report.Categories))
			default:
				responses = append(responses, formatter.FormatDailyReport(report.DateRange, report.TotalIncome, report.TotalExpense, report.Categories))
			}

		case "set_budget":
			if r.financeService == nil {
				responses = append(responses, formatter.FormatError("Service budget belum siap."))
				continue
			}

			var args ai.SetBudgetArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data budget tidak valid."))
				continue
			}

			if err := r.financeService.SetBudget(ctx, args.Category, args.Amount); err != nil {
				responses = append(responses, formatter.FormatError(err.Error()))
				continue
			}
			responses = append(responses, formatter.FormatBudgetSet(args.Category, args.Amount))

		case "save_note":
			if r.notesService == nil {
				responses = append(responses, formatter.FormatError("Service catatan belum siap."))
				continue
			}

			var args ai.SaveNoteArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data catatan tidak valid."))
				continue
			}

			if err := r.notesService.SaveNote(ctx, args.Content); err != nil {
				responses = append(responses, formatter.FormatError(err.Error()))
				continue
			}
			responses = append(responses, formatter.FormatNoteSaved(args.Content))

		case "edit_transaction":
			if r.financeService == nil {
				responses = append(responses, formatter.FormatError("Service edit belum siap."))
				continue
			}

			var args ai.EditTransactionArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data edit tidak valid."))
				continue
			}

			_, err := r.financeService.EditTransaction(ctx, args.ID, args.Field, args.Value)
			if err != nil {
				responses = append(responses, formatter.FormatError(err.Error()))
				continue
			}
			responses = append(responses, formatter.FormatTransactionEdited(args.ID, args.Field, "", args.Value))

		case "delete_transaction":
			if r.financeService == nil {
				responses = append(responses, formatter.FormatError("Service hapus belum siap."))
				continue
			}

			var args ai.DeleteTransactionArgs
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				responses = append(responses, formatter.FormatError("Format data hapus tidak valid."))
				continue
			}

			if err := r.financeService.DeleteTransaction(ctx, args.ID); err != nil {
				responses = append(responses, formatter.FormatError(err.Error()))
				continue
			}
			responses = append(responses, formatter.FormatTransactionDeleted(args.ID))
		}
	}

	if len(txResults) > 0 {
		responses = append([]string{formatCompactTransactionSummary(txResults)}, responses...)
	}

	if len(budgetAlerts) > 0 {
		responses = append(responses, strings.Join(budgetAlerts, "\n\n"))
	}

	if len(responses) == 0 {
		return formatter.FormatError("Tidak dapat memproses permintaan.")
	}

	return strings.Join(responses, "\n\n")
}

func (r *AppRouter) executeTransaction(ctx context.Context, args *ai.RecordTransactionArgs) string {
	result, budgetAlert, err := r.executeTransactionResult(ctx, args)
	if err != nil {
		return formatter.FormatError("Gagal mencatat: " + err.Error())
	}

	var response string
	if result.IsIncome {
		response = formatter.FormatIncomeRecorded(result.ID, result.Description, result.Category, result.Amount)
	} else {
		response = formatter.FormatExpenseRecorded(result.ID, result.Description, result.Category, result.Amount)
	}

	if strings.TrimSpace(budgetAlert) != "" {
		response += "\n\n" + budgetAlert
	}
	return response
}

func (r *AppRouter) executeTransactionResult(ctx context.Context, args *ai.RecordTransactionArgs) (transactionExecResult, string, error) {
	if r.financeService == nil {
		return transactionExecResult{}, "", fmt.Errorf("service transaksi belum siap")
	}
	if args == nil {
		return transactionExecResult{}, "", fmt.Errorf("data transaksi kosong")
	}

	tx, budgetAlert, err := r.financeService.RecordTransactionWithBudget(ctx, args)
	if err != nil {
		return transactionExecResult{}, "", err
	}

	return transactionExecResult{
		ID:          tx.ID,
		Description: tx.Description,
		Category:    tx.Category,
		Amount:      tx.Amount,
		IsIncome:    strings.EqualFold(strings.TrimSpace(args.Type), "income"),
		When:        tx.Date,
	}, budgetAlert, nil
}

func formatCompactTransactionSummary(results []transactionExecResult) string {
	if len(results) == 0 {
		return formatter.FormatError("Tidak ada transaksi yang dapat ditampilkan.")
	}

	allIncome := true
	allExpense := true
	for _, r := range results {
		if r.IsIncome {
			allExpense = false
		} else {
			allIncome = false
		}
	}

	title := "✅ *Transaksi berhasil dicatat!*"
	if allExpense {
		title = fmt.Sprintf("✅ *%d pengeluaran berhasil dicatat!*", len(results))
	} else if allIncome {
		title = fmt.Sprintf("✅ *%d pemasukan berhasil dicatat!*", len(results))
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	totalAmount := 0.0
	for i, item := range results {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("🆔 ID: ")
		b.WriteString(item.ID)
		b.WriteString("\n📝 Deskripsi: ")
		b.WriteString(item.Description)
		b.WriteString("\n📂 Kategori: ")
		b.WriteString(item.Category)
		b.WriteString("\n💰 Jumlah: ")
		b.WriteString(formatIDRCompact(item.Amount))

		totalAmount += item.Amount
	}

	icon := "💰"
	label := "Total"
	if allExpense {
		icon = "💸"
		label = "Total Pengeluaran"
	} else if allIncome {
		icon = "💵"
		label = "Total Pemasukan"
	}

	b.WriteString("\n\n")
	b.WriteString(icon)
	b.WriteString(" ")
	b.WriteString(label)
	b.WriteString(": ")
	b.WriteString(formatIDRCompact(totalAmount))

	last := results[len(results)-1].When.In(time.FixedZone("WIB", 7*60*60))
	b.WriteString("\n\n📅 ")
	b.WriteString(last.Format("02 Jan 2006, 15:04 WIB"))

	return b.String()
}

func formatIDRCompact(amount float64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}

	rounded := math.Round(amount*100) / 100
	intPart := int64(rounded)
	fracPart := int(math.Round((rounded - float64(intPart)) * 100))

	intText := withThousandDotsCompact(strconv.FormatInt(intPart, 10))
	if fracPart == 0 {
		return sign + "Rp " + intText
	}
	return fmt.Sprintf("%sRp %s,%02d", sign, intText, fracPart)
}

func withThousandDotsCompact(s string) string {
	if len(s) <= 3 {
		return s
	}

	prefix := len(s) % 3
	if prefix == 0 {
		prefix = 3
	}

	var b strings.Builder
	b.WriteString(s[:prefix])
	for i := prefix; i < len(s); i += 3 {
		b.WriteString(".")
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func normalizeReportPeriod(raw string, fallback string) string {
	p := strings.ToLower(strings.TrimSpace(raw))
	switch p {
	case "daily", "harian", "hari ini":
		return "daily"
	case "weekly", "mingguan", "minggu ini":
		return "weekly"
	case "monthly", "bulanan", "bulan ini":
		return "monthly"
	}

	fb := strings.ToLower(strings.TrimSpace(fallback))
	switch fb {
	case "daily", "weekly", "monthly":
		return fb
	default:
		return "daily"
	}
}
