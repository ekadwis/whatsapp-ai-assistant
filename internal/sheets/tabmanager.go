package sheets

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/sheets/v4"
)

type TabManager struct {
	service       *sheets.Service
	spreadsheetID string

	// Injectable hooks for deterministic unit testing.
	getSpreadsheet func(ctx context.Context) (*sheets.Spreadsheet, error)
	batchUpdate    func(ctx context.Context, req *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error)

	mu           sync.Mutex
	existingTabs map[string]int // tab name -> sheet ID
}

func NewTabManager(service *sheets.Service, spreadsheetID string) *TabManager {
	tm := &TabManager{
		service:       service,
		spreadsheetID: spreadsheetID,
		existingTabs:  make(map[string]int),
	}
	tm.getSpreadsheet = tm.defaultGetSpreadsheet
	tm.batchUpdate = tm.defaultBatchUpdate
	return tm
}

func (tm *TabManager) defaultGetSpreadsheet(ctx context.Context) (*sheets.Spreadsheet, error) {
	if tm.service == nil {
		return nil, fmt.Errorf("sheets service is nil")
	}
	return tm.service.Spreadsheets.
		Get(tm.spreadsheetID).
		Context(ctx).
		Do()
}

func (tm *TabManager) defaultBatchUpdate(ctx context.Context, req *sheets.BatchUpdateSpreadsheetRequest) (*sheets.BatchUpdateSpreadsheetResponse, error) {
	if tm.service == nil {
		return nil, fmt.Errorf("sheets service is nil")
	}
	return tm.service.Spreadsheets.
		BatchUpdate(tm.spreadsheetID, req).
		Context(ctx).
		Do()
}

// EnsureTab creates the tab if it doesn't exist. Thread-safe.
func (tm *TabManager) EnsureTab(ctx context.Context, tabName string) error {
	if tm == nil {
		return fmt.Errorf("tab manager is nil")
	}
	if tm.spreadsheetID == "" {
		return fmt.Errorf("spreadsheet ID is empty")
	}
	if tm.getSpreadsheet == nil {
		return fmt.Errorf("getSpreadsheet hook is nil")
	}
	if tm.batchUpdate == nil {
		return fmt.Errorf("batchUpdate hook is nil")
	}
	if tabName == "" {
		return fmt.Errorf("tab name is empty")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 1) Fast path: in-memory cache.
	if _, ok := tm.existingTabs[tabName]; ok {
		return nil
	}

	// 2) API check: maybe created previously outside this process.
	spreadsheet, err := tm.getSpreadsheet(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch spreadsheet metadata: %w", err)
	}

	for _, sh := range spreadsheet.Sheets {
		if sh == nil || sh.Properties == nil {
			continue
		}
		if sh.Properties.Title == tabName {
			tm.existingTabs[tabName] = int(sh.Properties.SheetId)
			return nil
		}
	}

	// 3) Not found -> create tab.
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: tabName,
					},
				},
			},
		},
	}

	resp, err := tm.batchUpdate(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create tab %q: %w", tabName, err)
	}

	// Cache created sheet ID if present in response.
	for _, reply := range resp.Replies {
		if reply == nil || reply.AddSheet == nil || reply.AddSheet.Properties == nil {
			continue
		}
		if reply.AddSheet.Properties.Title == tabName {
			tm.existingTabs[tabName] = int(reply.AddSheet.Properties.SheetId)
			return nil
		}
	}

	// Fallback: refresh cache from server if response lacked created sheet details.
	if err := tm.refreshCacheLocked(ctx); err != nil {
		return fmt.Errorf("tab created but cache refresh failed: %w", err)
	}
	if _, ok := tm.existingTabs[tabName]; !ok {
		return fmt.Errorf("tab %q creation could not be confirmed", tabName)
	}

	return nil
}

// RefreshCache reloads tab list from API.
func (tm *TabManager) RefreshCache(ctx context.Context) error {
	if tm == nil {
		return fmt.Errorf("tab manager is nil")
	}
	if tm.spreadsheetID == "" {
		return fmt.Errorf("spreadsheet ID is empty")
	}
	if tm.getSpreadsheet == nil {
		return fmt.Errorf("getSpreadsheet hook is nil")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	return tm.refreshCacheLocked(ctx)
}

func (tm *TabManager) refreshCacheLocked(ctx context.Context) error {
	spreadsheet, err := tm.getSpreadsheet(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch spreadsheet metadata: %w", err)
	}

	cache := make(map[string]int, len(spreadsheet.Sheets))
	for _, sh := range spreadsheet.Sheets {
		if sh == nil || sh.Properties == nil {
			continue
		}
		cache[sh.Properties.Title] = int(sh.Properties.SheetId)
	}
	tm.existingTabs = cache

	return nil
}

// GetSheetID returns the sheet ID for a tab name.
func (tm *TabManager) GetSheetID(tabName string) (int, bool) {
	if tm == nil {
		return 0, false
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	id, ok := tm.existingTabs[tabName]
	return id, ok
}
