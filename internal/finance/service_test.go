package finance

import (
	"context"
	"errors"
	"strings"
	"testing"

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
