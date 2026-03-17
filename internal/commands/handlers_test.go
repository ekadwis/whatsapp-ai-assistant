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

func TestIsValidExpenseCategory(t *testing.T) {
	t.Parallel()

	validCases := []string{
		"Makanan",
		"Transportasi",
		"makanan",
		"  Belanja  ",
	}
	for _, c := range validCases {
		if !isValidExpenseCategory(c) {
			t.Fatalf("expected category %q to be valid", c)
		}
	}

	invalidCases := []string{
		"",
		"   ",
		"Gaji",
		"UnknownCategory",
	}
	for _, c := range invalidCases {
		if isValidExpenseCategory(c) {
			t.Fatalf("expected category %q to be invalid", c)
		}
	}
}

func TestParseAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "plain number", input: "500000", want: 500000},
		{name: "with rp prefix", input: "Rp 50.000", want: 50000},
		{name: "k suffix", input: "16k", want: 16000},
		{name: "rb suffix", input: "16rb", want: 16000},
		{name: "ribu suffix", input: "16ribu", want: 16000},
		{name: "jt suffix", input: "1.5jt", want: 1500000},
		{name: "juta suffix", input: "2juta", want: 2000000},
		{name: "decimal comma", input: "16,5k", want: 16500},
		{name: "invalid text", input: "abc", wantErr: true},
		{name: "zero", input: "0", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseAmount(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseAmount(%q) expected error, got nil", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseAmount(%q) unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("parseAmount(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
