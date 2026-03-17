# WhatsApp AI Assistant

Personal WhatsApp AI assistant built with Go for:
- Financial tracking (expense/income)
- Budget setup
- Notes saving
- Reports
- AI chat routing + tool-calling flow
- Google Sheets as storage

---

## Features

- WhatsApp bot with session persistence
- Owner whitelist enforcement (single allowed number)
- Natural language transaction logging (Bahasa Indonesia format support)
- Command router for explicit bot commands
- Google Sheets repository layer (transactions, budget, notes)
- Budget alert checks
- Edit/delete transaction support
- Rich WhatsApp response formatting with emojis
- LLM tool-calling integration with confirmation flow

---

## Project Structure

- `cmd/bot/main.go` — app entrypoint and wiring
- `internal/config` — env config loader + validation
- `internal/whatsapp` — WhatsApp client, login, message handler, presence helpers
- `internal/ai` — LLM client, parser, tool definitions
- `internal/sheets` — Sheets models/repository/tab manager
- `internal/finance` — finance domain service
- `internal/notes` — notes domain service
- `internal/commands` — command handlers/router
- `internal/app` — central app message router
- `pkg/formatter` — WhatsApp message formatter

---

## Prerequisites

1. Go `1.25.5`
2. A valid OpenAI-compatible LLM endpoint + API key
3. Google Cloud service account with Sheets API access
4. A Google Spreadsheet ID
5. WhatsApp account for QR pairing
6. CGO toolchain for SQLite (macOS: Xcode Command Line Tools)

---

## Environment Variables

Create `.env` with:

- `LLM_API_KEY`
- `LLM_BASE_URL`
- `LLM_MODEL`
- `GOOGLE_APPLICATION_CREDENTIALS` (path to service account JSON)
- `SHEETS_SPREADSHEET_ID`
- `WHATSAPP_SESSION_DB_PATH`
- `OWNER_PHONE_NUMBER` (digits only, no `+`, min 10 chars)

Example:

```
LLM_API_KEY=sk-xxxx
LLM_BASE_URL=https://your-openai-compatible-endpoint/v1
LLM_MODEL=gpt-5.4
GOOGLE_APPLICATION_CREDENTIALS=./data/google/service-account.json
SHEETS_SPREADSHEET_ID=your_spreadsheet_id
WHATSAPP_SESSION_DB_PATH=./data/whatsapp/session.db
OWNER_PHONE_NUMBER=6281234567890
```

---

## Install Dependencies

```bash
go mod tidy
```

---

## Run

```bash
go run ./cmd/bot/
```

On first run:
1. QR code appears in terminal
2. Open WhatsApp → Linked Devices → scan QR
3. Bot starts handling messages from the whitelisted owner number

---

## Commands

### General
- `/start` — welcome message
- `/help` or `/menu` — command list
- `/kategori` — list categories
- `/export` — Google Sheets URL

### Finance
- Natural language message:
  - `beli ayam crispy 16k`
  - `gaji bulan ini 5jt`
- `/laporan [hari ini|minggu ini|bulan ini]` — report by period
- `/budget [kategori] [jumlah]` — set monthly budget
  - example: `/budget Makanan 500000`
- `/edit [ID] [field] [nilai]` — edit transaction
  - example: `/edit 20260317-001 jumlah 20000`
- `/hapus [ID]` — delete transaction

### Notes
- `/notes [catatan]`
  - example: `/notes beli kado ultah minggu depan`

---

## Testing & Verification

Run full checks:

```bash
go test ./...
go build ./...
go vet ./...
```

---

## Notes

- Time handling uses WIB (`UTC+7`) in service logic.
- Transaction ID format: `YYYYMMDD-NNN`.
- Monthly sheet tabs use Indonesian month naming (e.g., `Maret 2026`).
- Non-whitelisted senders are silently ignored.

---

## Security

- Never commit real `.env` secrets.
- Never hardcode API keys in source code.
- Restrict access to your Google service account file.
