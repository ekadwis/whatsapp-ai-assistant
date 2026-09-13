package chart_test

import (
	"strings"
	"testing"

	"github.com/verssache/whatsapp-ai-assistant/pkg/chart"
)

func TestGenerateCategoryPieChartURL(t *testing.T) {
	cats := map[string]float64{
		"Makanan":     150000,
		"Transport":   50000,
		"Belanja":     75000,
	}

	chartURL, err := chart.GenerateCategoryPieChartURL("Pengeluaran Bulanan", cats)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(chartURL, "https://quickchart.io/chart?c=") {
		t.Errorf("expected QuickChart url prefix, got %s", chartURL)
	}

	// Empty categories should return error
	_, err = chart.GenerateCategoryPieChartURL("Test", nil)
	if err == nil {
		t.Errorf("expected error for empty categories")
	}
}
