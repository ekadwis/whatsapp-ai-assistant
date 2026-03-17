package reminder

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

type mockRepo struct {
	appendedReminder *sheets.Reminder
	appendErr        error

	getByIDReminder *sheets.Reminder
	getByIDRow      int
	getByIDErr      error

	updatedReminder *sheets.Reminder
	updatedRow      int
	updateErr       error
}

func (m *mockRepo) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error { return nil }
func (m *mockRepo) GetTransactions(ctx context.Context, tabName string) ([]sheets.Transaction, error) {
	return nil, nil
}
func (m *mockRepo) GetTransactionByID(ctx context.Context, id string) (*sheets.Transaction, int, string, error) {
	return nil, 0, "", nil
}
func (m *mockRepo) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *sheets.Transaction) error {
	return nil
}
func (m *mockRepo) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error { return nil }
func (m *mockRepo) AppendNote(ctx context.Context, note *sheets.Note) error                    { return nil }
func (m *mockRepo) GetBudget(ctx context.Context, category string) (float64, error)            { return 0, nil }
func (m *mockRepo) SetBudget(ctx context.Context, category string, amount float64) error        { return nil }
func (m *mockRepo) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	return 0, nil
}
func (m *mockRepo) EnsureTabExists(ctx context.Context, tabName string) error { return nil }
func (m *mockRepo) FormatHeaders(ctx context.Context, tabName string) error   { return nil }
func (m *mockRepo) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	return nil
}
func (m *mockRepo) InitDashboard(ctx context.Context) error    { return nil }
func (m *mockRepo) InitBudgetTab(ctx context.Context) error    { return nil }
func (m *mockRepo) InitNotesTab(ctx context.Context) error     { return nil }
func (m *mockRepo) InitReminderTab(ctx context.Context) error  { return nil }
func (m *mockRepo) ListActiveReminders(ctx context.Context) ([]sheets.Reminder, error) {
	return nil, nil
}
func (m *mockRepo) ListDueReminders(ctx context.Context, now time.Time) ([]sheets.Reminder, error) {
	return nil, nil
}

func (m *mockRepo) AppendReminder(ctx context.Context, reminder *sheets.Reminder) error {
	m.appendedReminder = reminder
	return m.appendErr
}
func (m *mockRepo) GetReminderByID(ctx context.Context, id string) (*sheets.Reminder, int, error) {
	return m.getByIDReminder, m.getByIDRow, m.getByIDErr
}
func (m *mockRepo) UpdateReminder(ctx context.Context, rowIndex int, reminder *sheets.Reminder) error {
	m.updatedRow = rowIndex
	m.updatedReminder = reminder
	return m.updateErr
}

type mockNotifier struct{}

func (m *mockNotifier) SendText(ctx context.Context, recipient string, text string) error { return nil }

func TestParseReminderText_NoTime_DefaultRecurring(t *testing.T) {
	now := time.Date(2026, 3, 17, 22, 0, 0, 0, sheets.WIB)

	parsed, err := ParseReminderText("ingetin dong tgl 26 maret bayar vps tahunan contabo", now)
	if err != nil {
		t.Fatalf("ParseReminderText() error: %v", err)
	}

	if parsed.TargetDate.Format("2006-01-02") != "2026-03-26" {
		t.Fatalf("unexpected target date: %s", parsed.TargetDate.Format("2006-01-02"))
	}
	if parsed.TargetTime != "" {
		t.Fatalf("expected empty target time, got %q", parsed.TargetTime)
	}
	if parsed.Mode != sheets.ReminderModeUntilDone {
		t.Fatalf("expected mode %q, got %q", sheets.ReminderModeUntilDone, parsed.Mode)
	}
	if parsed.RemindersPerDay != 3 {
		t.Fatalf("expected 3 reminders/day, got %d", parsed.RemindersPerDay)
	}
	if !strings.Contains(strings.ToLower(parsed.Message), "bayar vps") {
		t.Fatalf("unexpected parsed message: %q", parsed.Message)
	}
}

func TestParseReminderText_WithTime_DefaultOnce(t *testing.T) {
	now := time.Date(2026, 3, 17, 22, 0, 0, 0, sheets.WIB)

	parsed, err := ParseReminderText("ingatkan tanggal 26 maret jam 09:30 bayar vps contabo", now)
	if err != nil {
		t.Fatalf("ParseReminderText() error: %v", err)
	}

	if parsed.TargetDate.Format("2006-01-02") != "2026-03-26" {
		t.Fatalf("unexpected target date: %s", parsed.TargetDate.Format("2006-01-02"))
	}
	if parsed.TargetTime != "09:30" {
		t.Fatalf("expected target time 09:30, got %q", parsed.TargetTime)
	}
	if parsed.Mode != sheets.ReminderModeOnce {
		t.Fatalf("expected mode %q, got %q", sheets.ReminderModeOnce, parsed.Mode)
	}
	if parsed.RemindersPerDay != 1 {
		t.Fatalf("expected reminders/day 1, got %d", parsed.RemindersPerDay)
	}
}

func TestCreateFromText_Defaults(t *testing.T) {
	repo := &mockRepo{}
	svc := NewService(repo, &mockNotifier{}, "6281234567890")

	fixedNow := time.Date(2026, 3, 17, 23, 0, 0, 0, sheets.WIB)
	svc.now = func() time.Time { return fixedNow }

	rem, err := svc.CreateFromText(context.Background(), "ingetin dong tgl 26 maret bayar vps contabo")
	if err != nil {
		t.Fatalf("CreateFromText() error: %v", err)
	}
	if rem == nil {
		t.Fatal("expected non-nil reminder")
	}
	if repo.appendedReminder == nil {
		t.Fatal("expected reminder appended to repo")
	}

	if rem.Mode != sheets.ReminderModeUntilDone {
		t.Fatalf("expected mode until_done, got %q", rem.Mode)
	}
	if rem.RemindersPerDay != 3 {
		t.Fatalf("expected reminders/day 3, got %d", rem.RemindersPerDay)
	}
	if rem.Status != sheets.ReminderStatusActive {
		t.Fatalf("expected status active, got %q", rem.Status)
	}
	if rem.ID == "" {
		t.Fatal("expected generated reminder ID")
	}
}

func TestCompleteByID_MarksCompleted(t *testing.T) {
	existing := &sheets.Reminder{
		ID:              "RMD-TEST-001",
		Message:         "Bayar VPS",
		TargetDate:      time.Date(2026, 3, 26, 0, 0, 0, 0, sheets.WIB),
		Mode:            sheets.ReminderModeUntilDone,
		RemindersPerDay: 3,
		Status:          sheets.ReminderStatusActive,
		CreatedAt:       time.Date(2026, 3, 17, 9, 0, 0, 0, sheets.WIB),
		UpdatedAt:       time.Date(2026, 3, 17, 9, 0, 0, 0, sheets.WIB),
	}
	repo := &mockRepo{
		getByIDReminder: existing,
		getByIDRow:      7,
	}
	svc := NewService(repo, &mockNotifier{}, "6281234567890")
	fixedNow := time.Date(2026, 3, 26, 10, 15, 0, 0, sheets.WIB)
	svc.now = func() time.Time { return fixedNow }

	rem, err := svc.CompleteByID(context.Background(), "RMD-TEST-001", "")
	if err != nil {
		t.Fatalf("CompleteByID() error: %v", err)
	}
	if rem == nil {
		t.Fatal("expected non-nil reminder")
	}
	if rem.Status != sheets.ReminderStatusCompleted {
		t.Fatalf("expected completed status, got %q", rem.Status)
	}
	if rem.CompletedAt == nil {
		t.Fatal("expected CompletedAt to be set")
	}
	if repo.updatedReminder == nil {
		t.Fatal("expected reminder to be updated")
	}
	if repo.updatedRow != 7 {
		t.Fatalf("expected row index 7, got %d", repo.updatedRow)
	}
}
