package chart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"
)

// GenerateCategoryPieChartURL generates a QuickChart.io URL for category expense breakdown.
func GenerateCategoryPieChartURL(title string, categories map[string]float64) (string, error) {
	if len(categories) == 0 {
		return "", fmt.Errorf("no categories provided")
	}

	type kv struct {
		k string
		v float64
	}
	var pairs []kv
	for k, v := range categories {
		if v > 0 {
			pairs = append(pairs, kv{k, v})
		}
	}
	if len(pairs) == 0 {
		return "", fmt.Errorf("no positive category values")
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].v > pairs[j].v
	})

	var labels []string
	var data []float64
	for _, p := range pairs {
		labels = append(labels, p.k)
		data = append(data, p.v)
	}

	chartConfig := map[string]interface{}{
		"type": "doughnut",
		"data": map[string]interface{}{
			"labels": labels,
			"datasets": []map[string]interface{}{
				{
					"data": data,
				},
			},
		},
		"options": map[string]interface{}{
			"title": map[string]interface{}{
				"display": true,
				"text":    title,
				"fontColor": "#ffffff",
				"fontSize":  16,
			},
			"legend": map[string]interface{}{
				"position": "bottom",
				"labels": map[string]interface{}{
					"fontColor": "#ffffff",
				},
			},
			"plugins": map[string]interface{}{
				"datalabels": map[string]interface{}{
					"color": "#ffffff",
					"font": map[string]interface{}{
						"weight": "bold",
					},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(chartConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chart json: %w", err)
	}

	chartURL := fmt.Sprintf(
		"https://quickchart.io/chart?c=%s&bkg=%%231e293b&w=500&h=350&f=png",
		url.QueryEscape(string(jsonBytes)),
	)
	return chartURL, nil
}

// DownloadChartImage downloads chart image bytes from a chart URL.
func DownloadChartImage(ctx context.Context, chartURL string) ([]byte, error) {
	if chartURL == "" {
		return nil, fmt.Errorf("chart url is empty")
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, chartURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chart image request: %w", err)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download chart image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("quickchart returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
