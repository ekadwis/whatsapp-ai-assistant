package commands

import (
	"context"
	"fmt"

	"github.com/verssache/whatsapp-ai-assistant/pkg/formatter"
)

var ExpenseCategories = []string{
	"Makanan", "Transportasi", "Rumah Tangga", "Belanja",
	"Kesehatan", "Pendidikan", "Hiburan", "Fashion",
	"Komunikasi", "Perawatan", "Sosial", "Lainnya",
}

var IncomeCategories = []string{
	"Gaji", "Freelance", "Investasi", "Hadiah", "Transfer", "Lainnya",
}

func StartHandler(ctx context.Context, args string) string {
	_ = ctx
	_ = args
	return formatter.FormatWelcome()
}

func HelpHandler(ctx context.Context, args string) string {
	_ = ctx
	_ = args
	return formatter.FormatHelp()
}

func MenuHandler(ctx context.Context, args string) string {
	_ = ctx
	_ = args
	return formatter.FormatHelp()
}

func CategoryHandler(ctx context.Context, args string) string {
	_ = ctx
	_ = args
	return formatter.FormatCategories(ExpenseCategories, IncomeCategories)
}

type ExportHandlerFactory struct {
	sheetsID string
}

func NewExportHandlerFactory(sheetsID string) *ExportHandlerFactory {
	return &ExportHandlerFactory{sheetsID: sheetsID}
}

func (f *ExportHandlerFactory) Handler(ctx context.Context, args string) string {
	_ = ctx
	_ = args

	url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", f.sheetsID)
	return formatter.FormatExport(url)
}
