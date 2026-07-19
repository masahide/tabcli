package release

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/masahide/tabcli/internal/buildinfo"
)

func TestValidateArtifactsAcceptsBuiltFilesAndFixedExtensionID(t *testing.T) {
	manifestData, err := os.ReadFile("../../extension/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	if got, err := ExtensionIDFromManifestKey(manifest.Key); err != nil || got != buildinfo.ExtensionID {
		t.Fatalf("extension ID = %q, %v; want %q", got, err, buildinfo.ExtensionID)
	}
	root := t.TempDir()
	writeArtifact(t, filepath.Join(root, "tabcli-darwin-arm64"), "mach-o", 0o700)
	writeArtifact(t, filepath.Join(root, "extension", "manifest.json"), string(manifestData), 0o600)
	writeArtifact(t, filepath.Join(root, "extension", "dist", "service-worker.js"), "console.log('built')", 0o600)
	writeArtifact(t, filepath.Join(root, "extension", "options.html"), "<html></html>", 0o600)
	if err := ValidateArtifacts(root); err != nil {
		t.Fatal(err)
	}
}

func TestValidateArtifactsRejectsSourcesPrivateKeysAndSecrets(t *testing.T) {
	for name, contents := range map[string]string{
		"source.ts":         "export const secret = 1",
		"extension-key.pem": "-----BEGIN PRIVATE KEY-----",
		"discovery.json":    `{"bearerToken":"should-not-ship"}`,
		"install.sh":        `curl -H '{"authorization":"bearer secret"}'`,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeArtifact(t, filepath.Join(root, name), contents, 0o600)
			if err := ValidateArtifacts(root); err == nil || !strings.Contains(err.Error(), name) {
				t.Fatalf("ValidateArtifacts() error = %v", err)
			}
		})
	}
}

func writeArtifact(t *testing.T, path, contents string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), mode); err != nil {
		t.Fatal(err)
	}
}
