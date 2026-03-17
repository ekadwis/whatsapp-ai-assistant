package formatter

import (
	"strings"
	"testing"
)

func assertContainsAll(t *testing.T, got string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(got, p) {
			t.Fatalf("expected output to contain %q, got:\n%s", p, got)
		}
	}
}

func TestFormatExpenseRecorded(t *testing.T) {
	got := FormatExpenseRecorded("20260317-001", "ayam crispy", "Makanan", 16000)
	assertContainsAll(t, got,
		"✅", "Pengeluaran Dicatat", "20260317-001", "ayam crispy", "Makanan", "Rp 16.000", "📅",
	)
}

func TestFormatIncomeRecorded(t *testing.T) {
	got := FormatIncomeRecorded("20260317-002", "gaji bulanan", "Gaji", 5000000)
	assertContainsAll(t, got,
		"✅", "Pemasukan Dicatat", "20260317-002", "gaji bulanan", "Gaji", "Rp 5.000.000",
	)
}

func TestFormatDailyReport(t *testing.T) {
	categories := map[string]float64{
		"Makanan":      16000,
		"Transportasi": 200000,
	}
	got := FormatDailyReport("17 Maret 2026", 5000000, 216000, categories)

	assertContainsAll(t, got,
		"📊", "Laporan Harian", "17 Maret 2026",
		"Rp 5.000.000", "Rp 216.000", "Rp 4.784.000",
		"Rincian Kategori", "Makanan", "Transportasi",
	)
}

func TestFormatWeeklyReport(t *testing.T) {
	got := FormatWeeklyReport("11-17 Maret 2026", 1000000, 750000, map[string]float64{
		"Belanja": 400000,
		"Makan":   350000,
	})
	assertContainsAll(t, got,
		"Laporan Mingguan", "11-17 Maret 2026", "Rp 1.000.000", "Rp 750.000", "Rp 250.000",
	)
}

func TestFormatMonthlyReport(t *testing.T) {
	got := FormatMonthlyReport("Maret 2026", 7000000, 3200000, map[string]float64{
		"Transportasi": 500000,
		"Belanja":      1200000,
	})
	assertContainsAll(t, got,
		"Laporan Bulanan", "Maret 2026", "Rp 7.000.000", "Rp 3.200.000", "Rp 3.800.000",
	)
}

func TestFormatWelcome(t *testing.T) {
	got := FormatWelcome()
	assertContainsAll(t, got,
		"👋", "Halo", "WA AI Assistant", "/help",
	)
}

func TestFormatHelp(t *testing.T) {
	got := FormatHelp()
	assertContainsAll(t, got,
		"📖", "/laporan", "/budget", "/edit", "/hapus", "/notes", "/kategori", "/export",
	)
}

func TestFormatBudgetAlert_Over(t *testing.T) {
	got := FormatBudgetAlert("Makanan", 500000, 550000, -50000)
	assertContainsAll(t, got,
		"🚨", "Peringatan Budget", "Makanan", "Rp 500.000", "Rp 550.000", "Rp 50.000",
	)
}

func TestFormatBudgetAlert_NearLimit(t *testing.T) {
	got := FormatBudgetAlert("Transportasi", 300000, 280000, 20000)
	assertContainsAll(t, got,
		"⚠️", "Peringatan Budget", "Transportasi", "Rp 300.000", "Rp 280.000", "Rp 20.000",
	)
}

func TestFormatBudgetSet(t *testing.T) {
	got := FormatBudgetSet("Makanan", 500000)
	assertContainsAll(t, got,
		"✅", "Makanan", "Rp 500.000",
	)
}

func TestFormatNoteSaved(t *testing.T) {
	got := FormatNoteSaved("beli kado ultah minggu depan")
	assertContainsAll(t, got,
		"✅", "Catatan disimpan", "beli kado ultah minggu depan",
	)
}

func TestFormatTransactionDeleted(t *testing.T) {
	got := FormatTransactionDeleted("20260317-001")
	assertContainsAll(t, got,
		"✅", "20260317-001", "berhasil dihapus",
	)
}

func TestFormatTransactionEdited(t *testing.T) {
	got := FormatTransactionEdited("20260317-001", "amount", "16000", "20000")
	assertContainsAll(t, got,
		"✅", "20260317-001", "amount", "16000", "20000", "→",
	)
}

func TestFormatCategories(t *testing.T) {
	got := FormatCategories(
		[]string{"Makanan", "Transportasi", "Belanja"},
		[]string{"Gaji", "Freelance"},
	)
	assertContainsAll(t, got,
		"📂", "Pengeluaran", "Pemasukan", "Makanan", "Transportasi", "Belanja", "Gaji", "Freelance",
	)
}

func TestFormatExport(t *testing.T) {
	url := "https://docs.google.com/spreadsheets/d/abc123"
	got := FormatExport(url)
	assertContainsAll(t, got,
		"📊", "Link Google Sheets", url,
	)
}

func TestFormatError(t *testing.T) {
	got := FormatError("gagal koneksi")
	assertContainsAll(t, got,
		"❌", "Error", "gagal koneksi",
	)
}

func TestFormatConfirmation(t *testing.T) {
	got := FormatConfirmation("ayam crispy", "pengeluaran", 16000, "Makanan")
	assertContainsAll(t, got,
		"🤔", "pengeluaran", "ayam crispy", "Makanan", "Rp 16.000", "ya", "bukan",
	)
}

func TestFormatReportTopFiveOrder(t *testing.T) {
	got := FormatDailyReport("17 Maret 2026", 0, 0, map[string]float64{
		"A": 1,
		"B": 2,
		"C": 3,
		"D": 4,
		"E": 5,
		"F": 6,
	})

	// F should appear and A should be excluded (top 5 only).
	if strings.Contains(got, "• A:") {
		t.Fatalf("expected category A to be excluded from top 5, got:\n%s", got)
	}
	assertContainsAll(t, got, "• F:", "• E:", "• D:", "• C:", "• B:")
}

func TestFormatIDR_Internal(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"whole", 16000, "Rp 16.000"},
		{"decimal", 1234.5, "Rp 1.234,50"},
		{"negative", -25000, "-Rp 25.000"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := formatIDR(tc.in)
			if got != tc.want {
				t.Fatalf("formatIDR(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSafe(t *testing.T) {
	if got := safe(""); got != "-" {
		t.Fatalf("safe(empty) = %q, want \"-\"", got)
	}
	if got := safe("  halo  "); got != "halo" {
		t.Fatalf("safe(trim) = %q, want \"halo\"", got)
	}
}
