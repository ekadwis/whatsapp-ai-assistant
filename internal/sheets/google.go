package sheets

import (
	"context"
	"fmt"

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

// AppendTransaction adds a transaction row to the month tab.
// TODO: implement Google Sheets append logic.
func (r *GoogleSheetRepository) AppendTransaction(ctx context.Context, tx *Transaction) error {
	_ = ctx
	_ = tx
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
