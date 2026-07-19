package app

import (
	"errors"
	"testing"
)

func TestDetectMode(t *testing.T) {
	const allowedOrigin = "chrome-extension://abcdefghijklmnopabcdefghijklmnop/"

	tests := []struct {
		name    string
		args    []string
		want    Mode
		wantErr error
	}{
		{name: "ordinary CLI", args: []string{"tabs", "list"}, want: ModeCLI},
		{name: "no arguments is CLI", args: nil, want: ModeCLI},
		{name: "allowed Chrome extension origin", args: []string{allowedOrigin}, want: ModeNativeHost},
		{name: "Windows origin and parent window", args: []string{allowedOrigin, "--parent-window=0"}, want: ModeNativeHost},
		{name: "unapproved Chrome extension origin", args: []string{"chrome-extension://zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz/"}, wantErr: ErrOriginNotAllowed},
		{name: "malformed Chrome extension origin", args: []string{"chrome-extension://abcdefghijklmnopabcdefghijklmnop/not-root"}, wantErr: ErrOriginNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectMode(tt.args, allowedOrigin)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("DetectMode() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("DetectMode() = %q, want %q", got, tt.want)
			}
		})
	}
}
