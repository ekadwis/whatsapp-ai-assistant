package ai

import (
	"math"
	"testing"
)

func TestParseIDRAmount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      float64
		wantError bool
	}{
		{name: "k suffix", input: "16k", want: 16000},
		{name: "rb suffix", input: "16rb", want: 16000},
		{name: "ribu suffix", input: "16ribu", want: 16000},
		{name: "dot thousand separator", input: "16.000", want: 16000},
		{name: "decimal comma + k", input: "16,5k", want: 16500},
		{name: "decimal dot + jt", input: "1.5jt", want: 1500000},
		{name: "plain number", input: "1500000", want: 1500000},
		{name: "50 thousand", input: "50.000", want: 50000},
		{name: "juta suffix", input: "2juta", want: 2000000},
		{name: "Rp prefix", input: "Rp 50.000", want: 50000},
		{name: "IDR prefix", input: "IDR 10.000", want: 10000},
		{name: "small amount", input: "500", want: 500},
		{name: "spaces around", input: "  1,25jt  ", want: 1250000},
		{name: "zero invalid", input: "0", wantError: true},
		{name: "empty invalid", input: "", wantError: true},
		{name: "non numeric invalid", input: "abc", wantError: true},
		{name: "suffix without number invalid", input: "k", wantError: true},
		{name: "negative invalid", input: "-1000", wantError: true},
	}

	const eps = 0.000001

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseIDRAmount(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("parseIDRAmount(%q) expected error, got nil", tc.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseIDRAmount(%q) unexpected error: %v", tc.input, err)
			}

			if math.Abs(got-tc.want) > eps {
				t.Fatalf("parseIDRAmount(%q) = %f, want %f", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatIDR(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  float64
		expect string
	}{
		{name: "whole number small", input: 16000, expect: "Rp 16.000"},
		{name: "whole number million", input: 1500000, expect: "Rp 1.500.000"},
		{name: "has decimals", input: 1234.56, expect: "Rp 1.234,56"},
		{name: "one decimal rounds to two", input: 1000.5, expect: "Rp 1.000,50"},
		{name: "rounding behavior", input: 1000.567, expect: "Rp 1.000,57"},
		{name: "zero", input: 0, expect: "Rp 0"},
		{name: "negative", input: -25000, expect: "-Rp 25.000"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := formatIDR(tc.input)
			if got != tc.expect {
				t.Fatalf("formatIDR(%f) = %q, want %q", tc.input, got, tc.expect)
			}
		})
	}
}
