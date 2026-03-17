package sheets

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type GoogleSheetRepository struct {
	service       *sheets.Service
	spreadsheetID string
	tabManager    *TabManager
}

var _ SheetRepository = (*GoogleSheetRepository)(nil)

func NewGoogleSheetRepository(credsPath, spreadsheetID string) (*GoogleSheetRepository, error) {
	ctx := context.Background()

	srv, err := sheets.NewService(
		ctx,
		option.WithCredentialsFile(credsPath),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create sheets service: %w", err)
	}

	return &GoogleSheetRepository{
		service:       srv,
		spreadsheetID: spreadsheetID,
		tabManager:    NewTabManager(srv, spreadsheetID),
	}, nil
}

var monthNamesID = map[time.Month]string{
	time.January:   "Januari",
	time.February:  "Februari",
	time.March:     "Maret",
	time.April:     "April",
	time.May:       "Mei",
	time.June:      "Juni",
	time.July:      "Juli",
	time.August:    "Agustus",
	time.September: "September",
	time.October:   "Oktober",
	time.November:  "November",
	time.December:  "Desember",
}

func tabNameForTime(t time.Time) string {
	local := t.In(WIB)
	monthName, ok := monthNamesID[local.Month()]
	if !ok {
		monthName = local.Month().String()
	}
	return fmt.Sprintf("%s %d", monthName, local.Year())
}

// AppendTransaction adds a transaction row to the month tab.
func (r *GoogleSheetRepository) AppendTransaction(ctx context.Context, tx *Transaction) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if r.service == nil {
		return fmt.Errorf("sheets service is nil")
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	tabName := tabNameForTime(tx.Date)

	if err := r.EnsureTabExists(ctx, tabName); err != nil {
		return fmt.Errorf("failed to ensure tab %q: %w", tabName, err)
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{tx.ToRow()},
	}
	appendRange := fmt.Sprintf("'%s'!A:G", tabName)

	_, err := r.service.Spreadsheets.Values.
		Append(r.spreadsheetID, appendRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("sheets append failed: %w", err)
	}

	return nil
}

// GetTransactions reads all transactions for a date range/tab.
// TODO: implement Google Sheets read logic.
func (r *GoogleSheetRepository) GetTransactions(ctx context.Context, tabName string) ([]Transaction, error) {
	_ = ctx
	_ = tabName
	return nil, nil
}

// GetTransactionByID finds a specific transaction across tabs.
// TODO: implement lookup logic across monthly tabs.
func (r *GoogleSheetRepository) GetTransactionByID(ctx context.Context, id string) (*Transaction, int, string, error) {
	_ = ctx
	_ = id
	return nil, 0, "", nil
}

// UpdateTransaction updates a row at specific index.
// TODO: implement row update logic.
func (r *GoogleSheetRepository) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *Transaction) error {
	_ = ctx
	_ = tabName
	_ = rowIndex
	_ = tx
	return nil
}

// DeleteTransaction removes a row.
// TODO: implement row delete logic.
func (r *GoogleSheetRepository) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error {
	_ = ctx
	_ = tabName
	_ = rowIndex
	return nil
}

// AppendNote adds a note to the Notes tab.
// TODO: implement note append logic.
func (r *GoogleSheetRepository) AppendNote(ctx context.Context, note *Note) error {
	_ = ctx
	_ = note
	return nil
}

// GetBudget reads the budget for a category.
// TODO: implement budget lookup logic.
func (r *GoogleSheetRepository) GetBudget(ctx context.Context, category string) (float64, error) {
	_ = ctx
	_ = category
	return 0, nil
}

// SetBudget writes/updates budget for a category.
// TODO: implement budget upsert logic.
func (r *GoogleSheetRepository) SetBudget(ctx context.Context, category string, amount float64) error {
	_ = ctx
	_ = category
	_ = amount
	return nil
}

// GetCategoryTotal sums amounts for a category in current month/tab.
// TODO: implement category aggregation logic.
func (r *GoogleSheetRepository) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	_ = ctx
	_ = tabName
	_ = category
	return 0, nil
}

// EnsureTabExists creates tab if it doesn't exist.
func (r *GoogleSheetRepository) EnsureTabExists(ctx context.Context, tabName string) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if r.tabManager == nil {
		return fmt.Errorf("tab manager is nil")
	}
	return r.tabManager.EnsureTab(ctx, tabName)
}

// FormatHeaders applies header formatting to a tab.
// TODO: implement header formatting logic.
func (r *GoogleSheetRepository) FormatHeaders(ctx context.Context, tabName string) error {
	_ = ctx
	_ = tabName
	return nil
}

// FormatRow applies expense/income row coloring.
// TODO: implement row formatting logic.
func (r *GoogleSheetRepository) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	_ = ctx
	_ = tabName
	_ = rowIndex
	_ = isExpense
	return nil
}

// InitDashboard creates/updates Dashboard tab with formulas.
// TODO: implement dashboard initialization logic.
func (r *GoogleSheetRepository) InitDashboard(ctx context.Context) error {
	_ = ctx
	return nil
}

// InitBudgetTab creates Budget tab structure.
// TODO: implement budget tab initialization logic.
func (r *GoogleSheetRepository) InitBudgetTab(ctx context.Context) error {
	_ = ctx
	return nil
}

// InitNotesTab creates Notes tab structure.
// TODO: implement notes tab initialization logic.
func (r *GoogleSheetRepository) InitNotesTab(ctx context.Context) error {
	_ = ctx
	return nil
}
