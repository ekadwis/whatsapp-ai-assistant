# WhatsApp AI Assistant

Personal WhatsApp AI assistant built with Go, LLM tool-calling, and Google Sheets as the database.

## What this bot can do

### Finance
- Auto-detect **expense** and **income** from natural chat
- Parse Indonesian amounts (`10k`, `50rb`, `1.5jt`, `16.000`, etc.)
- Save transactions to monthly sheets (`Maret 2026`, etc.)
- Daily/weekly/monthly reports
- Budget per category + warnings
- Edit/delete transactions by ID

### Notes
- Save quick notes to a dedicated Notes tab

### Reminders
- Create reminders from natural chat (agent-like behavior)
- `/reminder [teks]` to create reminder explicitly
- `/done [ID]` to mark reminder complete
- If reminder has no explicit time, it defaults to recurring reminders (until done)

### WhatsApp behavior
- QR login flow with persistent session
- Owner whitelist (only your number can interact)
- Typing presence and anti-ban delays

---

## Architecture (high level)

- **WhatsApp**: receives messages and sends replies
- **App Router**: command-first, then NLP/LLM routing
- **LLM**: decides actions via tool calls
- **Google Sheets**: stores transactions, notes, budgets, reminders
- **Reminder Scheduler**: background processor for due reminders

---

## Requirements

1. Go `1.25.5`
2. LLM endpoint (OpenAI-compatible) + API key
3. Google Cloud project with Sheets API enabled
4. Google service account JSON credential
5. One Google Spreadsheet for bot data

---

## 1) Create Google Service Account JSON

1. Open Google Cloud Console.
2. Create/select a project.
3. Enable **Google Sheets API**.
4. Go to **IAM & Admin → Service Accounts**.
5. Create a service account (e.g. `wa-assistant-bot`).
6. Open the service account → **Keys** → **Add Key** → **Create new key** → **JSON**.
7. Download the JSON file and place it in your project, for example:
   - `./data/google/service-account.json`

> Keep this file private. Never commit it to Git.

---

## 2) Create Google Spreadsheet and get Spreadsheet ID

1. Create a new Google Sheet.
2. Copy the URL, for example:
   - `https://docs.google.com/spreadsheets/d/1AbCdEfGhIjKlMnOpQrStUvWxYz1234567890/edit#gid=0`
3. Your `SHEETS_SPREADSHEET_ID` is the part between `/d/` and `/edit`:
   - `1AbCdEfGhIjKlMnOpQrStUvWxYz1234567890`

---

## 3) Share Sheet with Service Account Email (important)

1. Open your service account details in Google Cloud Console.
2. Copy the service account email, usually like:
   - `wa-assistant-bot@your-project-id.iam.gserviceaccount.com`
3. Open your spreadsheet → **Share**.
4. Add the service account email as **Editor**.
5. Save.

If this step is skipped, the bot cannot read/write your sheet.

---

## 4) Configure environment

Create `.env` from `.env.example`:

```bash
cp .env.example .env
```

Then fill values in `.env`.

### Required env vars

- `LLM_API_KEY`
- `LLM_BASE_URL` (OpenAI-compatible base URL)
- `LLM_MODEL`
- `GOOGLE_APPLICATION_CREDENTIALS` (path to your JSON file)
- `SHEETS_SPREADSHEET_ID`
- `WHATSAPP_SESSION_DB_PATH` (SQLite file path)
- `OWNER_PHONE_NUMBER` (digits only, no `+`, e.g. `6281234567890`)

---

## 5) Install dependencies

```bash
go mod tidy
```

---

## 6) Run the bot

```bash
go run ./cmd/bot/
```

On first run:
1. QR code appears in terminal
2. Open WhatsApp on phone → **Linked Devices**
3. Scan QR
4. Bot is online and starts processing messages

---

## Commands

### General
- `/start` — welcome
- `/help` or `/menu` — command list
- `/kategori` — category list
- `/export` — sheet URL

### Finance
- Natural examples:
  - `beli nasi goreng 18k`
  - `gaji bulan ini 5jt`
- `/laporan [hari ini|minggu ini|bulan ini]`
- `/budget [kategori] [jumlah]`
  - example: `/budget Makanan 500000`
- `/edit [ID] [field] [nilai]`
- `/hapus [ID]`

### Notes
- `/notes [catatan]`

### Reminders
- Natural examples:
  - `ingetin besok bayar kuliah`
  - `ingatkan tanggal 26 maret jam 10 ada zoom meeting`
  - `gw udah ada zoom meeting` (completion intent)
- Explicit:
  - `/reminder [teks]`
  - `/done [ID]`

---

## Data tabs used in Sheets

Bot will create/manage tabs such as:
- Monthly finance tab (e.g. `Maret 2026`)
- `Dashboard`
- `Budget`
- `Notes`
- `Reminders`

---

## Security notes

- Never commit `.env` with real secrets.
- Never commit service account JSON to public repo.
- Rotate LLM key/service-account key if exposed.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
