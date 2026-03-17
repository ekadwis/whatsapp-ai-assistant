package commands

import (
	"context"
	"testing"
)

func TestRouter_Route(t *testing.T) {
	t.Parallel()

	r := NewRouter()

	var gotArgs string
	r.Register("/help", func(ctx context.Context, args string) string {
		gotArgs = args
		return "help-ok"
	})
	r.Register("/start", func(ctx context.Context, args string) string {
		gotArgs = args
		return "start-ok"
	})

	tests := []struct {
		name       string
		input      string
		wantMatch  bool
		wantResult string
		wantArgs   string
	}{
		{
			name:       "exact lowercase command",
			input:      "/help",
			wantMatch:  true,
			wantResult: "help-ok",
			wantArgs:   "",
		},
		{
			name:       "case insensitive command",
			input:      "/HELP",
			wantMatch:  true,
			wantResult: "help-ok",
			wantArgs:   "",
		},
		{
			name:       "command with args",
			input:      "/help extra args",
			wantMatch:  true,
			wantResult: "help-ok",
			wantArgs:   "extra args",
		},
		{
			name:       "trim surrounding spaces",
			input:      "   /help   ",
			wantMatch:  true,
			wantResult: "help-ok",
			wantArgs:   "",
		},
		{
			name:       "unknown command",
			input:      "/unknown",
			wantMatch:  false,
			wantResult: "",
			wantArgs:   "",
		},
		{
			name:       "natural language should not match",
			input:      "beli ayam 16k",
			wantMatch:  false,
			wantResult: "",
			wantArgs:   "",
		},
		{
			name:       "empty input should not match",
			input:      "   ",
			wantMatch:  false,
			wantResult: "",
			wantArgs:   "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotArgs = "__unchanged__"
			gotResult, gotMatch := r.Route(context.Background(), tt.input)

			if gotMatch != tt.wantMatch {
				t.Fatalf("Route(%q) match = %v, want %v", tt.input, gotMatch, tt.wantMatch)
			}
			if gotResult != tt.wantResult {
				t.Fatalf("Route(%q) result = %q, want %q", tt.input, gotResult, tt.wantResult)
			}

			if tt.wantMatch {
				if gotArgs != tt.wantArgs {
					t.Fatalf("Route(%q) args = %q, want %q", tt.input, gotArgs, tt.wantArgs)
				}
			} else {
				if gotArgs != "__unchanged__" {
					t.Fatalf("Route(%q) should not call handler, but args changed to %q", tt.input, gotArgs)
				}
			}
		})
	}
}

func TestRouter_Register_Normalization(t *testing.T) {
	t.Parallel()

	r := NewRouter()

	r.Register("help", func(ctx context.Context, args string) string {
		return "ok"
	})

	got, matched := r.Route(context.Background(), "/HELP")
	if !matched {
		t.Fatal("expected /HELP to match handler registered as 'help'")
	}
	if got != "ok" {
		t.Fatalf("expected result %q, got %q", "ok", got)
	}
}

func TestRouter_Register_NilHandlerIgnored(t *testing.T) {
	t.Parallel()

	r := NewRouter()
	r.Register("/help", nil)

	if got, matched := r.Route(context.Background(), "/help"); matched || got != "" {
		t.Fatalf("expected nil handler registration to be ignored, got matched=%v result=%q", matched, got)
	}
}
