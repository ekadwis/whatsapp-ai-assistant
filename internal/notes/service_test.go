package notes

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

type mockSheetRepository struct {
	ensureErr error
	appendErr error

	ensureCalls int
	appendCalls int

	lastEnsuredTab string
	lastNote       *sheets.Note
}

func (m *mockSheetRepository) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error {
	return nil
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
	m.appendCalls++
	m.lastNote = note
	return m.appendErr
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

func TestNotesService_SaveNote_Success(t *testing.T) {
	repo := &mockSheetRepository{}
	svc := NewNotesService(repo)

	err := svc.SaveNote(context.Background(), "  beli kado ultah minggu depan  ")
	if err != nil {
		t.Fatalf("SaveNote() returned error: %v", err)
	}

	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists to be called once, got %d", repo.ensureCalls)
	}
	if repo.lastEnsuredTab != "Notes" {
		t.Fatalf("expected EnsureTabExists called with tab %q, got %q", "Notes", repo.lastEnsuredTab)
	}

	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendNote to be called once, got %d", repo.appendCalls)
	}
	if repo.lastNote == nil {
		t.Fatal("expected note to be passed to AppendNote")
	}
	if repo.lastNote.Content != "beli kado ultah minggu depan" {
		t.Fatalf("expected trimmed note content, got %q", repo.lastNote.Content)
	}
	if repo.lastNote.Date.IsZero() {
		t.Fatal("expected note date to be set")
	}
}

func TestNotesService_SaveNote_EmptyValidation(t *testing.T) {
	repo := &mockSheetRepository{}
	svc := NewNotesService(repo)

	cases := []string{"", "   ", "\n\t "}
	for _, input := range cases {
		err := svc.SaveNote(context.Background(), input)
		if err == nil {
			t.Fatalf("expected error for input %q, got nil", input)
		}
		if !strings.Contains(err.Error(), "catatan tidak boleh kosong") {
			t.Fatalf("unexpected error for input %q: %v", input, err)
		}
	}

	if repo.ensureCalls != 0 {
		t.Fatalf("expected EnsureTabExists not called for empty note, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 0 {
		t.Fatalf("expected AppendNote not called for empty note, got %d", repo.appendCalls)
	}
}

func TestNotesService_SaveNote_EnsureTabError(t *testing.T) {
	repo := &mockSheetRepository{
		ensureErr: errors.New("ensure tab failed"),
	}
	svc := NewNotesService(repo)

	err := svc.SaveNote(context.Background(), "catatan")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to ensure Notes tab") {
		t.Fatalf("expected wrapped ensure error, got: %v", err)
	}

	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 0 {
		t.Fatalf("expected AppendNote not called when EnsureTabExists fails, got %d", repo.appendCalls)
	}
}

func TestNotesService_SaveNote_AppendError(t *testing.T) {
	repo := &mockSheetRepository{
		appendErr: errors.New("append failed"),
	}
	svc := NewNotesService(repo)

	err := svc.SaveNote(context.Background(), "catatan test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to append note") {
		t.Fatalf("expected wrapped append error, got: %v", err)
	}

	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendNote called once, got %d", repo.appendCalls)
	}
}
