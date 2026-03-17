//go:build integration

package sheets_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

func TestGoogleSheetRepository_Integration(t *testing.T) {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	spreadsheetID := os.Getenv("SHEETS_SPREADSHEET_ID")

	if credsPath == "" || spreadsheetID == "" {
		t.Skip("integration env not set: GOOGLE_APPLICATION_CREDENTIALS and SHEETS_SPREADSHEET_ID are required")
	}

	repo, err := sheets.NewGoogleSheetRepository(credsPath, spreadsheetID)
	if err != nil {
		t.Fatalf("failed to create GoogleSheetRepository: %v", err)
	}

	ctx := context.Background()
	now := time.Now().In(sheets.WIB)
	tabName := fmt.Sprintf("Integration %d", now.Unix())

	t.Run("EnsureTabExists", func(t *testing.T) {
		if err := repo.EnsureTabExists(ctx, tabName); err != nil {
			t.Fatalf("EnsureTabExists failed: %v", err)
		}
	})

	t.Run("AppendAndReadTransaction", func(t *testing.T) {
		tx := &sheets.Transaction{
			ID:          sheets.GenerateTransactionID(now),
			Date:        now,
			Type:        sheets.Expense,
			Category:    "Makanan",
			Description: "integration test row",
			Amount:      12345,
		}

		if err := repo.AppendTransaction(ctx, tx); err != nil {
			t.Fatalf("AppendTransaction failed: %v", err)
		}

		rows, err := repo.GetTransactions(ctx, tabName)
		if err != nil {
			t.Fatalf("GetTransactions failed: %v", err)
		}
		if len(rows) == 0 {
			t.Fatalf("expected at least one transaction row in tab %q", tabName)
		}

		found := false
		for _, row := range rows {
			if row.ID == tx.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("appended transaction ID %q not found in tab %q", tx.ID, tabName)
		}
	})
}
