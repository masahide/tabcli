package cli

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{input: "30m", want: 30 * time.Minute},
		{input: "24h", want: 24 * time.Hour},
		{input: "7d", want: 7 * 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil || got != tt.want {
				t.Fatalf("ParseDuration(%q) = %v, %v; want %v", tt.input, got, err, tt.want)
			}
		})
	}
}

func TestParseDurationRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "7", "-1h", "1.5d", "forever"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseDuration(input); err == nil {
				t.Fatalf("ParseDuration(%q) succeeded", input)
			}
		})
	}
}
