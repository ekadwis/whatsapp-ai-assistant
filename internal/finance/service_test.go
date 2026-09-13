package finance_test

import (
	"context"
	"strings"
	"testing"
	"time"

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

func TestGenerateAllTimeReport(t *testing.T) {
	d1 := time.Date(2026, 1, 15, 10, 0, 0, 0, sheets.WIB)
	d2 := time.Date(2026, 2, 20, 12, 0, 0, 0, sheets.WIB)
	d3 := time.Date(2026, 3, 5, 14, 0, 0, 0, sheets.WIB)

	repo := &mockSheetRepo{
		allTxs: []sheets.Transaction{
			{ID: "20260115-001", Date: d1, Type: sheets.Income, Amount: 10000000, Category: "Gaji", Description: "Gaji Januari"},
			{ID: "20260220-001", Date: d2, Type: sheets.Expense, Amount: 2000000, Category: "Makanan", Description: "Makan & Minum"},
			{ID: "20260305-001", Date: d3, Type: sheets.Expense, Amount: 1500000, Category: "Belanja", Description: "Beli baju"},
		},
	}

	service := finance.NewFinanceService(repo, nil)
	report, err := service.GenerateAllTimeReport(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.TotalIncome != 10000000 {
		t.Errorf("expected TotalIncome 10000000, got %v", report.TotalIncome)
	}
	if report.TotalExpense != 3500000 {
		t.Errorf("expected TotalExpense 3500000, got %v", report.TotalExpense)
	}
	if report.NetBalance != 6500000 {
		t.Errorf("expected NetBalance 6500000, got %v", report.NetBalance)
	}
	if report.TotalCount != 3 {
		t.Errorf("expected TotalCount 3, got %d", report.TotalCount)
	}
}

func TestGenerateSalaryCycleReport(t *testing.T) {
	d1 := time.Date(2026, 2, 25, 10, 0, 0, 0, sheets.WIB)
	d2 := time.Date(2026, 3, 2, 12, 0, 0, 0, sheets.WIB)

	repo := &mockSheetRepo{
		rangeTxs: []sheets.Transaction{
			{ID: "20260225-001", Date: d1, Type: sheets.Income, Amount: 7500000, Category: "Gaji", Description: "Gaji Februari"},
			{ID: "20260302-001", Date: d2, Type: sheets.Expense, Amount: 500000, Category: "Makanan", Description: "Makan Resto"},
		},
	}

	service := finance.NewFinanceService(repo, nil)
	report, err := service.GenerateSalaryCycleReport(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Period != "gajian" {
		t.Errorf("expected period gajian, got %s", report.Period)
	}
	if report.TotalIncome != 7500000 {
		t.Errorf("expected TotalIncome 7500000, got %v", report.TotalIncome)
	}
	if report.TotalExpense != 500000 {
		t.Errorf("expected TotalExpense 500000, got %v", report.TotalExpense)
	}
	if report.NetBalance != 7000000 {
		t.Errorf("expected NetBalance 7000000, got %v", report.NetBalance)
	}
	if !strings.Contains(report.DateRange, "23") {
		t.Errorf("expected DateRange to start from 23, got %s", report.DateRange)
	}
}
