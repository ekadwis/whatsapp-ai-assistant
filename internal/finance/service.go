package finance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/verssache/whatsapp-ai-assistant/internal/ai"
	"github.com/verssache/whatsapp-ai-assistant/internal/sheets"
)

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

type FinanceService struct {
	repo sheets.SheetRepository
	ai   *ai.LLMClient
}

func NewFinanceService(repo sheets.SheetRepository, aiClient *ai.LLMClient) *FinanceService {
	return &FinanceService{
		repo: repo,
		ai:   aiClient,
	}
}

// RecordTransaction creates and stores one transaction to the current WIB month tab.
func (s *FinanceService) RecordTransaction(ctx context.Context, args *ai.RecordTransactionArgs) (*sheets.Transaction, error) {
	if s == nil {
		return nil, fmt.Errorf("finance service is nil")
	}
	if s.repo == nil {
		return nil, fmt.Errorf("sheet repository is nil")
	}
	if args == nil {
		return nil, fmt.Errorf("record transaction args is nil")
	}
	if strings.TrimSpace(args.Description) == "" {
		return nil, fmt.Errorf("description is required")
	}
	if strings.TrimSpace(args.Category) == "" {
		return nil, fmt.Errorf("category is required")
	}
	if args.Amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than zero")
	}

	now := nowWIB()
	tabName := tabNameForTime(now)

	if err := s.repo.EnsureTabExists(ctx, tabName); err != nil {
		return nil, fmt.Errorf("failed to ensure tab %q: %w", tabName, err)
	}

	tx := &sheets.Transaction{
		ID:          sheets.GenerateTransactionID(now),
		Date:        now,
		Type:        toTransactionType(args.Type),
		Category:    strings.TrimSpace(args.Category),
		Description: strings.TrimSpace(args.Description),
		Amount:      args.Amount,
	}

	if err := s.repo.AppendTransaction(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to append transaction: %w", err)
	}

	return tx, nil
}

func nowWIB() time.Time {
	return time.Now().In(sheets.WIB)
}

func tabNameForTime(t time.Time) string {
	monthName, ok := monthNamesID[t.In(sheets.WIB).Month()]
	if !ok {
		monthName = t.In(sheets.WIB).Month().String()
	}
	return fmt.Sprintf("%s %d", monthName, t.In(sheets.WIB).Year())
}

func toTransactionType(raw string) sheets.TransactionType {
	if strings.EqualFold(strings.TrimSpace(raw), "income") {
		return sheets.Income
	}
	return sheets.Expense
}
