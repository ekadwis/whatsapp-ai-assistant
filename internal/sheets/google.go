package sheets

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	nextID, err := r.nextDailyTransactionID(ctx, tabName, tx.Date)
	if err != nil {
		return fmt.Errorf("failed to recalculate next transaction id: %w", err)
	}
	tx.ID = nextID

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{tx.ToRow()},
	}
	appendRange := fmt.Sprintf("'%s'!A:G", tabName)

	resp, err := r.service.Spreadsheets.Values.
		Append(r.spreadsheetID, appendRange, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		IncludeValuesInResponse(true).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("sheets append failed: %w", err)
	}

	rowIndex := -1
	if resp != nil && resp.Updates != nil && resp.Updates.UpdatedRange != "" {
		rowIndex = parseUpdatedRowIndex(resp.Updates.UpdatedRange)
	}
	if rowIndex > 0 {
		_ = r.FormatRow(ctx, tabName, rowIndex, tx.Type == Expense)
	}

	return nil
}

// GetTransactions reads all transactions for a date range/tab.
func (r *GoogleSheetRepository) GetTransactions(ctx context.Context, tabName string) ([]Transaction, error) {
	if r == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if strings.TrimSpace(tabName) == "" {
		return nil, fmt.Errorf("tab name is required")
	}

	readRange := fmt.Sprintf("'%s'!A2:G", tabName)
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read transactions from %s: %w", tabName, err)
	}

	if resp == nil || len(resp.Values) == 0 {
		return []Transaction{}, nil
	}

	out := make([]Transaction, 0, len(resp.Values))
	for _, row := range resp.Values {
		tx, err := TransactionFromRow(row)
		if err != nil {
			continue
		}
		out = append(out, *tx)
	}

	return out, nil
}

// GetTransactionByID finds a specific transaction in the monthly tab inferred from ID date.
func (r *GoogleSheetRepository) GetTransactionByID(ctx context.Context, id string) (*Transaction, int, string, error) {
	if r == nil {
		return nil, 0, "", fmt.Errorf("repository is nil")
	}
	id = strings.TrimSpace(id)
	if len(id) < 8 {
		return nil, 0, "", fmt.Errorf("invalid transaction id: %s", id)
	}

	datePart := id[:8]
	t, err := time.ParseInLocation("20060102", datePart, WIB)
	if err != nil {
		return nil, 0, "", fmt.Errorf("invalid transaction id date: %w", err)
	}
	tabName := tabNameForTime(t)

	txs, err := r.GetTransactions(ctx, tabName)
	if err != nil {
		return nil, 0, "", err
	}

	for i := range txs {
		if strings.TrimSpace(txs[i].ID) == id {
			tx := txs[i]
			return &tx, i + 2, tabName, nil
		}
	}

	return nil, 0, tabName, fmt.Errorf("transaction %s not found", id)
}

// UpdateTransaction updates a row at specific index.
func (r *GoogleSheetRepository) UpdateTransaction(ctx context.Context, tabName string, rowIndex int, tx *Transaction) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if strings.TrimSpace(tabName) == "" {
		return fmt.Errorf("tab name is required")
	}
	if rowIndex < 2 {
		return fmt.Errorf("invalid row index: %d", rowIndex)
	}
	if tx == nil {
		return fmt.Errorf("transaction is nil")
	}

	writeRange := fmt.Sprintf("'%s'!A%d:G%d", tabName, rowIndex, rowIndex)
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{tx.ToRow()},
	}

	_, err := r.service.Spreadsheets.Values.
		Update(r.spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update transaction row %d on %s: %w", rowIndex, tabName, err)
	}

	_ = r.FormatRow(ctx, tabName, rowIndex, tx.Type == Expense)
	return nil
}

// DeleteTransaction removes a row.
func (r *GoogleSheetRepository) DeleteTransaction(ctx context.Context, tabName string, rowIndex int) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if strings.TrimSpace(tabName) == "" {
		return fmt.Errorf("tab name is required")
	}
	if rowIndex < 2 {
		return fmt.Errorf("invalid row index: %d", rowIndex)
	}

	sheetID, err := r.getSheetIDWithRefresh(ctx, tabName)
	if err != nil {
		return err
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    int64(sheetID),
						Dimension:  "ROWS",
						StartIndex: int64(rowIndex - 1),
						EndIndex:   int64(rowIndex),
					},
				},
			},
		},
	}

	_, err = r.service.Spreadsheets.BatchUpdate(r.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to delete transaction row %d on %s: %w", rowIndex, tabName, err)
	}

	return nil
}

// AppendNote adds a note to the Notes tab.
func (r *GoogleSheetRepository) AppendNote(ctx context.Context, note *Note) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if note == nil {
		return fmt.Errorf("note is nil")
	}

	if err := r.EnsureTabExists(ctx, "Notes"); err != nil {
		return err
	}
	_ = r.ensureNotesHeader(ctx)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{note.ToRow()},
	}

	_, err := r.service.Spreadsheets.Values.
		Append(r.spreadsheetID, "'Notes'!A:C", valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to append note: %w", err)
	}

	return nil
}

// GetBudget reads the budget for a category.
func (r *GoogleSheetRepository) GetBudget(ctx context.Context, category string) (float64, error) {
	if r == nil {
		return 0, fmt.Errorf("repository is nil")
	}

	cat := strings.TrimSpace(category)
	if cat == "" {
		return 0, fmt.Errorf("category is required")
	}

	if err := r.EnsureTabExists(ctx, "Budget"); err != nil {
		return 0, err
	}
	_ = r.ensureBudgetHeader(ctx)

	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, "'Budget'!A2:B").Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to read budget: %w", err)
	}
	if resp == nil || len(resp.Values) == 0 {
		return 0, nil
	}

	for _, row := range resp.Values {
		if len(row) < 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", row[0])), cat) {
			amount, err := cellFloat64(row[1])
			if err != nil {
				return 0, fmt.Errorf("invalid budget value for %s: %w", cat, err)
			}
			return amount, nil
		}
	}

	return 0, nil
}

// SetBudget writes/updates budget for a category.
func (r *GoogleSheetRepository) SetBudget(ctx context.Context, category string, amount float64) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}

	cat := strings.TrimSpace(category)
	if cat == "" {
		return fmt.Errorf("category is required")
	}
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	if err := r.EnsureTabExists(ctx, "Budget"); err != nil {
		return err
	}
	_ = r.ensureBudgetHeader(ctx)

	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, "'Budget'!A2:E").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to read budget tab: %w", err)
	}

	targetRow := -1
	if resp != nil {
		for i, row := range resp.Values {
			if len(row) == 0 {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", row[0])), cat) {
				targetRow = i + 2
				break
			}
		}
	}

	if targetRow > 0 {
		writeRange := fmt.Sprintf("'Budget'!A%d:E%d", targetRow, targetRow)
		values := &sheets.ValueRange{
			Values: [][]interface{}{
				{cat, amount, "", "", ""},
			},
		}
		_, err = r.service.Spreadsheets.Values.Update(r.spreadsheetID, writeRange, values).
			ValueInputOption("USER_ENTERED").
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("failed to update budget row: %w", err)
		}
		return nil
	}

	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{cat, amount, 0, amount, "✅ Aman"},
		},
	}
	_, err = r.service.Spreadsheets.Values.
		Append(r.spreadsheetID, "'Budget'!A:E", values).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to append budget row: %w", err)
	}

	return nil
}

// GetCategoryTotal sums amounts for a category in current month/tab.
func (r *GoogleSheetRepository) GetCategoryTotal(ctx context.Context, tabName string, category string) (float64, error) {
	if r == nil {
		return 0, fmt.Errorf("repository is nil")
	}
	tabName = strings.TrimSpace(tabName)
	category = strings.TrimSpace(category)
	if tabName == "" {
		return 0, fmt.Errorf("tab name is required")
	}
	if category == "" {
		return 0, fmt.Errorf("category is required")
	}

	readRange := fmt.Sprintf("'%s'!D2:G", tabName)
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to read category total from %s: %w", tabName, err)
	}
	if resp == nil || len(resp.Values) == 0 {
		return 0, nil
	}

	var total float64
	for _, row := range resp.Values {
		if len(row) < 4 {
			continue
		}
		txType := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		cat := strings.TrimSpace(fmt.Sprintf("%v", row[1]))
		if txType != string(Expense) {
			continue
		}
		if !strings.EqualFold(cat, category) {
			continue
		}
		amount, err := cellFloat64(row[3])
		if err != nil {
			continue
		}
		total += amount
	}

	return total, nil
}

// EnsureTabExists creates tab if it doesn't exist.
func (r *GoogleSheetRepository) EnsureTabExists(ctx context.Context, tabName string) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if r.tabManager == nil {
		return fmt.Errorf("tab manager is nil")
	}

	if err := r.tabManager.EnsureTab(ctx, tabName); err != nil {
		return err
	}

	switch tabName {
	case "Budget":
		return r.ensureBudgetHeader(ctx)
	case "Notes":
		return r.ensureNotesHeader(ctx)
	case "Dashboard":
		return nil
	case ReminderSheetName:
		return r.ensureReminderHeader(ctx)
	default:
		if isMonthlyTabName(tabName) {
			if err := r.ensureMonthlyHeader(ctx, tabName); err != nil {
				return err
			}
		}
	}

	return nil
}

// FormatHeaders applies header formatting to a tab.
func (r *GoogleSheetRepository) FormatHeaders(ctx context.Context, tabName string) error {
	sheetID, err := r.getSheetIDWithRefresh(ctx, tabName)
	if err != nil {
		return err
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						EndColumnIndex:   7,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							BackgroundColor: &sheets.Color{
								Red:   0.08,
								Green: 0.40,
								Blue:  0.75,
							},
							TextFormat: &sheets.TextFormat{
								Bold: true,
								ForegroundColor: &sheets.Color{
									Red:   1,
									Green: 1,
									Blue:  1,
								},
							},
						},
					},
					Fields: "userEnteredFormat(backgroundColor,textFormat)",
				},
			},
		},
	}

	_, err = r.service.Spreadsheets.BatchUpdate(r.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to format headers on tab %s: %w", tabName, err)
	}
	return nil
}

// FormatRow applies expense/income row coloring.
func (r *GoogleSheetRepository) FormatRow(ctx context.Context, tabName string, rowIndex int, isExpense bool) error {
	if rowIndex < 2 {
		return nil
	}

	sheetID, err := r.getSheetIDWithRefresh(ctx, tabName)
	if err != nil {
		return err
	}

	bg := &sheets.Color{
		Red:   0.91,
		Green: 0.96,
		Blue:  0.91,
	}
	if isExpense {
		bg = &sheets.Color{
			Red:   0.99,
			Green: 0.89,
			Blue:  0.93,
		}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    int64(rowIndex - 1),
						EndRowIndex:      int64(rowIndex),
						StartColumnIndex: 0,
						EndColumnIndex:   7,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							BackgroundColor: bg,
						},
					},
					Fields: "userEnteredFormat(backgroundColor)",
				},
			},
		},
	}

	_, err = r.service.Spreadsheets.BatchUpdate(r.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to format row %d on %s: %w", rowIndex, tabName, err)
	}
	return nil
}

// InitDashboard creates/updates Dashboard tab with formulas.
func (r *GoogleSheetRepository) InitDashboard(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}

	if err := r.EnsureTabExists(ctx, "Dashboard"); err != nil {
		return err
	}

	values := &sheets.ValueRange{
		Values: [][]interface{}{
			{"📊 Dashboard Keuangan", ""},
			{"", ""},
			{"Total Pemasukan:", 0},
			{"Total Pengeluaran:", 0},
			{"Saldo Bersih:", "=B3-B4"},
			{"", ""},
			{"📂 Ringkasan per Kategori", ""},
			{"Kategori", "Total Pengeluaran"},
		},
	}

	_, err := r.service.Spreadsheets.Values.
		Update(r.spreadsheetID, "'Dashboard'!A1:B8", values).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to initialize Dashboard values: %w", err)
	}

	sheetID, err := r.getSheetIDWithRefresh(ctx, "Dashboard")
	if err != nil {
		return err
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				MergeCells: &sheets.MergeCellsRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						EndColumnIndex:   2,
					},
					MergeType: "MERGE_ALL",
				},
			},
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						EndColumnIndex:   2,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							TextFormat: &sheets.TextFormat{
								Bold:     true,
								FontSize: 14,
							},
						},
					},
					Fields: "userEnteredFormat(textFormat)",
				},
			},
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    7,
						EndRowIndex:      8,
						StartColumnIndex: 0,
						EndColumnIndex:   2,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							BackgroundColor: &sheets.Color{
								Red:   0.08,
								Green: 0.40,
								Blue:  0.75,
							},
							TextFormat: &sheets.TextFormat{
								Bold: true,
								ForegroundColor: &sheets.Color{
									Red:   1,
									Green: 1,
									Blue:  1,
								},
							},
						},
					},
					Fields: "userEnteredFormat(backgroundColor,textFormat)",
				},
			},
		},
	}

	_, err = r.service.Spreadsheets.BatchUpdate(r.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to format Dashboard tab: %w", err)
	}

	return nil
}

// InitBudgetTab creates Budget tab structure.
func (r *GoogleSheetRepository) InitBudgetTab(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}

	if err := r.EnsureTabExists(ctx, "Budget"); err != nil {
		return err
	}
	if err := r.ensureBudgetHeader(ctx); err != nil {
		return err
	}

	if err := r.formatHeaderRow(ctx, "Budget", 5); err != nil {
		return err
	}

	return nil
}

// InitNotesTab creates Notes tab structure.
func (r *GoogleSheetRepository) InitNotesTab(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}

	if err := r.EnsureTabExists(ctx, "Notes"); err != nil {
		return err
	}
	if err := r.ensureNotesHeader(ctx); err != nil {
		return err
	}

	if err := r.formatHeaderRow(ctx, "Notes", 3); err != nil {
		return err
	}

	return nil
}

// InitReminderTab creates Reminders tab structure.
func (r *GoogleSheetRepository) InitReminderTab(ctx context.Context) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}

	if err := r.EnsureTabExists(ctx, ReminderSheetName); err != nil {
		return err
	}
	if err := r.ensureReminderHeader(ctx); err != nil {
		return err
	}

	if err := r.formatHeaderRow(ctx, ReminderSheetName, int64(len(ReminderHeaders))); err != nil {
		return err
	}

	return nil
}

func (r *GoogleSheetRepository) formatHeaderRow(ctx context.Context, tabName string, colCount int64) error {
	if colCount <= 0 {
		return nil
	}

	sheetID, err := r.getSheetIDWithRefresh(ctx, tabName)
	if err != nil {
		return err
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				RepeatCell: &sheets.RepeatCellRequest{
					Range: &sheets.GridRange{
						SheetId:          int64(sheetID),
						StartRowIndex:    0,
						EndRowIndex:      1,
						StartColumnIndex: 0,
						EndColumnIndex:   colCount,
					},
					Cell: &sheets.CellData{
						UserEnteredFormat: &sheets.CellFormat{
							BackgroundColor: &sheets.Color{
								Red:   0.08,
								Green: 0.40,
								Blue:  0.75,
							},
							TextFormat: &sheets.TextFormat{
								Bold: true,
								ForegroundColor: &sheets.Color{
									Red:   1,
									Green: 1,
									Blue:  1,
								},
							},
						},
					},
					Fields: "userEnteredFormat(backgroundColor,textFormat)",
				},
			},
		},
	}

	_, err = r.service.Spreadsheets.BatchUpdate(r.spreadsheetID, req).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to format header row on %s: %w", tabName, err)
	}
	return nil
}

func (r *GoogleSheetRepository) ensureMonthlyHeader(ctx context.Context, tabName string) error {
	headerRange := fmt.Sprintf("'%s'!A1:G1", tabName)
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, headerRange).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to check monthly header on %s: %w", tabName, err)
	}

	if resp != nil && len(resp.Values) > 0 && len(resp.Values[0]) >= 7 {
		return nil
	}

	headers := &sheets.ValueRange{
		Values: [][]interface{}{
			{"ID", "Tanggal", "Waktu", "Tipe", "Kategori", "Deskripsi", "Jumlah"},
		},
	}
	_, err = r.service.Spreadsheets.Values.Update(r.spreadsheetID, headerRange, headers).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to write monthly headers on %s: %w", tabName, err)
	}
	return r.FormatHeaders(ctx, tabName)
}

func (r *GoogleSheetRepository) ensureBudgetHeader(ctx context.Context) error {
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, "'Budget'!A1:E1").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to check budget header: %w", err)
	}
	if resp != nil && len(resp.Values) > 0 && len(resp.Values[0]) >= 5 {
		return nil
	}

	headers := &sheets.ValueRange{
		Values: [][]interface{}{
			{"Kategori", "Budget Bulanan", "Terpakai", "Sisa", "Status"},
		},
	}
	_, err = r.service.Spreadsheets.Values.Update(r.spreadsheetID, "'Budget'!A1:E1", headers).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to write budget headers: %w", err)
	}
	return nil
}

func (r *GoogleSheetRepository) ensureNotesHeader(ctx context.Context) error {
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, "'Notes'!A1:C1").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to check notes header: %w", err)
	}
	if resp != nil && len(resp.Values) > 0 && len(resp.Values[0]) >= 3 {
		return nil
	}

	headers := &sheets.ValueRange{
		Values: [][]interface{}{
			{"Tanggal", "Waktu", "Catatan"},
		},
	}
	_, err = r.service.Spreadsheets.Values.Update(r.spreadsheetID, "'Notes'!A1:C1", headers).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to write notes headers: %w", err)
	}
	return nil
}

func (r *GoogleSheetRepository) ensureReminderHeader(ctx context.Context) error {
	resp, err := r.service.Spreadsheets.Values.
		Get(r.spreadsheetID, "'Reminders'!A1:P1").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to check reminders header: %w", err)
	}
	if resp != nil && len(resp.Values) > 0 && len(resp.Values[0]) >= len(ReminderHeaders) {
		return nil
	}

	headers := &sheets.ValueRange{
		Values: [][]interface{}{ReminderHeaders},
	}
	_, err = r.service.Spreadsheets.Values.
		Update(r.spreadsheetID, "'Reminders'!A1:P1", headers).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to write reminders headers: %w", err)
	}
	return nil
}

func (r *GoogleSheetRepository) getSheetIDWithRefresh(ctx context.Context, tabName string) (int, error) {
	if id, ok := r.tabManager.GetSheetID(tabName); ok {
		return id, nil
	}
	if err := r.tabManager.RefreshCache(ctx); err != nil {
		return 0, fmt.Errorf("failed to refresh tab cache: %w", err)
	}
	if id, ok := r.tabManager.GetSheetID(tabName); ok {
		return id, nil
	}
	return 0, fmt.Errorf("tab %s not found", tabName)
}

func isMonthlyTabName(tabName string) bool {
	parts := strings.Fields(strings.TrimSpace(tabName))
	if len(parts) != 2 {
		return false
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil || year < 2000 || year > 9999 {
		return false
	}

	for _, name := range monthNamesID {
		if strings.EqualFold(parts[0], name) {
			return true
		}
	}
	return false
}

func parseUpdatedRowIndex(updatedRange string) int {
	if updatedRange == "" {
		return -1
	}
	excl := strings.LastIndex(updatedRange, "!")
	target := updatedRange
	if excl >= 0 && excl+1 < len(updatedRange) {
		target = updatedRange[excl+1:]
	}
	colon := strings.Index(target, ":")
	if colon >= 0 {
		target = target[:colon]
	}
	var digits strings.Builder
	for _, r := range target {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	if digits.Len() == 0 {
		return -1
	}
	n, err := strconv.Atoi(digits.String())
	if err != nil {
		return -1
	}
	return n
}

// AppendReminder adds a reminder row to Reminders tab.
func (r *GoogleSheetRepository) AppendReminder(ctx context.Context, reminder *Reminder) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if reminder == nil {
		return fmt.Errorf("reminder is nil")
	}
	reminder.Normalize()
	if err := reminder.Validate(); err != nil {
		return err
	}

	if err := r.InitReminderTab(ctx); err != nil {
		return err
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{reminder.ToRow()},
	}

	_, err := r.service.Spreadsheets.Values.
		Append(r.spreadsheetID, "'Reminders'!A:P", valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to append reminder: %w", err)
	}

	return nil
}

// ListActiveReminders returns active reminders.
func (r *GoogleSheetRepository) ListActiveReminders(ctx context.Context) ([]Reminder, error) {
	if r == nil {
		return nil, fmt.Errorf("repository is nil")
	}
	if err := r.InitReminderTab(ctx); err != nil {
		return nil, err
	}

	resp, err := r.service.Spreadsheets.Values.
		Get(r.spreadsheetID, "'Reminders'!A2:P").
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list reminders: %w", err)
	}
	if resp == nil || len(resp.Values) == 0 {
		return []Reminder{}, nil
	}

	out := make([]Reminder, 0, len(resp.Values))
	for _, row := range resp.Values {
		rem, err := ReminderFromRow(row)
		if err != nil {
			continue
		}
		if rem.Status == ReminderStatusActive {
			out = append(out, *rem)
		}
	}
	return out, nil
}

// GetReminderByID returns reminder and row index in Reminders tab.
func (r *GoogleSheetRepository) GetReminderByID(ctx context.Context, id string) (*Reminder, int, error) {
	if r == nil {
		return nil, 0, fmt.Errorf("repository is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, 0, fmt.Errorf("reminder ID is required")
	}
	if err := r.InitReminderTab(ctx); err != nil {
		return nil, 0, err
	}

	resp, err := r.service.Spreadsheets.Values.
		Get(r.spreadsheetID, "'Reminders'!A2:P").
		Context(ctx).
		Do()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read reminders: %w", err)
	}
	if resp == nil || len(resp.Values) == 0 {
		return nil, 0, fmt.Errorf("reminder %s not found", id)
	}

	for i, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		rowID := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if rowID != id {
			continue
		}
		rem, err := ReminderFromRow(row)
		if err != nil {
			return nil, 0, err
		}
		return rem, i + 2, nil
	}

	return nil, 0, fmt.Errorf("reminder %s not found", id)
}

// UpdateReminder updates reminder row at rowIndex.
func (r *GoogleSheetRepository) UpdateReminder(ctx context.Context, rowIndex int, reminder *Reminder) error {
	if r == nil {
		return fmt.Errorf("repository is nil")
	}
	if rowIndex < 2 {
		return fmt.Errorf("invalid row index: %d", rowIndex)
	}
	if reminder == nil {
		return fmt.Errorf("reminder is nil")
	}
	reminder.Normalize()
	if err := reminder.Validate(); err != nil {
		return err
	}

	if err := r.InitReminderTab(ctx); err != nil {
		return err
	}

	writeRange := fmt.Sprintf("'Reminders'!A%d:P%d", rowIndex, rowIndex)
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{reminder.ToRow()},
	}
	_, err := r.service.Spreadsheets.Values.
		Update(r.spreadsheetID, writeRange, valueRange).
		ValueInputOption("USER_ENTERED").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("failed to update reminder row %d: %w", rowIndex, err)
	}

	return nil
}

// ListDueReminders returns active reminders that can be sent at the provided time.
func (r *GoogleSheetRepository) ListDueReminders(ctx context.Context, now time.Time) ([]Reminder, error) {
	active, err := r.ListActiveReminders(ctx)
	if err != nil {
		return nil, err
	}

	due := make([]Reminder, 0, len(active))
	for _, rem := range active {
		copyRem := rem
		if copyRem.CanSendNow(now) {
			due = append(due, copyRem)
		}
	}
	return due, nil
}

func (r *GoogleSheetRepository) nextDailyTransactionID(ctx context.Context, tabName string, when time.Time) (string, error) {
	datePrefix := when.In(WIB).Format("20060102")
	readRange := fmt.Sprintf("'%s'!A2:A", tabName)

	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to read existing transaction IDs from %s: %w", tabName, err)
	}

	maxCounter := 0
	for _, row := range resp.Values {
		if len(row) == 0 {
			continue
		}
		id := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		counter, ok := parseDailyCounter(id, datePrefix)
		if !ok {
			continue
		}
		if counter > maxCounter {
			maxCounter = counter
		}
	}

	return fmt.Sprintf("%s-%03d", datePrefix, maxCounter+1), nil
}

func parseDailyCounter(id string, datePrefix string) (int, bool) {
	prefix := datePrefix + "-"
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}

	suffix := strings.TrimPrefix(id, prefix)
	if len(suffix) != 3 {
		return 0, false
	}

	n, err := strconv.Atoi(suffix)
	if err != nil || n <= 0 {
		return 0, false
	}

	return n, true
}
