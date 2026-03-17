package sheets

import "context"

// SheetRepository defines the contract for all Google Sheets persistence operations
// used by finance, reporting, budget, and notes features.
type SheetRepository interface {
	// AppendTransaction adds a transaction row to the month tab.
	AppendTransaction(ctx context.Context, tx *Transaction) error

	// GetTransactions reads all transactions for a tab (typically month tab).
	GetTransactions(ctx context.Context, tabName string) ([]Transaction, error)

	// GetTransactionByID finds a specific transaction across tabs.
	// Returns: transaction, row index, tab name, error.
	GetTransactionByID(ctx context.Context, id string) (*Transaction, int, string, error)

	// UpdateTransaction updates a row at specific index in a tab.
	UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *Transaction) error

	// DeleteTransaction removes a row from a tab.
	DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error

	// AppendNote adds a note to the Notes tab.
	AppendNote(ctx context.Context, note *Note) error

	// GetBudget reads the configured monthly budget for a category.
	GetBudget(ctx context.Context, category string) (float64, error)

	// SetBudget writes or updates monthly budget for a category.
	SetBudget(ctx context.Context, category string, amount float64) error

	// GetCategoryTotal sums amounts for a category in a given tab.
	GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error)

	// EnsureTabExists creates a tab if it does not already exist.
	EnsureTabExists(ctx context.Context, tabName string) error

	// FormatHeaders applies header formatting to a tab.
	FormatHeaders(ctx context.Context, tabName string) error

	// FormatRow applies row styling based on transaction type.
	FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error

	// InitDashboard creates or updates the Dashboard tab structure/formulas.
	InitDashboard(ctx context.Context) error

	// InitBudgetTab creates Budget tab structure.
	InitBudgetTab(ctx context.Context) error

	// InitNotesTab creates Notes tab structure.
	InitNotesTab(ctx context.Context) error
}
