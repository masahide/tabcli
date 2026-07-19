package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStressChromeExitAndNativeHostRestartCleanup(t *testing.T) {
	for iteration := 0; iteration < 20; iteration++ {
		path := filepath.Join(t.TempDir(), "runtime", "discovery.json")
		if err := RunNativeHost(context.Background(), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}, path); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("iteration %d left stale discovery: %v", iteration, err)
		}
	}
}
