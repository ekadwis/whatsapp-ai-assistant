package commands

import (
	"context"
	"strings"
	"testing"
)

func TestStartHandler(t *testing.T) {
	t.Parallel()

	got := StartHandler(context.Background(), "")
	if got == "" {
		t.Fatal("expected non-empty response")
	}
	if !strings.Contains(got, "Halo") {
		t.Fatalf("expected welcome message to contain %q, got: %s", "Halo", got)
	}
	if !strings.Contains(got, "/help") {
		t.Fatalf("expected welcome message to contain %q, got: %s", "/help", got)
	}
}

func TestHelpAndMenuHandlers(t *testing.T) {
	t.Parallel()

	help := HelpHandler(context.Background(), "")
	menu := MenuHandler(context.Background(), "")

	if help == "" {
		t.Fatal("expected help response to be non-empty")
	}
	if !strings.Contains(help, "/laporan") {
		t.Fatalf("expected help message to contain %q, got: %s", "/laporan", help)
	}

	if menu != help {
		t.Fatalf("expected menu response to match help response\nhelp: %q\nmenu: %q", help, menu)
	}
}

func TestCategoryHandler(t *testing.T) {
	t.Parallel()

	got := CategoryHandler(context.Background(), "")
	if got == "" {
		t.Fatal("expected category response to be non-empty")
	}

	expectedSnippets := []string{
		"Daftar Kategori",
		"Makanan",
		"Transportasi",
		"Gaji",
		"Freelance",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(got, snippet) {
			t.Fatalf("expected category response to contain %q, got: %s", snippet, got)
		}
	}
}

func TestExportHandlerFactory_GeneratesGoogleSheetsURL(t *testing.T) {
	t.Parallel()

	sheetsID := "1AbCDeFGhijkLMNOPqRstUVwxyz1234567890"
	factory := NewExportHandlerFactory(sheetsID)

	if factory == nil {
		t.Fatal("expected non-nil export handler factory")
	}

	got := factory.Handler(context.Background(), "")
	if got == "" {
		t.Fatal("expected export response to be non-empty")
	}

	expectedURL := "https://docs.google.com/spreadsheets/d/" + sheetsID
	if !strings.Contains(got, expectedURL) {
		t.Fatalf("expected export response to contain URL %q, got: %s", expectedURL, got)
	}
	if !strings.Contains(got, "Link Google Sheets") {
		t.Fatalf("expected export response to contain %q, got: %s", "Link Google Sheets", got)
	}
}

func TestCategoryConstants_NonEmpty(t *testing.T) {
	t.Parallel()

	if len(ExpenseCategories) == 0 {
		t.Fatal("expected ExpenseCategories to be non-empty")
	}
	if len(IncomeCategories) == 0 {
		t.Fatal("expected IncomeCategories to be non-empty")
	}
	if ExpenseCategories[0] != "Makanan" {
		t.Fatalf("expected first expense category to be %q, got %q", "Makanan", ExpenseCategories[0])
	}
	if IncomeCategories[0] != "Gaji" {
		t.Fatalf("expected first income category to be %q, got %q", "Gaji", IncomeCategories[0])
	}
}
