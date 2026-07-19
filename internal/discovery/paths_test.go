package discovery

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestDarwinCurrentUserPaths(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS path contract")
	}
	config := filepath.Join(string(filepath.Separator), "Users", "test", "Library", "Application Support")
	want := filepath.Join(config, "ChromeTabOrganizer", "runtime", "discovery.json")
	if got := PathIn(config); got != want {
		t.Fatalf("discovery path=%q want=%q", got, want)
	}
}
