package sheets

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"google.golang.org/api/sheets/v4"
)

func TestTabManager_EnsureTab_ConcurrentSingleCreate(t *testing.T) {
	t.Parallel()

	var getCalls int32
	var createCalls int32

	tm := &TabManager{
		spreadsheetID: "spreadsheet-1",
		existingTabs:  map[string]int{},
	}

	tm.getSpreadsheet = func(ctx context.Context) (*sheets.Spreadsheet, error) {
		atomic.AddInt32(&getCalls, 1)
		// Simulate "tab does not exist yet" from API metadata.
		return &sheets.Spreadsheet{
			Sheets: []*sheets.Sheet{
				{
					Properties: &sheets.SheetProperties{
						Title:   "Dashboard",
						SheetId: 1,
					},
				},
			},
		}, nil
	}

	tm.batchUpdate = func(ctx context.Context, req *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
		atomic.AddInt32(&createCalls, 1)

		// Basic guard: ensure the add request targets the expected tab.
		if len(req.Requests) != 1 || req.Requests[0].AddSheet == nil || req.Requests[0].AddSheet.Properties == nil {
			t.Fatalf("unexpected add sheet request shape: %#v", req)
		}
		if got := req.Requests[0].AddSheet.Properties.Title; got != "Maret 2026" {
			t.Fatalf("unexpected tab title in add request: got %q", got)
		}

		return &sheets.BatchUpdateSpreadsheetResponse{
			Replies: []*sheets.Response{
				{
					AddSheet: &sheets.AddSheetResponse{
						Properties: &sheets.SheetProperties{
							Title:   "Maret 2026",
							SheetId: 42,
						},
					},
				},
			},
		}, nil
	}

	const workers = 25
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, workers)

	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			errCh <- tm.EnsureTab(context.Background(), "Maret 2026")
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("EnsureTab returned error: %v", err)
		}
	}

	if got := atomic.LoadInt32(&createCalls); got != 1 {
		t.Fatalf("expected exactly 1 create call, got %d", got)
	}

	// One metadata read should be enough in this flow.
	if got := atomic.LoadInt32(&getCalls); got != 1 {
		t.Fatalf("expected exactly 1 metadata get call, got %d", got)
	}

	id, ok := tm.GetSheetID("Maret 2026")
	if !ok {
		t.Fatalf("expected tab to be cached after creation")
	}
	if id != 42 {
		t.Fatalf("expected cached sheet id 42, got %d", id)
	}
}

func TestTabManager_EnsureTab_CacheHitFastPath(t *testing.T) {
	t.Parallel()

	var getCalls int32
	var createCalls int32

	tm := &TabManager{
		spreadsheetID: "spreadsheet-1",
		existingTabs: map[string]int{
			"Maret 2026": 42,
		},
		getSpreadsheet: func(ctx context.Context) (*sheets.Spreadsheet, error) {
			atomic.AddInt32(&getCalls, 1)
			return &sheets.Spreadsheet{}, nil
		},
		batchUpdate: func(ctx context.Context, req *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
			atomic.AddInt32(&createCalls, 1)
			return &sheets.BatchUpdateSpreadsheetResponse{}, nil
		},
	}

	if err := tm.EnsureTab(context.Background(), "Maret 2026"); err != nil {
		t.Fatalf("EnsureTab returned error on cache hit: %v", err)
	}

	if got := atomic.LoadInt32(&getCalls); got != 0 {
		t.Fatalf("expected 0 metadata calls on cache hit, got %d", got)
	}
	if got := atomic.LoadInt32(&createCalls); got != 0 {
		t.Fatalf("expected 0 create calls on cache hit, got %d", got)
	}
}
