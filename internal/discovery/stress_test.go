package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStressStaleDiscoveryIsNeverAccepted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runtime", "discovery.json")
	if err := Write(path, File{Endpoint: "http://127.0.0.1:43123/mcp", PID: 999999, InstanceID: "old", ProfileID: "default", ProtocolVersion: 1, CreatedAt: time.Now(), Token: "old-token"}); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 1000; index++ {
		if _, err := Read(path, ReadOptions{ProtocolVersion: 1, ProcessAlive: func(int) bool { return false }}); err == nil {
			t.Fatalf("stale discovery accepted at iteration %d", index)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("read unexpectedly changed discovery: %v", err)
	}
}
