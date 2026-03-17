package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/ai"
	"github.com/verssache/whatsapp-ai-assistant/internal/commands"
	"github.com/verssache/whatsapp-ai-assistant/internal/finance"
	"github.com/verssache/whatsapp-ai-assistant/internal/notes"
	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

type mockSheetRepo struct {
	ensureCalls int
	appendCalls int
}

func (m *mockSheetRepo) AppendTransaction(ctx context.Context, tx *sheets.Transaction) error {
	m.appendCalls++
	return nil
}
func (m *mockSheetRepo) GetTransactions(ctx context.Context, tabName string) ([]sheets.Transaction, error) {
	return nil, nil
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
	m.ensureCalls++
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

func TestHandleMessage_CommandPriorityOverLLM(t *testing.T) {
	cmd := commands.NewRouter()
	cmd.Register("/help", func(ctx context.Context, args string) string {
		return "help-response"
	})

	// Intentionally invalid LLM endpoint: if command priority is correct, this won't be called.
	llm := ai.NewLLMClient("http://127.0.0.1:1", "x", "m")
	r := NewAppRouter(cmd, llm, nil, nil)

	got := r.HandleMessage(context.Background(), "628123", "/help")
	if got != "help-response" {
		t.Fatalf("expected command response, got %q", got)
	}
}

func TestHandleMessage_LLMTextFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices":[{"message":{"content":"Halo! Saya siap bantu."}}
		]}`))
	}))
	defer srv.Close()

	cmd := commands.NewRouter()
	llm := ai.NewLLMClient(srv.URL, "test-key", "test-model")
	r := NewAppRouter(cmd, llm, nil, nil)

	got := r.HandleMessage(context.Background(), "628123", "halo")
	if got != "Halo! Saya siap bantu." {
		t.Fatalf("expected LLM text response, got %q", got)
	}
}

func TestHandleMessage_ConfirmationYesExecutesPendingTransaction(t *testing.T) {
	cmd := commands.NewRouter()
	repo := &mockSheetRepo{}
	fin := finance.NewFinanceService(repo, nil)
	noteSvc := notes.NewNotesService(repo)

	r := NewAppRouter(cmd, nil, fin, noteSvc)

	sender := "628123"
	r.pendingActions.Store(sender, &PendingAction{
		Transaction: &ai.RecordTransactionArgs{
			Type:        "expense",
			Amount:      16000,
			Category:    "Makanan",
			Description: "ayam crispy",
		},
		CreatedAt: time.Now(),
	})

	got := r.HandleMessage(context.Background(), sender, "ya")
	if !strings.Contains(got, "Pengeluaran Dicatat") {
		t.Fatalf("expected confirmation execution response, got %q", got)
	}
	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendTransaction called once, got %d", repo.appendCalls)
	}
	if _, ok := r.pendingActions.Load(sender); ok {
		t.Fatal("expected pending action to be cleared after confirmation")
	}
}

func TestHandleMessage_ConfirmationCancel(t *testing.T) {
	r := NewAppRouter(commands.NewRouter(), nil, nil, nil)
	sender := "628123"

	r.pendingActions.Store(sender, &PendingAction{
		Transaction: &ai.RecordTransactionArgs{
			Type:        "expense",
			Amount:      10000,
			Category:    "Makanan",
			Description: "tes",
		},
		CreatedAt: time.Now(),
	})

	got := r.HandleMessage(context.Background(), sender, "bukan")
	if !strings.Contains(strings.ToLower(got), "dibatalkan") {
		t.Fatalf("expected cancel response, got %q", got)
	}
	if _, ok := r.pendingActions.Load(sender); ok {
		t.Fatal("expected pending action to be cleared after cancellation")
	}
}

func TestHandleMessage_PendingExpiredThenFallsBackToNormalFlow(t *testing.T) {
	r := NewAppRouter(commands.NewRouter(), nil, nil, nil)
	sender := "628123"

	r.pendingActions.Store(sender, &PendingAction{
		Transaction: &ai.RecordTransactionArgs{
			Type:        "expense",
			Amount:      12000,
			Category:    "Makanan",
			Description: "expired tx",
		},
		CreatedAt: time.Now().Add(-10 * time.Minute),
	})

	got := r.HandleMessage(context.Background(), sender, "ya")
	if !strings.Contains(strings.ToLower(got), "layanan ai belum siap") {
		t.Fatalf("expected fallback error after pending expiry, got %q", got)
	}
	if _, ok := r.pendingActions.Load(sender); ok {
		t.Fatal("expected expired pending action to be cleared")
	}
}

func TestHandleMessage_RecordTransactionToolCallExecutesImmediately(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "",
					"tool_calls": [{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "record_transaction",
							"arguments": "{\"type\":\"expense\",\"amount\":220000,\"category\":\"Transportasi\",\"description\":\"Beli bensin mobil\"}"
						}
					}]
				}
			}]
		}`))
	}))
	defer srv.Close()

	repo := &mockSheetRepo{}
	fin := finance.NewFinanceService(repo, nil)
	noteSvc := notes.NewNotesService(repo)
	llm := ai.NewLLMClient(srv.URL, "test-key", "test-model")

	r := NewAppRouter(commands.NewRouter(), llm, fin, noteSvc)

	sender := "628123"
	got := r.HandleMessage(context.Background(), sender, "beli bensin mobil 220k")

	if !strings.Contains(strings.ToLower(got), "pengeluaran berhasil dicatat") {
		t.Fatalf("expected immediate compact transaction response, got %q", got)
	}
	if repo.ensureCalls != 1 {
		t.Fatalf("expected EnsureTabExists called once, got %d", repo.ensureCalls)
	}
	if repo.appendCalls != 1 {
		t.Fatalf("expected AppendTransaction called once, got %d", repo.appendCalls)
	}
	if _, ok := r.pendingActions.Load(sender); ok {
		t.Fatal("expected no pending confirmation after immediate execution")
	}
}

func TestHandleMessage_MultipleRecordTransactionToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "",
					"tool_calls": [
						{
							"id": "call_1",
							"type": "function",
							"function": {
								"name": "record_transaction",
								"arguments": "{\"type\":\"expense\",\"amount\":10000,\"category\":\"Makanan\",\"description\":\"beli cimol\"}"
							}
						},
						{
							"id": "call_2",
							"type": "function",
							"function": {
								"name": "record_transaction",
								"arguments": "{\"type\":\"expense\",\"amount\":10000,\"category\":\"Makanan\",\"description\":\"beli bakso bakar\"}"
							}
						},
						{
							"id": "call_3",
							"type": "function",
							"function": {
								"name": "record_transaction",
								"arguments": "{\"type\":\"expense\",\"amount\":24000,\"category\":\"Belanja\",\"description\":\"belanja sayur\"}"
							}
						}
					]
				}
			}]
		}`))
	}))
	defer srv.Close()

	repo := &mockSheetRepo{}
	fin := finance.NewFinanceService(repo, nil)
	noteSvc := notes.NewNotesService(repo)
	llm := ai.NewLLMClient(srv.URL, "test-key", "test-model")

	r := NewAppRouter(commands.NewRouter(), llm, fin, noteSvc)

	got := r.HandleMessage(context.Background(), "628123", "beli cimol 10k, beli bakso bakar 10k, belanja sayur 24k")

	if repo.appendCalls != 3 {
		t.Fatalf("expected 3 transactions appended, got %d", repo.appendCalls)
	}
	if repo.ensureCalls != 3 {
		t.Fatalf("expected EnsureTabExists called 3 times, got %d", repo.ensureCalls)
	}

	if !strings.Contains(got, "beli cimol") {
		t.Fatalf("expected response to include first transaction, got %q", got)
	}
	if !strings.Contains(got, "beli bakso bakar") {
		t.Fatalf("expected response to include second transaction, got %q", got)
	}
	if !strings.Contains(got, "belanja sayur") {
		t.Fatalf("expected response to include third transaction, got %q", got)
	}
}
