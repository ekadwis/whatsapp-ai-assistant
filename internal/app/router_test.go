package app_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/app"
	"github.com/verssache/whatsapp-ai-assistant/internal/commands"
	"github.com/verssache/whatsapp-ai-assistant/internal/finance"
	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

type mockSheetRepo struct {
	txs      []sheets.Transaction
	allTxs   []sheets.Transaction
	rangeTxs []sheets.Transaction
}

func (m *mockSheetRepo) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error {
	return nil
}
func (m *mockSheetRepo) GetTransactions(ctx context.Context, tabName string) ([]sheets.Transaction, error) {
	return m.txs, nil
}
func (m *mockSheetRepo) GetAllTransactions(ctx context.Context) ([]sheets.Transaction, error) {
	return m.allTxs, nil
}
func (m *mockSheetRepo) GetTransactionsBetweenDates(ctx context.Context, startDate, endDate time.Time) ([]sheets.Transaction, error) {
	return m.rangeTxs, nil
}
func (m *mockSheetRepo) GetTransactionByID(ctx context.Context, id string) (*sheets.Transaction, int, string, error) {
	return nil, 0, "", nil
}
func (m *mockSheetRepo) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *sheets.Transaction) error {
	return nil
}
func (m *mockSheetRepo) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error {
	return nil
}
func (m *mockSheetRepo) AppendNote(ctx context.Context, note *sheets.Note) error {
	return nil
}
func (m *mockSheetRepo) GetBudget(ctx context.Context, category string) (float64, error) {
	return 0, nil
}
func (m *mockSheetRepo) SetBudget(ctx context.Context, category string, amount float64) error {
	return nil
}
func (m *mockSheetRepo) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	return 0, nil
}
func (m *mockSheetRepo) EnsureTabExists(ctx context.Context, tabName string) error {
	return nil
}
func (m *mockSheetRepo) FormatHeaders(ctx context.Context, tabName string) error {
	return nil
}
func (m *mockSheetRepo) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	return nil
}
func (m *mockSheetRepo) InitDashboard(ctx context.Context) error {
	return nil
}
func (m *mockSheetRepo) InitBudgetTab(ctx context.Context) error {
	return nil
}
func (m *mockSheetRepo) InitNotesTab(ctx context.Context) error {
	return nil
}
func (m *mockSheetRepo) InitReminderTab(ctx context.Context) error {
	return nil
}
func (m *mockSheetRepo) AppendReminder(ctx context.Context, reminder *sheets.Reminder) error {
	return nil
}
func (m *mockSheetRepo) ListActiveReminders(ctx context.Context) ([]sheets.Reminder, error) {
	return nil, nil
}
func (m *mockSheetRepo) GetReminderByID(ctx context.Context, id string) (*sheets.Reminder, int, error) {
	return nil, 0, nil
}
func (m *mockSheetRepo) UpdateReminder(ctx context.Context, rowIndex int, reminder *sheets.Reminder) error {
	return nil
}
func (m *mockSheetRepo) ListDueReminders(ctx context.Context, now time.Time) ([]sheets.Reminder, error) {
	return nil, nil
}

func TestAppRouterSalaryCycleAndAllTimeCommands(t *testing.T) {
	repo := &mockSheetRepo{
		allTxs: []sheets.Transaction{
			{ID: "20260115-001", Date: time.Now(), Type: sheets.Income, Amount: 10000000, Category: "Gaji", Description: "Gaji"},
			{ID: "20260220-001", Date: time.Now(), Type: sheets.Expense, Amount: 2000000, Category: "Makanan", Description: "Makan"},
		},
		rangeTxs: []sheets.Transaction{
			{ID: "20260225-001", Date: time.Now(), Type: sheets.Income, Amount: 8000000, Category: "Gaji", Description: "Gaji"},
		},
	}

	finService := finance.NewFinanceService(repo, nil)

	cmdRouter := commands.NewRouter()
	cmdRouter.Register("/laporan", commands.NewReportHandlerFactory(finService).Handler)

	appRouter := app.NewAppRouter(cmdRouter, nil, finService, nil)

	// 1) Test /laporan gajian
	resp := appRouter.HandleMessage(context.Background(), "6281234567890", "/laporan gajian")
	if !strings.Contains(resp, "Laporan Siklus Gajian") {
		t.Errorf("expected /laporan gajian response to contain 'Laporan Siklus Gajian', got %q", resp)
	}

	// 2) Test /laporan total
	respTotal := appRouter.HandleMessage(context.Background(), "6281234567890", "/laporan total")
	if !strings.Contains(respTotal, "Laporan Keseluruhan (All-Time)") {
		t.Errorf("expected /laporan total response to contain 'Laporan Keseluruhan (All-Time)', got %q", respTotal)
	}
}
