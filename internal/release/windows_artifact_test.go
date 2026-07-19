package release

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestBuildWindowsArtifactShape(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	extensionRoot := t.TempDir()
	manifest, err := os.ReadFile(filepath.Join(root, "extension", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeArtifact(t, filepath.Join(extensionRoot, "manifest.json"), string(manifest), 0o600)
	writeArtifact(t, filepath.Join(extensionRoot, "options.html"), "<html></html>", 0o600)
	writeArtifact(t, filepath.Join(extensionRoot, "dist", "service-worker.js"), "export {};", 0o600)
	writeArtifact(t, filepath.Join(extensionRoot, "dist", "options.js"), "export {};", 0o600)
	config := BuildConfig{
		Version: "0.3.0", Commit: "test", Timestamp: time.Unix(1_700_000_000, 0),
		Stdout: io.Discard, Stderr: io.Discard,
	}
	if err := buildWindows(config, root, out, extensionRoot); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"tabcli.exe", "tabcli-extension.zip", "tabcli-extension-unpacked",
		"install.ps1", "install-with-gh.ps1", "INSTALL.txt", "version.json",
		"SHA256SUMS", "tabcli-0.3.0-windows-amd64.zip",
	} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("missing Windows artifact %s: %v", name, err)
		}
	}
	archive, err := zip.OpenReader(filepath.Join(out, "tabcli-0.3.0-windows-amd64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var names []string
	for _, file := range archive.File {
		names = append(names, file.Name)
	}
	sort.Strings(names)
	want := []string{"INSTALL.txt", "install.ps1", "tabcli-extension.zip", "tabcli.exe", "version.json"}
	if len(names) != len(want) {
		t.Fatalf("bundle files = %v, want %v", names, want)
	}
	for index := range want {
		if names[index] != want[index] {
			t.Fatalf("bundle files = %v, want %v", names, want)
		}
	}
}
