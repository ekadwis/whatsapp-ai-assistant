package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	credsPath := createTempCredsFile(t)
	setValidEnv(t, credsPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLMApiKey != "sk-test" {
		t.Fatalf("unexpected LLMApiKey: %q", cfg.LLMApiKey)
	}
	if cfg.LLMBaseURL != "https://api.example.com/v1" {
		t.Fatalf("unexpected LLMBaseURL: %q", cfg.LLMBaseURL)
	}
	if cfg.LLMModel != "gpt-5.4" {
		t.Fatalf("unexpected LLMModel: %q", cfg.LLMModel)
	}
	if cfg.GoogleCredsPath != credsPath {
		t.Fatalf("unexpected GoogleCredsPath: %q", cfg.GoogleCredsPath)
	}
	if cfg.SheetsID != "sheet-id-123" {
		t.Fatalf("unexpected SheetsID: %q", cfg.SheetsID)
	}
	if cfg.WASessionDBPath != "./data/whatsapp/session.db" {
		t.Fatalf("unexpected WASessionDBPath: %q", cfg.WASessionDBPath)
	}
	if cfg.OwnerPhoneNumber != "6281234567890" {
		t.Fatalf("unexpected OwnerPhoneNumber: %q", cfg.OwnerPhoneNumber)
	}
}

func TestLoad_MissingRequiredEnvVars(t *testing.T) {
	cases := []struct {
		name   string
		key    string
		errMsg string
	}{
		{"missing LLM_API_KEY", "LLM_API_KEY", "missing required env var: LLM_API_KEY"},
		{"missing LLM_BASE_URL", "LLM_BASE_URL", "missing required env var: LLM_BASE_URL"},
		{"missing LLM_MODEL", "LLM_MODEL", "missing required env var: LLM_MODEL"},
		{"missing GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_APPLICATION_CREDENTIALS", "missing required env var: GOOGLE_APPLICATION_CREDENTIALS"},
		{"missing SHEETS_SPREADSHEET_ID", "SHEETS_SPREADSHEET_ID", "missing required env var: SHEETS_SPREADSHEET_ID"},
		{"missing WHATSAPP_SESSION_DB_PATH", "WHATSAPP_SESSION_DB_PATH", "missing required env var: WHATSAPP_SESSION_DB_PATH"},
		{"missing OWNER_PHONE_NUMBER", "OWNER_PHONE_NUMBER", "missing required env var: OWNER_PHONE_NUMBER"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credsPath := createTempCredsFile(t)
			setValidEnv(t, credsPath)
			t.Setenv(tc.key, "")

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tc.errMsg {
				t.Fatalf("expected error %q, got %q", tc.errMsg, err.Error())
			}
		})
	}
}

func TestLoad_InvalidOwnerPhoneNumber(t *testing.T) {
	cases := []struct {
		name    string
		phone   string
		wantErr string
	}{
		{
			name:    "plus prefix is not allowed",
			phone:   "+6281234567890",
			wantErr: "invalid OWNER_PHONE_NUMBER: must contain digits only",
		},
		{
			name:    "too short",
			phone:   "123",
			wantErr: "invalid OWNER_PHONE_NUMBER: must be at least 10 digits",
		},
		{
			name:    "contains letters",
			phone:   "62812abc7890",
			wantErr: "invalid OWNER_PHONE_NUMBER: must contain digits only",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			credsPath := createTempCredsFile(t)
			setValidEnv(t, credsPath)
			t.Setenv("OWNER_PHONE_NUMBER", tc.phone)

			_, err := Load()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tc.wantErr, err.Error())
			}
		})
	}
}

func TestLoad_GoogleCredsPathNotFound(t *testing.T) {
	credsPath := filepath.Join(t.TempDir(), "does-not-exist.json")
	setValidEnv(t, credsPath)

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid GOOGLE_APPLICATION_CREDENTIALS: file does not exist") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_InvalidBaseURL(t *testing.T) {
	credsPath := createTempCredsFile(t)
	setValidEnv(t, credsPath)
	t.Setenv("LLM_BASE_URL", "ftp://example.com")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "invalid LLM_BASE_URL: must start with \"http\"" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_FromDotEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	credsPath := filepath.Join(tmpDir, "service-account.json")
	if err := os.WriteFile(credsPath, []byte(`{"type":"service_account"}`), 0o644); err != nil {
		t.Fatalf("failed writing creds file: %v", err)
	}

	envContent := strings.Join([]string{
		"LLM_API_KEY=sk-from-dotenv",
		"LLM_BASE_URL=https://dotenv.example.com/v1",
		"LLM_MODEL=gpt-dotenv",
		"GOOGLE_APPLICATION_CREDENTIALS=" + credsPath,
		"SHEETS_SPREADSHEET_ID=sheet-from-dotenv",
		"WHATSAPP_SESSION_DB_PATH=./data/wa/session.db",
		"OWNER_PHONE_NUMBER=628111222333",
		"",
	}, "\n")

	if err := os.WriteFile(filepath.Join(tmpDir, ".env"), []byte(envContent), 0o644); err != nil {
		t.Fatalf("failed writing .env file: %v", err)
	}

	// Ensure values come from .env, not process env.
	for _, key := range requiredEnvKeys() {
		t.Setenv(key, "")
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLMApiKey != "sk-from-dotenv" {
		t.Fatalf("unexpected LLMApiKey: %q", cfg.LLMApiKey)
	}
	if cfg.LLMBaseURL != "https://dotenv.example.com/v1" {
		t.Fatalf("unexpected LLMBaseURL: %q", cfg.LLMBaseURL)
	}
	if cfg.LLMModel != "gpt-dotenv" {
		t.Fatalf("unexpected LLMModel: %q", cfg.LLMModel)
	}
	if cfg.GoogleCredsPath != credsPath {
		t.Fatalf("unexpected GoogleCredsPath: %q", cfg.GoogleCredsPath)
	}
	if cfg.SheetsID != "sheet-from-dotenv" {
		t.Fatalf("unexpected SheetsID: %q", cfg.SheetsID)
	}
	if cfg.WASessionDBPath != "./data/wa/session.db" {
		t.Fatalf("unexpected WASessionDBPath: %q", cfg.WASessionDBPath)
	}
	if cfg.OwnerPhoneNumber != "628111222333" {
		t.Fatalf("unexpected OwnerPhoneNumber: %q", cfg.OwnerPhoneNumber)
	}
}

func createTempCredsFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account"}`), 0o644); err != nil {
		t.Fatalf("failed writing temp creds file: %v", err)
	}
	return path
}

func setValidEnv(t *testing.T, credsPath string) {
	t.Helper()
	t.Setenv("LLM_API_KEY", "sk-test")
	t.Setenv("LLM_BASE_URL", "https://api.example.com/v1")
	t.Setenv("LLM_MODEL", "gpt-5.4")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credsPath)
	t.Setenv("SHEETS_SPREADSHEET_ID", "sheet-id-123")
	t.Setenv("WHATSAPP_SESSION_DB_PATH", "./data/whatsapp/session.db")
	t.Setenv("OWNER_PHONE_NUMBER", "6281234567890")
}

func requiredEnvKeys() []string {
	return []string{
		"LLM_API_KEY",
		"LLM_BASE_URL",
		"LLM_MODEL",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"SHEETS_SPREADSHEET_ID",
		"WHATSAPP_SESSION_DB_PATH",
		"OWNER_PHONE_NUMBER",
	}
}
