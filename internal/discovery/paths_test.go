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
	want := filepath.Join(config, "tabcli", "runtime", "discovery.json")
	if got := PathIn(config); got != want {
		t.Fatalf("discovery path=%q want=%q", got, want)
	}
}

func TestWindowsCurrentUserPath(t *testing.T) {
	localAppData := filepath.Join(`C:\Users`, "日本語 User", "AppData", "Local")
	want := filepath.Join(localAppData, "tabcli", "runtime", "discovery.json")
	if got := WindowsPathIn(localAppData); got != want {
		t.Fatalf("discovery path=%q want=%q", got, want)
	}
}
