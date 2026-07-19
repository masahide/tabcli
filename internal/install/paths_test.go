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
	if got, want := NativeMessagingManifestIn(config), filepath.Join(config, "Google", "Chrome", "NativeMessagingHosts", "io.github.yamasaki_masahide_cyg.tabcli.json"); got != want {
		t.Fatalf("manifest path=%q want=%q", got, want)
	}
	if got, want := ProductDirectoryIn(config), filepath.Join(config, "ChromeTabOrganizer"); got != want {
		t.Fatalf("product path=%q want=%q", got, want)
	}
}
