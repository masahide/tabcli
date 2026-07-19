package install

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
	if got, want := NativeMessagingManifestIn(config), filepath.Join(config, "Google", "Chrome", "NativeMessagingHosts", "io.github.masahide.tabcli.json"); got != want {
		t.Fatalf("manifest path=%q want=%q", got, want)
	}
	if got, want := ProductDirectoryIn(config), filepath.Join(config, "tabcli"); got != want {
		t.Fatalf("product path=%q want=%q", got, want)
	}
}

func TestWindowsCurrentUserPaths(t *testing.T) {
	localAppData := filepath.Join(`C:\Users`, "日本語 User", "AppData", "Local")
	if got, want := WindowsProductDirectoryIn(localAppData), filepath.Join(localAppData, "tabcli"); got != want {
		t.Fatalf("product directory=%q want=%q", got, want)
	}
	if got, want := WindowsNativeMessagingManifestIn(localAppData), filepath.Join(localAppData, "tabcli", "native-messaging", "io.github.masahide.tabcli.json"); got != want {
		t.Fatalf("manifest=%q want=%q", got, want)
	}
}
