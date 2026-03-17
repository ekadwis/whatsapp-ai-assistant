package sheets

import (
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestIDGenerator_Next_SequenceAndReset(t *testing.T) {
	gen := &IDGenerator{}

	day1 := time.Date(2026, 3, 17, 10, 0, 0, 0, WIB)
	day2 := time.Date(2026, 3, 18, 9, 30, 0, 0, WIB)

	id1 := gen.Next(day1)
	id2 := gen.Next(day1)
	id3 := gen.Next(day2)

	if id1 != "20260317-001" {
		t.Fatalf("expected first id %q, got %q", "20260317-001", id1)
	}
	if id2 != "20260317-002" {
		t.Fatalf("expected second id %q, got %q", "20260317-002", id2)
	}
	if id3 != "20260318-001" {
		t.Fatalf("expected reset id %q, got %q", "20260318-001", id3)
	}
}

func TestGenerateTransactionID_Format(t *testing.T) {
	re := regexp.MustCompile(`^\d{8}-\d{3}$`)
	id := GenerateTransactionID(time.Now().In(WIB))
	if !re.MatchString(id) {
		t.Fatalf("expected id %q to match format YYYYMMDD-NNN", id)
	}
}

func TestTransaction_ToRow(t *testing.T) {
	ts := time.Date(2026, 3, 17, 14, 30, 0, 0, WIB)
	tx := &Transaction{
		ID:          "20260317-001",
		Date:        ts,
		Type:        Expense,
		Category:    "Makanan",
		Description: "ayam crispy",
		Amount:      16000,
	}

	row := tx.ToRow()
	if len(row) != 7 {
		t.Fatalf("expected 7 columns, got %d", len(row))
	}

	expect := []interface{}{
		"20260317-001",
		"17/03/2026",
		"14:30",
		"Pengeluaran",
		"Makanan",
		"ayam crispy",
		float64(16000),
	}

	for i := range expect {
		if fmt.Sprint(row[i]) != fmt.Sprint(expect[i]) {
			t.Fatalf("unexpected row[%d]: got %v, want %v", i, row[i], expect[i])
		}
	}
}

func TestTransactionFromRow_RoundTrip(t *testing.T) {
	original := &Transaction{
		ID:          "20260317-001",
		Date:        time.Date(2026, 3, 17, 14, 30, 0, 0, WIB),
		Type:        Income,
		Category:    "Gaji",
		Description: "gaji bulan maret",
		Amount:      5000000,
	}

	row := original.ToRow()
	got, err := TransactionFromRow(row)
	if err != nil {
		t.Fatalf("TransactionFromRow returned error: %v", err)
	}

	if got.ID != original.ID {
		t.Fatalf("ID mismatch: got %q, want %q", got.ID, original.ID)
	}
	if got.Type != original.Type {
		t.Fatalf("Type mismatch: got %q, want %q", got.Type, original.Type)
	}
	if got.Category != original.Category {
		t.Fatalf("Category mismatch: got %q, want %q", got.Category, original.Category)
	}
	if got.Description != original.Description {
		t.Fatalf("Description mismatch: got %q, want %q", got.Description, original.Description)
	}
	if got.Amount != original.Amount {
		t.Fatalf("Amount mismatch: got %v, want %v", got.Amount, original.Amount)
	}
	if got.Date.In(WIB).Format("02/01/2006 15:04") != original.Date.In(WIB).Format("02/01/2006 15:04") {
		t.Fatalf("Date mismatch: got %s, want %s", got.Date.In(WIB).Format("02/01/2006 15:04"), original.Date.In(WIB).Format("02/01/2006 15:04"))
	}
}

func TestTransactionFromRow_InvalidCases(t *testing.T) {
	t.Run("too few columns", func(t *testing.T) {
		_, err := TransactionFromRow([]interface{}{"only", "two"})
		if err == nil {
			t.Fatal("expected error for short row, got nil")
		}
	})

	t.Run("invalid amount", func(t *testing.T) {
		row := []interface{}{
			"20260317-001",
			"17/03/2026",
			"14:30",
			"Pengeluaran",
			"Makanan",
			"ayam",
			"abc",
		}
		_, err := TransactionFromRow(row)
		if err == nil {
			t.Fatal("expected error for invalid amount, got nil")
		}
	})

	t.Run("invalid date/time", func(t *testing.T) {
		row := []interface{}{
			"20260317-001",
			"31/02/2026",
			"25:99",
			"Pengeluaran",
			"Makanan",
			"ayam",
			16000,
		}
		_, err := TransactionFromRow(row)
		if err == nil {
			t.Fatal("expected error for invalid date/time, got nil")
		}
	})
}

func TestIDGenerator_ConcurrentUnique(t *testing.T) {
	gen := &IDGenerator{}
	when := time.Date(2026, 3, 17, 8, 0, 0, 0, WIB)

	const n = 100
	ids := make(chan string, n)

	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ids <- gen.Next(when)
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[string]struct{}, n)
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id generated: %s", id)
		}
		seen[id] = struct{}{}
	}

	if len(seen) != n {
		t.Fatalf("expected %d unique ids, got %d", n, len(seen))
	}
}
