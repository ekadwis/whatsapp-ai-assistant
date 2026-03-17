package finance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	aiinternal "github.com/verssache/whatsapp-ai-assistant/internal/ai"
	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

type mockSheetRepository struct {
	ensureErr error
	appendErr error

	ensureCalls int
	appendCalls int

	lastEnsuredTab string
	lastAppendedTx *sheets.Transaction

	callOrder []string
}

func (m *mockSheetRepository) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error {
	m.appendCalls++
	m.callOrder = append(m.callOrder, "append")
	m.lastAppendedTx = tx
	return m.appendErr
}

func (m *mockSheetRepository) GetTransactions(ctx context.Context, tabName string) ([]sheets.Transaction, error) {
	return nil, nil
}

func (m *mockSheetRepository) GetTransactionByID(ctx context.Context, id string) (*sheets.Transaction, int, string, error) {
	return nil, 0, "", nil
}

func (m *mockSheetRepository) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *sheets.Transaction) error {
	return nil
}

func (m *mockSheetRepository) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error {
	return nil
}

func (m *mockSheetRepository) AppendNote(ctx context.Context, note *sheets.Note) error {
	return nil
}

func (m *mockSheetRepository) GetBudget(ctx context.Context, category string) (float64, error) {
	return 0, nil
}

func (m *mockSheetRepository) SetBudget(ctx context.Context, category string, amount float64) error {
	return nil
}

func (m *mockSheetRepository) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	return 0, nil
}

func (m *mockSheetRepository) EnsureTabExists(ctx context.Context, tabName string) error {
	m.ensureCalls++
	m.callOrder = append(m.callOrder, "ensure")
	m.lastEnsuredTab = tabName
	return m.ensureErr
}

func (m *mockSheetRepository) FormatHeaders(ctx context.Context, tabName string) error {
	return nil
}

func (m *mockSheetRepository) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	return nil
}

func (m *mockSheetRepository) InitDashboard(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepository) InitBudgetTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepository) InitNotesTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepository) InitReminderTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepository) AppendReminder(ctx context.Context, reminder *sheets.Reminder) error {
	return nil
}

func (m *mockSheetRepository) ListActiveReminders(ctx context.Context) ([]sheets.Reminder, error) {
	return nil, nil
}

func (m *mockSheetRepository) GetReminderByID(ctx context.Context, id string) (*sheets.Reminder, int, error) {
	return nil, 0, nil
}

func (m *mockSheetRepository) UpdateReminder(ctx context.Context, rowIndex int, reminder *sheets.Reminder) error {
	return nil
}

func (m *mockSheetRepository) ListDueReminders(ctx context.Context, now time.Time) ([]sheets.Reminder, error) {
	return nil, nil
}

func TestRecordTransaction_Expense_Success(t *testing.T) {
	repo := &mockSheetRepository{}
	svc := NewFinanceService(repo, nil)

	args := &aiinternal.RecordTransactionArgs{
		Type:        "expense",
		Amount:      16000,
		Category:    "Makanan",
		Description: "ayam crispy",
	}

	tx, err := svc.RecordTransaction(context.Background(), args)
	if err != nil {
		t.Fatalf("RecordTransaction returned error: %v", err)
	}
	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists to be called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendTransaction to be called once, got %d", repo.appendCalls)
	}
	if repo.lastEnsuredTab == "" {
		t.Fatal("expected ensured tab name to be non-empty")
	}
	if len(repo.callOrder) < 2 || repo.callOrder[0] != "ensure" || repo.callOrder[1] != "append" {
		t.Fatalf("expected call order ensure -> append, got %v", repo.callOrder)
	}
	if repo.lastAppendedTx == nil {
		t.Fatal("expected appended transaction to be captured")
	}

	if tx.Type != sheets.Expense {
		t.Fatalf("expected transaction type %q, got %q", sheets.Expense, tx.Type)
	}
	if tx.Category != "Makanan" {
		t.Fatalf("expected category %q, got %q", "Makanan", tx.Category)
	}
	if tx.Description != "ayam crispy" {
		t.Fatalf("expected description %q, got %q", "ayam crispy", tx.Description)
	}
	if tx.Amount != 16000 {
		t.Fatalf("expected amount %v, got %v", 16000.0, tx.Amount)
	}
	if tx.ID == "" {
		t.Fatal("expected generated transaction ID to be non-empty")
	}
	if tx.Date.IsZero() {
		t.Fatal("expected transaction Date to be set")
	}

	// Ensure the exact object returned was appended.
	if repo.lastAppendedTx != tx {
		t.Fatal("expected appended transaction pointer to match returned transaction")
	}
}

func TestRecordTransaction_Income_Success(t *testing.T) {
	repo := &mockSheetRepository{}
	svc := NewFinanceService(repo, nil)

	args := &aiinternal.RecordTransactionArgs{
		Type:        "income",
		Amount:      5000000,
		Category:    "Gaji",
		Description: "gaji bulanan",
	}

	tx, err := svc.RecordTransaction(context.Background(), args)
	if err != nil {
		t.Fatalf("RecordTransaction returned error: %v", err)
	}
	if tx == nil {
		t.Fatal("expected non-nil transaction")
	}

	if tx.Type != sheets.Income {
		t.Fatalf("expected transaction type %q, got %q", sheets.Income, tx.Type)
	}
	if tx.Category != "Gaji" {
		t.Fatalf("expected category %q, got %q", "Gaji", tx.Category)
	}
	if tx.Description != "gaji bulanan" {
		t.Fatalf("expected description %q, got %q", "gaji bulanan", tx.Description)
	}
	if tx.Amount != 5000000 {
		t.Fatalf("expected amount %v, got %v", 5000000.0, tx.Amount)
	}
}

func TestRecordTransaction_EnsureTabError(t *testing.T) {
	repo := &mockSheetRepository{
		ensureErr: errors.New("ensure failed"),
	}
	svc := NewFinanceService(repo, nil)

	args := &aiinternal.RecordTransactionArgs{
		Type:        "expense",
		Amount:      10000,
		Category:    "Makanan",
		Description: "test",
	}

	tx, err := svc.RecordTransaction(context.Background(), args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if tx != nil {
		t.Fatal("expected nil transaction on ensure tab error")
	}
	if !strings.Contains(err.Error(), "failed to ensure tab") {
		t.Fatalf("expected wrapped ensure-tab error, got: %v", err)
	}
	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 0 {
		t.Fatalf("expected AppendTransaction not called when ensure tab fails, got %d", repo.appendCalls)
	}
}

func TestRecordTransaction_AppendError(t *testing.T) {
	repo := &mockSheetRepository{
		appendErr: errors.New("append failed"),
	}
	svc := NewFinanceService(repo, nil)

	args := &aiinternal.RecordTransactionArgs{
		Type:        "expense",
		Amount:      22000,
		Category:    "Transportasi",
		Description: "ojek",
	}

	tx, err := svc.RecordTransaction(context.Background(), args)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if tx != nil {
		t.Fatal("expected nil transaction on append error")
	}
	if !strings.Contains(err.Error(), "failed to append transaction") {
		t.Fatalf("expected wrapped append error, got: %v", err)
	}
	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendTransaction called once, got %d", repo.appendCalls)
	}
}

type mockSheetRepositoryAdvanced struct {
	transactions      []sheets.Transaction
	getTransactionsErr error

	getBudgetValue      float64
	getBudgetErr        error
	categoryTotalValue  float64
	getCategoryTotalErr error

	setBudgetErr   error
	setBudgetCalls int
	lastSetCategory string
	lastSetAmount   float64

	getTxByIDTx    *sheets.Transaction
	getTxByIDRow   int
	getTxByIDTab   string
	getTxByIDErr   error

	updateErr      error
	updateCalls    int
	lastUpdateTab  string
	lastUpdateRow  int
	lastUpdatedTx  *sheets.Transaction

	deleteErr      error
	deleteCalls    int
	lastDeleteTab  string
	lastDeleteRow  int
}

func (m *mockSheetRepositoryAdvanced) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) GetTransactions(ctx context.Context, tabName string) ([]sheets.Transaction, error) {
	if m.getTransactionsErr != nil {
		return nil, m.getTransactionsErr
	}
	return m.transactions, nil
}

func (m *mockSheetRepositoryAdvanced) GetTransactionByID(ctx context.Context, id string) (*sheets.Transaction, int, string, error) {
	if m.getTxByIDErr != nil {
		return nil, 0, "", m.getTxByIDErr
	}
	return m.getTxByIDTx, m.getTxByIDRow, m.getTxByIDTab, nil
}

func (m *mockSheetRepositoryAdvanced) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *sheets.Transaction) error {
	m.updateCalls++
	m.lastUpdateTab = tabName
	m.lastUpdateRow = rowIndex
	m.lastUpdatedTx = tx
	return m.updateErr
}

func (m *mockSheetRepositoryAdvanced) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error {
	m.deleteCalls++
	m.lastDeleteTab = tabName
	m.lastDeleteRow = rowIndex
	return m.deleteErr
}

func (m *mockSheetRepositoryAdvanced) AppendNote(ctx context.Context, note *sheets.Note) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) GetBudget(ctx context.Context, category string) (float64, error) {
	if m.getBudgetErr != nil {
		return 0, m.getBudgetErr
	}
	return m.getBudgetValue, nil
}

func (m *mockSheetRepositoryAdvanced) SetBudget(ctx context.Context, category string, amount float64) error {
	m.setBudgetCalls++
	m.lastSetCategory = category
	m.lastSetAmount = amount
	return m.setBudgetErr
}

func (m *mockSheetRepositoryAdvanced) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	if m.getCategoryTotalErr != nil {
		return 0, m.getCategoryTotalErr
	}
	return m.categoryTotalValue, nil
}

func (m *mockSheetRepositoryAdvanced) EnsureTabExists(ctx context.Context, tabName string) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) FormatHeaders(ctx context.Context, tabName string) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) InitDashboard(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) InitBudgetTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) InitNotesTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) InitReminderTab(ctx context.Context) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) AppendReminder(ctx context.Context, reminder *sheets.Reminder) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) ListActiveReminders(ctx context.Context) ([]sheets.Reminder, error) {
	return nil, nil
}

func (m *mockSheetRepositoryAdvanced) GetReminderByID(ctx context.Context, id string) (*sheets.Reminder, int, error) {
	return nil, 0, nil
}

func (m *mockSheetRepositoryAdvanced) UpdateReminder(ctx context.Context, rowIndex int, reminder *sheets.Reminder) error {
	return nil
}

func (m *mockSheetRepositoryAdvanced) ListDueReminders(ctx context.Context, now time.Time) ([]sheets.Reminder, error) {
	return nil, nil
}

func TestGenerateReport_Periods(t *testing.T) {
	now := nowWIB()
	repo := &mockSheetRepositoryAdvanced{
		transactions: []sheets.Transaction{
			{
				ID:          "1",
				Date:        now,
				Type:        sheets.Expense,
				Category:    "Makanan",
				Description: "today expense",
				Amount:      10000,
			},
			{
				ID:          "2",
				Date:        now,
				Type:        sheets.Income,
				Category:    "Gaji",
				Description: "today income",
				Amount:      20000,
			},
			{
				ID:          "3",
				Date:        now.AddDate(0, 0, -3),
				Type:        sheets.Expense,
				Category:    "Transportasi",
				Description: "week expense",
				Amount:      5000,
			},
			{
				ID:          "4",
				Date:        now.AddDate(0, 0, -10),
				Type:        sheets.Expense,
				Category:    "Belanja",
				Description: "old expense",
				Amount:      7000,
			},
		},
	}
	svc := NewFinanceService(repo, nil)

	daily, err := svc.GenerateReport(context.Background(), "hari ini")
	if err != nil {
		t.Fatalf("daily report unexpected error: %v", err)
	}
	if daily.TotalIncome != 20000 {
		t.Fatalf("daily income = %v, want %v", daily.TotalIncome, 20000.0)
	}
	if daily.TotalExpense != 10000 {
		t.Fatalf("daily expense = %v, want %v", daily.TotalExpense, 10000.0)
	}
	if daily.Categories["Makanan"] != 10000 {
		t.Fatalf("daily category Makanan = %v, want %v", daily.Categories["Makanan"], 10000.0)
	}

	weekly, err := svc.GenerateReport(context.Background(), "minggu ini")
	if err != nil {
		t.Fatalf("weekly report unexpected error: %v", err)
	}
	if weekly.TotalIncome != 20000 {
		t.Fatalf("weekly income = %v, want %v", weekly.TotalIncome, 20000.0)
	}
	if weekly.TotalExpense != 15000 {
		t.Fatalf("weekly expense = %v, want %v", weekly.TotalExpense, 15000.0)
	}
	if weekly.Categories["Transportasi"] != 5000 {
		t.Fatalf("weekly category Transportasi = %v, want %v", weekly.Categories["Transportasi"], 5000.0)
	}

	monthly, err := svc.GenerateReport(context.Background(), "bulan ini")
	if err != nil {
		t.Fatalf("monthly report unexpected error: %v", err)
	}
	if monthly.TotalIncome != 20000 {
		t.Fatalf("monthly income = %v, want %v", monthly.TotalIncome, 20000.0)
	}
	if monthly.TotalExpense != 22000 {
		t.Fatalf("monthly expense = %v, want %v", monthly.TotalExpense, 22000.0)
	}
}

func TestGenerateReport_UnknownPeriod(t *testing.T) {
	repo := &mockSheetRepositoryAdvanced{}
	svc := NewFinanceService(repo, nil)

	_, err := svc.GenerateReport(context.Background(), "kuartal ini")
	if err == nil {
		t.Fatal("expected error for unknown period")
	}
	if !strings.Contains(err.Error(), "periode tidak dikenal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBudgetMethods(t *testing.T) {
	repo := &mockSheetRepositoryAdvanced{}
	svc := NewFinanceService(repo, nil)

	if err := svc.SetBudget(context.Background(), "Makanan", 500000); err != nil {
		t.Fatalf("SetBudget unexpected error: %v", err)
	}
	if repo.setBudgetCalls != 1 {
		t.Fatalf("expected SetBudget call count 1, got %d", repo.setBudgetCalls)
	}
	if repo.lastSetCategory != "Makanan" || repo.lastSetAmount != 500000 {
		t.Fatalf("unexpected set budget payload: category=%q amount=%v", repo.lastSetCategory, repo.lastSetAmount)
	}

	repo.getBudgetValue = 0
	alert, err := svc.CheckBudget(context.Background(), "Makanan")
	if err != nil {
		t.Fatalf("CheckBudget unexpected error: %v", err)
	}
	if alert != "" {
		t.Fatalf("expected empty alert when budget not set, got %q", alert)
	}

	repo.getBudgetValue = 100000
	repo.categoryTotalValue = 85000
	alert, err = svc.CheckBudget(context.Background(), "Makanan")
	if err != nil {
		t.Fatalf("CheckBudget warning case error: %v", err)
	}
	if !strings.Contains(alert, "Peringatan Budget") {
		t.Fatalf("expected warning alert, got %q", alert)
	}

	repo.getBudgetValue = 100000
	repo.categoryTotalValue = 120000
	alert, err = svc.CheckBudget(context.Background(), "Makanan")
	if err != nil {
		t.Fatalf("CheckBudget over case error: %v", err)
	}
	if !strings.Contains(alert, "Peringatan Budget") {
		t.Fatalf("expected over-budget alert, got %q", alert)
	}
}

func TestEditAndDeleteTransaction(t *testing.T) {
	repo := &mockSheetRepositoryAdvanced{
		getTxByIDTx: &sheets.Transaction{
			ID:          "20260317-001",
			Date:        time.Date(2026, 3, 17, 10, 0, 0, 0, sheets.WIB),
			Type:        sheets.Expense,
			Category:    "Makanan",
			Description: "ayam",
			Amount:      10000,
		},
		getTxByIDRow: 5,
		getTxByIDTab: "Maret 2026",
	}
	svc := NewFinanceService(repo, nil)

	updated, err := svc.EditTransaction(context.Background(), "20260317-001", "jumlah", "20k")
	if err != nil {
		t.Fatalf("EditTransaction unexpected error: %v", err)
	}
	if updated.Amount != 20000 {
		t.Fatalf("expected updated amount 20000, got %v", updated.Amount)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected UpdateTransaction call count 1, got %d", repo.updateCalls)
	}
	if repo.lastUpdateTab != "Maret 2026" || repo.lastUpdateRow != 5 {
		t.Fatalf("unexpected update target: tab=%q row=%d", repo.lastUpdateTab, repo.lastUpdateRow)
	}

	_, err = svc.EditTransaction(context.Background(), "20260317-001", "unknown", "x")
	if err == nil {
		t.Fatal("expected error for unknown edit field")
	}

	if err := svc.DeleteTransaction(context.Background(), "20260317-001"); err != nil {
		t.Fatalf("DeleteTransaction unexpected error: %v", err)
	}
	if repo.deleteCalls != 1 {
		t.Fatalf("expected DeleteTransaction call count 1, got %d", repo.deleteCalls)
	}
	if repo.lastDeleteTab != "Maret 2026" || repo.lastDeleteRow != 5 {
		t.Fatalf("unexpected delete target: tab=%q row=%d", repo.lastDeleteTab, repo.lastDeleteRow)
	}
}

func TestNormalizeCategoryForType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		txType  sheets.TransactionType
		want    string
	}{
		{
			name:   "expense alias makanan dan minuman",
			input:  "Makanan & Minuman",
			txType: sheets.Expense,
			want:   "Makanan",
		},
		{
			name:   "expense canonical case-insensitive",
			input:  "transportasi",
			txType: sheets.Expense,
			want:   "Transportasi",
		},
		{
			name:   "expense unknown fallback",
			input:  "Kategori Bebas",
			txType: sheets.Expense,
			want:   "Lainnya",
		},
		{
			name:   "income canonical case-insensitive",
			input:  "freelance",
			txType: sheets.Income,
			want:   "Freelance",
		},
		{
			name:   "income unknown fallback",
			input:  "Makanan",
			txType: sheets.Income,
			want:   "Lainnya",
		},
		{
			name:   "empty fallback",
			input:  "   ",
			txType: sheets.Expense,
			want:   "Lainnya",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeCategoryForType(tc.input, tc.txType)
			if got != tc.want {
				t.Fatalf("normalizeCategoryForType(%q, %q) = %q, want %q", tc.input, tc.txType, got, tc.want)
			}
		})
	}
}

func TestNextTransactionIDFromSheet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 17, 9, 0, 0, 0, sheets.WIB)

	t.Run("increments from max existing counter for same date", func(t *testing.T) {
		repo := &mockSheetRepositoryAdvanced{
			transactions: []sheets.Transaction{
				{ID: "20260317-001"},
				{ID: "20260317-010"},
				{ID: "20260316-999"}, // different date, ignored
				{ID: "invalid-id"},   // malformed, ignored
			},
		}
		svc := NewFinanceService(repo, nil)

		got, err := svc.nextTransactionIDFromSheet(context.Background(), now)
		if err != nil {
			t.Fatalf("nextTransactionIDFromSheet returned error: %v", err)
		}
		if got != "20260317-011" {
			t.Fatalf("expected ID %q, got %q", "20260317-011", got)
		}
	})

	t.Run("starts at 001 when no existing transaction for date", func(t *testing.T) {
		repo := &mockSheetRepositoryAdvanced{
			transactions: []sheets.Transaction{
				{ID: "20260316-003"},
			},
		}
		svc := NewFinanceService(repo, nil)

		got, err := svc.nextTransactionIDFromSheet(context.Background(), now)
		if err != nil {
			t.Fatalf("nextTransactionIDFromSheet returned error: %v", err)
		}
		if got != "20260317-001" {
			t.Fatalf("expected ID %q, got %q", "20260317-001", got)
		}
	})
}
