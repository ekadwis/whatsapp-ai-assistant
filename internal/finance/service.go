package finance

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/ai"
	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
	"github.com/verssache/whatsapp-ai-assistant/pkg/formatter"
)

var monthNamesID = map[time.Month]string{
	time.January:   "Januari",
	time.February:  "Februari",
	time.March:     "Maret",
	time.April:     "April",
	time.May:       "Mei",
	time.June:      "Juni",
	time.July:      "Juli",
	time.August:    "Agustus",
	time.September: "September",
	time.October:   "Oktober",
	time.November:  "November",
	time.December:  "Desember",
}

var expenseCategoriesCanonical = []string{
	"Makanan", "Transportasi", "Rumah Tangga", "Belanja",
	"Kesehatan", "Pendidikan", "Hiburan", "Fashion",
	"Komunikasi", "Perawatan", "Sosial", "Lainnya",
}

var incomeCategoriesCanonical = []string{
	"Gaji", "Freelance", "Investasi", "Hadiah", "Transfer", "Lainnya",
}

var categoryAliases = map[string]string{
	"makanan & minuman": "Makanan",
	"makanan dan minuman": "Makanan",
	"food & beverage": "Makanan",
	"food and beverage": "Makanan",
	"f&b": "Makanan",
	"makan minum": "Makanan",
}

type FinanceService struct {
	repo sheets.SheetRepository
	ai   *ai.LLMClient
}

type ReportData struct {
	Period       string
	DateRange    string
	TotalIncome  float64
	TotalExpense float64
	NetBalance   float64
	Categories   map[string]float64
}

type AllTimeReportData struct {
	TotalIncome   float64
	TotalExpense  float64
	NetBalance    float64
	TotalCount    int
	IncomeCount   int
	ExpenseCount  int
	EarliestDate  time.Time
	LatestDate    time.Time
	TopCategories map[string]float64
}

func NewFinanceService(repo sheets.SheetRepository, aiClient *ai.LLMClient) *FinanceService {
	return &FinanceService{
		repo: repo,
		ai:   aiClient,
	}
}

// RecordTransaction keeps backward compatibility and ignores budget alert output.
func (s *FinanceService) RecordTransaction(ctx context.Context, args *ai.RecordTransactionArgs) (*sheets.Transaction, error) {
	tx, _, err := s.RecordTransactionWithBudget(ctx, args)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// RecordTransactionWithBudget creates and stores one transaction to the current WIB month tab
// and returns optional budget alert text for expense transactions.
func (s *FinanceService) RecordTransactionWithBudget(ctx context.Context, args *ai.RecordTransactionArgs) (*sheets.Transaction, string, error) {
	if s == nil {
		return nil, "", fmt.Errorf("finance service is nil")
	}
	if s.repo == nil {
		return nil, "", fmt.Errorf("sheet repository is nil")
	}
	if args == nil {
		return nil, "", fmt.Errorf("record transaction args is nil")
	}
	if strings.TrimSpace(args.Description) == "" {
		return nil, "", fmt.Errorf("description is required")
	}
	if strings.TrimSpace(args.Category) == "" {
		return nil, "", fmt.Errorf("category is required")
	}
	if args.Amount <= 0 {
		return nil, "", fmt.Errorf("amount must be greater than zero")
	}

	now := nowWIB()
	tabName := tabNameForTime(now)

	if err := s.repo.EnsureTabExists(ctx, tabName); err != nil {
		return nil, "", fmt.Errorf("failed to ensure tab %q: %w", tabName, err)
	}

	txType := toTransactionType(args.Type)

	txID, err := s.nextTransactionIDFromSheet(ctx, now)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	tx := &sheets.Transaction{
		ID:          txID,
		Date:        now,
		Type:        txType,
		Category:    normalizeCategoryForType(args.Category, txType),
		Description: strings.TrimSpace(args.Description),
		Amount:      args.Amount,
	}

	if err := s.repo.AppendTransaction(ctx, tx); err != nil {
		return nil, "", fmt.Errorf("failed to append transaction: %w", err)
	}

	var budgetAlert string
	if tx.Type == sheets.Expense {
		alert, err := s.CheckBudget(ctx, tx.Category)
		if err == nil {
			budgetAlert = alert
		}
	}

	return tx, budgetAlert, nil
}

func (s *FinanceService) SetBudget(ctx context.Context, category string, amount float64) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("sheet repository is nil")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		return fmt.Errorf("kategori tidak boleh kosong")
	}
	if amount <= 0 {
		return fmt.Errorf("jumlah budget harus lebih dari 0")
	}
	return s.repo.SetBudget(ctx, category, amount)
}

func (s *FinanceService) CheckBudget(ctx context.Context, category string) (string, error) {
	if s == nil || s.repo == nil {
		return "", fmt.Errorf("sheet repository is nil")
	}
	category = normalizeCategoryForType(category, sheets.Expense)
	if category == "" {
		return "", nil
	}

	budget, err := s.repo.GetBudget(ctx, category)
	if err != nil {
		return "", err
	}
	if budget <= 0 {
		return "", nil
	}

	spent, err := s.repo.GetCategoryTotal(ctx, tabNameForTime(nowWIB()), category)
	if err != nil {
		return "", err
	}

	remaining := budget - spent
	if remaining < 0 {
		return formatter.FormatBudgetAlert(category, budget, spent, remaining), nil
	}
	if remaining <= budget*0.2 {
		return formatter.FormatBudgetAlert(category, budget, spent, remaining), nil
	}

	return "", nil
}

func (s *FinanceService) GenerateReport(ctx context.Context, period string) (*ReportData, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("sheet repository is nil")
	}

	now := nowWIB()
	tabName := tabNameForTime(now)

	txs, err := s.repo.GetTransactions(ctx, tabName)
	if err != nil {
		return nil, fmt.Errorf("failed to load transactions: %w", err)
	}

	mode := normalizePeriod(period)
	filtered, dateRange, err := filterTransactionsByPeriod(txs, now, mode)
	if err != nil {
		return nil, err
	}

	report := &ReportData{
		Period:     mode,
		DateRange:  dateRange,
		Categories: map[string]float64{},
	}

	for _, tx := range filtered {
		if tx.Type == sheets.Income {
			report.TotalIncome += tx.Amount
			continue
		}
		report.TotalExpense += tx.Amount
		normalizedCategory := normalizeCategoryForType(tx.Category, sheets.Expense)
		report.Categories[normalizedCategory] += tx.Amount
	}

	report.Categories = topNCategories(report.Categories, 5)
	report.NetBalance = report.TotalIncome - report.TotalExpense

	return report, nil
}

// GenerateAllTimeReport generates an aggregated report of all transactions since inception.
func (s *FinanceService) GenerateAllTimeReport(ctx context.Context) (*AllTimeReportData, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("sheet repository is nil")
	}

	allTxs, err := s.repo.GetAllTransactions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load all transactions: %w", err)
	}

	out := &AllTimeReportData{
		TotalCount:    len(allTxs),
		TopCategories: make(map[string]float64),
	}

	for _, tx := range allTxs {
		if out.EarliestDate.IsZero() || tx.Date.Before(out.EarliestDate) {
			out.EarliestDate = tx.Date
		}
		if out.LatestDate.IsZero() || tx.Date.After(out.LatestDate) {
			out.LatestDate = tx.Date
		}

		if tx.Type == sheets.Income {
			out.TotalIncome += tx.Amount
			out.IncomeCount++
		} else {
			out.TotalExpense += tx.Amount
			out.ExpenseCount++
			normCat := normalizeCategoryForType(tx.Category, sheets.Expense)
			out.TopCategories[normCat] += tx.Amount
		}
	}

	out.NetBalance = out.TotalIncome - out.TotalExpense
	out.TopCategories = topNCategories(out.TopCategories, 5)

	return out, nil
}

// GenerateSalaryCycleReport generates a report for the salary cycle:
// from the 23rd of previous month to the 23rd of current month (or today if current day < 23).
func (s *FinanceService) GenerateSalaryCycleReport(ctx context.Context) (*ReportData, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("sheet repository is nil")
	}

	now := nowWIB()

	var startDate, endDate time.Time
	if now.Day() < 23 {
		// e.g. 13 March -> 23 Feb 00:00:00 to 13 March 23:59:59
		prevMonth := now.AddDate(0, -1, 0)
		startDate = time.Date(prevMonth.Year(), prevMonth.Month(), 23, 0, 0, 0, 0, sheets.WIB)
		endDate = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, sheets.WIB)
	} else {
		// e.g. 26 March -> 23 Feb 00:00:00 to 23 March 23:59:59
		prevMonth := now.AddDate(0, -1, 0)
		startDate = time.Date(prevMonth.Year(), prevMonth.Month(), 23, 0, 0, 0, 0, sheets.WIB)
		endDate = time.Date(now.Year(), now.Month(), 23, 23, 59, 59, 999999999, sheets.WIB)
	}

	txs, err := s.repo.GetTransactionsBetweenDates(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions for salary cycle: %w", err)
	}

	dateRange := fmt.Sprintf("%02d %s %d - %02d %s %d",
		startDate.Day(), monthNamesID[startDate.Month()], startDate.Year(),
		endDate.Day(), monthNamesID[endDate.Month()], endDate.Year(),
	)

	report := &ReportData{
		Period:     "gajian",
		DateRange:  dateRange,
		Categories: make(map[string]float64),
	}

	for _, tx := range txs {
		if tx.Type == sheets.Income {
			report.TotalIncome += tx.Amount
			continue
		}
		report.TotalExpense += tx.Amount
		normCat := normalizeCategoryForType(tx.Category, sheets.Expense)
		report.Categories[normCat] += tx.Amount
	}

	report.Categories = topNCategories(report.Categories, 5)
	report.NetBalance = report.TotalIncome - report.TotalExpense

	return report, nil
}

// GenerateFinancialEvaluation evaluates current financial health and offers suggestions via LLM.
func (s *FinanceService) GenerateFinancialEvaluation(ctx context.Context) (string, error) {
	if s == nil || s.repo == nil {
		return "", fmt.Errorf("sheet repository is nil")
	}
	if s.ai == nil {
		return "", fmt.Errorf("ai client is nil")
	}

	now := nowWIB()
	tabName := tabNameForTime(now)
	txs, err := s.repo.GetTransactions(ctx, tabName)
	if err != nil {
		return "", fmt.Errorf("failed to load transactions: %w", err)
	}

	if len(txs) == 0 {
		return "📊 *Evaluasi Finansial*\n\nBelum ada data transaksi yang dicatat untuk bulan ini.", nil
	}

	totalIncome := 0.0
	totalExpense := 0.0
	catTotals := make(map[string]float64)

	var txSummaries []string
	for _, tx := range txs {
		if tx.Type == sheets.Income {
			totalIncome += tx.Amount
		} else {
			totalExpense += tx.Amount
			normCat := normalizeCategoryForType(tx.Category, sheets.Expense)
			catTotals[normCat] += tx.Amount
		}
		txSummaries = append(txSummaries, fmt.Sprintf("- [%s] %s (%s): %s",
			tx.Date.Format("02/01"), tx.Description, tx.Category, formatter.FormatIDR(tx.Amount)))
	}

	limitTx := txSummaries
	if len(limitTx) > 40 {
		limitTx = limitTx[len(limitTx)-40:]
	}

	prompt := fmt.Sprintf(`Bulan ini: %s
Total Pemasukan: %s
Total Pengeluaran: %s
Sisa Saldo: %s

Breakdown Pengeluaran:
`, tabName, formatter.FormatIDR(totalIncome), formatter.FormatIDR(totalExpense), formatter.FormatIDR(totalIncome-totalExpense))

	for cat, amt := range catTotals {
		prompt += fmt.Sprintf("- %s: %s\n", cat, formatter.FormatIDR(amt))
	}

	prompt += "\nDaftar Transaksi Terbaru:\n" + strings.Join(limitTx, "\n")

	systemPrompt := `Kamu adalah financial advisor AI pribadi yang bijak, ramah, dan profesional.
Tugasmu:
1. Berikan analisis singkat dan to the point tentang kondisi keuangan user bulan ini.
2. Identifikasi pos pengeluaran mana yang paling besar atau berpotensi boros/bocor halus.
3. Berikan 3 rekomendasi atau saran tindakan konkret yang realistis dan aplikatif.
Gunakan Bahasa Indonesia, formatting WhatsApp yang rapi dengan emoji (*bold*, bullet point), dan hindari kata-kata bertele-tele.`

	resp, err := s.ai.Chat(ctx, systemPrompt, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to get AI evaluation: %w", err)
	}

	return resp.Content, nil
}

func (s *FinanceService) EditTransaction(ctx context.Context, id, field, value string) (*sheets.Transaction, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("sheet repository is nil")
	}
	id = strings.TrimSpace(id)
	field = strings.ToLower(strings.TrimSpace(field))
	value = strings.TrimSpace(value)

	if id == "" {
		return nil, fmt.Errorf("id transaksi tidak boleh kosong")
	}
	if field == "" {
		return nil, fmt.Errorf("field tidak boleh kosong")
	}
	if value == "" {
		return nil, fmt.Errorf("nilai baru tidak boleh kosong")
	}

	tx, rowIndex, tabName, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("transaksi %s tidak ditemukan: %w", id, err)
	}

	switch field {
	case "jumlah", "amount":
		amount, err := parseIDRAmountLocal(value)
		if err != nil {
			return nil, fmt.Errorf("jumlah tidak valid: %w", err)
		}
		tx.Amount = amount
	case "kategori", "category":
		tx.Category = normalizeCategoryForType(value, tx.Type)
	case "deskripsi", "description":
		tx.Description = value
	default:
		return nil, fmt.Errorf("field tidak dikenal: %s. Gunakan: jumlah, kategori, atau deskripsi", field)
	}

	if err := s.repo.UpdateTransaction(ctx, tabName, rowIndex, tx); err != nil {
		return nil, fmt.Errorf("gagal memperbarui transaksi: %w", err)
	}

	return tx, nil
}

func (s *FinanceService) DeleteTransaction(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("sheet repository is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id transaksi tidak boleh kosong")
	}

	_, rowIndex, tabName, err := s.repo.GetTransactionByID(ctx, id)
	if err != nil {
		return fmt.Errorf("transaksi %s tidak ditemukan: %w", id, err)
	}

	if err := s.repo.DeleteTransaction(ctx, tabName, rowIndex); err != nil {
		return fmt.Errorf("gagal menghapus transaksi: %w", err)
	}

	return nil
}

func nowWIB() time.Time {
	return time.Now().In(sheets.WIB)
}

func tabNameForTime(t time.Time) string {
	monthName, ok := monthNamesID[t.In(sheets.WIB).Month()]
	if !ok {
		monthName = t.In(sheets.WIB).Month().String()
	}
	return fmt.Sprintf("%s %d", monthName, t.In(sheets.WIB).Year())
}

func toTransactionType(raw string) sheets.TransactionType {
	if strings.EqualFold(strings.TrimSpace(raw), "income") {
		return sheets.Income
	}
	return sheets.Expense
}

func (s *FinanceService) nextTransactionIDFromSheet(ctx context.Context, now time.Time) (string, error) {
	tabName := tabNameForTime(now)
	txs, err := s.repo.GetTransactions(ctx, tabName)
	if err != nil {
		return "", err
	}

	datePrefix := now.In(sheets.WIB).Format("20060102") + "-"
	maxCounter := 0

	for _, tx := range txs {
		id := strings.TrimSpace(tx.ID)
		if !strings.HasPrefix(id, datePrefix) {
			continue
		}

		parts := strings.Split(id, "-")
		if len(parts) != 2 {
			continue
		}

		counter, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		if counter > maxCounter {
			maxCounter = counter
		}
	}

	return fmt.Sprintf("%s%03d", datePrefix, maxCounter+1), nil
}

func normalizePeriod(period string) string {
	p := strings.ToLower(strings.TrimSpace(period))
	switch p {
	case "", "daily", "hari ini", "harian":
		return "daily"
	case "weekly", "minggu ini", "mingguan":
		return "weekly"
	case "monthly", "bulan ini", "bulanan":
		return "monthly"
	default:
		return p
	}
}

func filterTransactionsByPeriod(txs []sheets.Transaction, now time.Time, mode string) ([]sheets.Transaction, string, error) {
	switch mode {
	case "daily":
		todayKey := now.In(sheets.WIB).Format("20060102")
		var out []sheets.Transaction
		for _, tx := range txs {
			if tx.Date.In(sheets.WIB).Format("20060102") == todayKey {
				out = append(out, tx)
			}
		}
		return out, formatDateID(now), nil

	case "weekly":
		start := now.AddDate(0, 0, -6)
		startKey := start.In(sheets.WIB).Format("20060102")
		endKey := now.In(sheets.WIB).Format("20060102")
		var out []sheets.Transaction
		for _, tx := range txs {
			k := tx.Date.In(sheets.WIB).Format("20060102")
			if k >= startKey && k <= endKey {
				out = append(out, tx)
			}
		}
		return out, fmt.Sprintf("%s - %s", formatDateID(start), formatDateID(now)), nil

	case "monthly":
		return txs, fmt.Sprintf("%s %d", monthNamesID[now.Month()], now.Year()), nil

	default:
		return nil, "", fmt.Errorf("periode tidak dikenal: %s", mode)
	}
}

func topNCategories(in map[string]float64, n int) map[string]float64 {
	type kv struct {
		Key string
		Val float64
	}
	items := make([]kv, 0, len(in))
	for k, v := range in {
		items = append(items, kv{Key: k, Val: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Val == items[j].Val {
			return items[i].Key < items[j].Key
		}
		return items[i].Val > items[j].Val
	})
	if len(items) > n {
		items = items[:n]
	}
	out := make(map[string]float64, len(items))
	for _, it := range items {
		out[it.Key] = it.Val
	}
	return out
}

func formatDateID(t time.Time) string {
	local := t.In(sheets.WIB)
	month := monthNamesID[local.Month()]
	return fmt.Sprintf("%02d %s %d", local.Day(), month, local.Year())
}

func normalizeCategoryForType(raw string, txType sheets.TransactionType) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "Lainnya"
	}

	key := strings.ToLower(s)
	if alias, ok := categoryAliases[key]; ok {
		s = alias
	}

	var canonical []string
	if txType == sheets.Income {
		canonical = incomeCategoriesCanonical
	} else {
		canonical = expenseCategoriesCanonical
	}

	for _, c := range canonical {
		if strings.EqualFold(c, s) {
			return c
		}
	}

	return "Lainnya"
}

func parseIDRAmountLocal(s string) (float64, error) {
	raw := strings.ToLower(strings.TrimSpace(s))
	if raw == "" {
		return 0, fmt.Errorf("nilai kosong")
	}

	raw = strings.TrimSpace(strings.TrimPrefix(raw, "rp."))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "rp"))
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "idr"))
	raw = strings.ReplaceAll(raw, " ", "")

	multiplier := 1.0
	switch {
	case strings.HasSuffix(raw, "juta"):
		multiplier = 1_000_000
		raw = strings.TrimSuffix(raw, "juta")
	case strings.HasSuffix(raw, "jt"):
		multiplier = 1_000_000
		raw = strings.TrimSuffix(raw, "jt")
	case strings.HasSuffix(raw, "ribu"):
		multiplier = 1_000
		raw = strings.TrimSuffix(raw, "ribu")
	case strings.HasSuffix(raw, "rb"):
		multiplier = 1_000
		raw = strings.TrimSuffix(raw, "rb")
	case strings.HasSuffix(raw, "k"):
		multiplier = 1_000
		raw = strings.TrimSuffix(raw, "k")
	}

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("format jumlah tidak valid")
	}

	if strings.Contains(raw, ",") && strings.Contains(raw, ".") {
		if strings.LastIndex(raw, ",") > strings.LastIndex(raw, ".") {
			raw = strings.ReplaceAll(raw, ".", "")
			raw = strings.ReplaceAll(raw, ",", ".")
		} else {
			raw = strings.ReplaceAll(raw, ",", "")
		}
	} else if strings.Contains(raw, ",") {
		raw = strings.ReplaceAll(raw, ",", ".")
	} else if strings.Count(raw, ".") >= 1 {
		parts := strings.Split(raw, ".")
		grouped := true
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) != 3 {
				grouped = false
				break
			}
		}
		if grouped {
			raw = strings.ReplaceAll(raw, ".", "")
		}
	}

	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("format jumlah tidak valid")
	}
	v *= multiplier

	if v <= 0 {
		return 0, fmt.Errorf("jumlah harus lebih dari 0")
	}

	return v, nil
}
