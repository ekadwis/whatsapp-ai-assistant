package formatter_test

import (
	"strings"
	"testing"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/pkg/formatter"
)

func TestFormatIDR(t *testing.T) {
	tests := []struct {
		amount   float64
		expected string
	}{
		{0, "Rp 0"},
		{15000, "Rp 15.000"},
		{1500000, "Rp 1.500.000"},
		{-50000, "-Rp 50.000"},
		{1234.56, "Rp 1.234,56"},
	}

	for _, tt := range tests {
		got := formatter.FormatIDR(tt.amount)
		if got != tt.expected {
			t.Errorf("FormatIDR(%v) = %q; want %q", tt.amount, got, tt.expected)
		}
	}
}

func TestFormatSalaryCycleReport(t *testing.T) {
	cats := map[string]float64{
		"Makanan": 250000,
		"Belanja": 100000,
	}
	out := formatter.FormatSalaryCycleReport("23 Feb 2026 - 13 Mar 2026", 5000000, 350000, cats)
	if !strings.Contains(out, "Laporan Siklus Gajian") {
		t.Errorf("expected title to contain Laporan Siklus Gajian, got %s", out)
	}
	if !strings.Contains(out, "23 Feb 2026 - 13 Mar 2026") {
		t.Errorf("expected date range in output, got %s", out)
	}
	if !strings.Contains(out, "Rp 5.000.000") {
		t.Errorf("expected total income in output, got %s", out)
	}
}

func TestFormatAllTimeReport(t *testing.T) {
	earliest := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC)
	cats := map[string]float64{
		"Makanan": 1500000,
		"Tagihan": 2000000,
	}

	out := formatter.FormatAllTimeReport(20000000, 3500000, 16500000, 150, 10, 140, earliest, latest, cats)
	if !strings.Contains(out, "Laporan Keseluruhan (All-Time)") {
		t.Errorf("expected header in output, got %s", out)
	}
	if !strings.Contains(out, "Rp 20.000.000") {
		t.Errorf("expected total income in output, got %s", out)
	}
	if !strings.Contains(out, "Rp 16.500.000") {
		t.Errorf("expected net balance in output, got %s", out)
	}
}
