package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/masahide/tabcli/internal/buildinfo"
)

func TestWriteManifest(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tabcli")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "Chrome", "NativeMessagingHosts", "host.json")
	if err := WriteManifest(manifestPath, executable); err != nil {
		t.Fatalf("WriteManifest() error = %v", err)
	}
	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("manifest mode = %o, want 600", info.Mode().Perm())
	}
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var got nativeManifest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Path != executable || len(got.AllowedOrigins) != 1 || got.AllowedOrigins[0] != buildinfo.AllowedExtensionOrigin {
		t.Fatalf("manifest = %#v", got)
	}
}

func TestWriteManifestReplacesExistingManifest(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tabcli")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, buildinfo.NativeManifestFileName)
	if err := os.WriteFile(manifestPath, []byte(`{"name":"old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(manifestPath, executable); err != nil {
		t.Fatal(err)
	}
	if err := validateInstalledManifest(manifestPath, executable); err != nil {
		t.Fatal(err)
	}
}
