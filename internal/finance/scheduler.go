package finance

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
	"github.com/verssache/whatsapp-ai-assistant/internal/whatsapp"
	"github.com/verssache/whatsapp-ai-assistant/pkg/chart"
	"github.com/verssache/whatsapp-ai-assistant/pkg/formatter"
)

type ReportScheduler struct {
	financeService *FinanceService
	messenger      whatsapp.Messenger
	recipient      string
	interval       time.Duration
	now            func() time.Time

	stopCh  chan struct{}
	started atomic.Bool
	wg      sync.WaitGroup

	lastDailyReport   string
	lastWeeklyReport  string
	lastMonthlyReport string
}

func NewReportScheduler(financeService *FinanceService, messenger whatsapp.Messenger, recipient string) *ReportScheduler {
	return &ReportScheduler{
		financeService: financeService,
		messenger:      messenger,
		recipient:      strings.TrimSpace(recipient),
		interval:       30 * time.Second,
		now: func() time.Time {
			return time.Now().In(sheets.WIB)
		},
		stopCh: make(chan struct{}),
	}
}

func (s *ReportScheduler) Start(ctx context.Context) error {
	if s == nil {
		return fmt.Errorf("report scheduler is nil")
	}
	if s.financeService == nil {
		return fmt.Errorf("finance service is nil")
	}
	if s.messenger == nil {
		return fmt.Errorf("messenger is nil")
	}
	if s.recipient == "" {
		return fmt.Errorf("recipient is empty")
	}

	if s.started.Swap(true) {
		return nil
	}

	s.wg.Add(1)
	go s.loop()
	return nil
}

func (s *ReportScheduler) Stop() {
	if s == nil || !s.started.Load() {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
	s.started.Store(false)
}

func (s *ReportScheduler) loop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAndSendScheduledReports(context.Background())
		}
	}
}

func (s *ReportScheduler) checkAndSendScheduledReports(ctx context.Context) {
	now := s.now()
	hour := now.Hour()
	min := now.Minute()

	// 1) Daily Report: 00:00 WIB
	if hour == 0 && min == 0 {
		dailyKey := now.Format("2006-01-02")
		if s.lastDailyReport != dailyKey {
			s.lastDailyReport = dailyKey
			s.sendScheduledDailyReport(ctx, now)
		}
	}

	// 2) Weekly Report: Monday 00:01 WIB
	if now.Weekday() == time.Monday && hour == 0 && min == 1 {
		weeklyKey := now.Format("2006-01-02")
		if s.lastWeeklyReport != weeklyKey {
			s.lastWeeklyReport = weeklyKey
			s.sendScheduledWeeklyReport(ctx, now)
		}
	}

	// 3) Monthly Report: 1st of month 00:01 WIB
	if now.Day() == 1 && hour == 0 && min == 1 {
		monthlyKey := now.Format("2006-01")
		if s.lastMonthlyReport != monthlyKey {
			s.lastMonthlyReport = monthlyKey
			s.sendScheduledMonthlyReport(ctx, now)
		}
	}
}

func (s *ReportScheduler) sendScheduledDailyReport(ctx context.Context, now time.Time) {
	// Send report for yesterday
	yesterday := now.AddDate(0, 0, -1)
	tabName := tabNameForTime(yesterday)

	txs, err := s.financeService.repo.GetTransactions(ctx, tabName)
	if err != nil {
		log.Printf("❌ [AutoReport] Gagal membaca transaksi harian: %v", err)
		return
	}

	filtered, dateRange, err := filterTransactionsByPeriod(txs, yesterday, "daily")
	if err != nil {
		log.Printf("❌ [AutoReport] Gagal memfilter transaksi: %v", err)
		return
	}

	totalIncome := 0.0
	totalExpense := 0.0
	categories := make(map[string]float64)
	for _, tx := range filtered {
		if tx.Type == sheets.Income {
			totalIncome += tx.Amount
		} else {
			totalExpense += tx.Amount
			normCat := normalizeCategoryForType(tx.Category, sheets.Expense)
			categories[normCat] += tx.Amount
		}
	}

	topCats := topNCategories(categories, 5)
	msg := fmt.Sprintf("🌙 *Laporan Harian Otomatis*\n\n%s", formatter.FormatDailyReport(dateRange, totalIncome, totalExpense, topCats))

	s.sendReportWithChart(ctx, msg, "Pengeluaran Harian", topCats)
}

func (s *ReportScheduler) sendScheduledWeeklyReport(ctx context.Context, now time.Time) {
	// Previous week report
	lastWeek := now.AddDate(0, 0, -7)
	report, err := s.financeService.GenerateReport(ctx, "weekly")
	if err != nil {
		log.Printf("❌ [AutoReport] Gagal membuat laporan mingguan: %v", err)
		return
	}

	_ = lastWeek
	msg := fmt.Sprintf("📅 *Laporan Mingguan Otomatis*\n\n%s", formatter.FormatWeeklyReport(report.DateRange, report.TotalIncome, report.TotalExpense, report.Categories))

	s.sendReportWithChart(ctx, msg, "Pengeluaran Mingguan", report.Categories)
}

func (s *ReportScheduler) sendScheduledMonthlyReport(ctx context.Context, now time.Time) {
	// Report for previous month
	prevMonth := now.AddDate(0, -1, 0)
	tabName := tabNameForTime(prevMonth)

	txs, err := s.financeService.repo.GetTransactions(ctx, tabName)
	if err != nil {
		log.Printf("❌ [AutoReport] Gagal membaca transaksi bulan lalu: %v", err)
		return
	}

	totalIncome := 0.0
	totalExpense := 0.0
	categories := make(map[string]float64)
	for _, tx := range txs {
		if tx.Type == sheets.Income {
			totalIncome += tx.Amount
		} else {
			totalExpense += tx.Amount
			normCat := normalizeCategoryForType(tx.Category, sheets.Expense)
			categories[normCat] += tx.Amount
		}
	}

	topCats := topNCategories(categories, 5)
	dateRange := tabName
	msg := fmt.Sprintf("🗓️ *Laporan Bulanan Otomatis*\n\n%s", formatter.FormatMonthlyReport(dateRange, totalIncome, totalExpense, topCats))

	s.sendReportWithChart(ctx, msg, "Pengeluaran Bulanan "+tabName, topCats)
}

func (s *ReportScheduler) sendReportWithChart(ctx context.Context, textMsg, chartTitle string, categories map[string]float64) {
	_ = s.messenger.SendPresence(ctx, s.recipient)

	// Attempt chart generation
	if len(categories) > 0 {
		chartURL, err := chart.GenerateCategoryPieChartURL(chartTitle, categories)
		if err == nil {
			imgBytes, err := chart.DownloadChartImage(ctx, chartURL)
			if err == nil && len(imgBytes) > 0 {
				if err := s.messenger.SendImage(ctx, s.recipient, imgBytes, textMsg); err == nil {
					log.Printf("✅ [AutoReport] Laporan dengan chart berhasil dikirim ke %s", s.recipient)
					return
				}
			}
		}
	}

	// Fallback to text message
	if err := s.messenger.SendText(ctx, s.recipient, textMsg); err != nil {
		log.Printf("❌ [AutoReport] Gagal mengirim teks laporan: %v", err)
	} else {
		log.Printf("✅ [AutoReport] Teks laporan berhasil dikirim ke %s", s.recipient)
	}
}
