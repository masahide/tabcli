package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunNativeHostCleansDiscoveryOnEOF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime", "discovery.json")
	var stdout, stderr bytes.Buffer
	err := RunNativeHost(context.Background(), bytes.NewReader(nil), &stdout, &stderr, path)
	if err != nil {
		t.Fatalf("RunNativeHost() error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout contains non-frame data: %q", stdout.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("discovery remains after EOF: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("stderr is not structured JSON: %q: %v", line, err)
		}
		for _, forbidden := range []string{"url", "title", "body", "text", "token", "bearerToken", "authorization"} {
			if _, present := record[forbidden]; present {
				t.Fatalf("log record contains sensitive field %q: %#v", forbidden, record)
			}
		}
	}
}
