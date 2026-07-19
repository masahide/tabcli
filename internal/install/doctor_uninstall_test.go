package install

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func TestDiagnoseIsReadOnly(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tabcli")
	manifest := filepath.Join(dir, "native-host.json")
	discovery := filepath.Join(dir, "discovery.json")
	for path, contents := range map[string]string{
		executable: "binary",
		manifest:   `{"name":"io.github.yamasaki_masahide_cyg.tabcli","description":"test","path":"` + executable + `","type":"stdio","allowed_origins":["chrome-extension://ddgfmgclndpdobieomcjaklboinbaoel/"]}`,
		discovery:  `{"endpoint":"http://127.0.0.1:1234/mcp"}`,
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	before := fileDigests(t, executable, manifest, discovery)
	chromeChecks, mcpChecks := 0, 0
	result := Diagnose(DoctorOptions{
		ExecutablePath: executable,
		ManifestPath:   manifest,
		DiscoveryPath:  discovery,
		CheckChrome: func() error {
			chromeChecks++
			return nil
		},
		CheckMCP: func() error {
			mcpChecks++
			return nil
		},
	})
	if !result.Healthy() || chromeChecks != 1 || mcpChecks != 1 {
		t.Fatalf("Diagnose() = %#v, chrome=%d mcp=%d", result, chromeChecks, mcpChecks)
	}
	after := fileDigests(t, executable, manifest, discovery)
	for path, digest := range before {
		if after[path] != digest {
			t.Fatalf("doctor changed %s", path)
		}
	}
}

func TestDiagnoseRejectsMismatchedManifest(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tabcli")
	manifest := filepath.Join(dir, "native-host.json")
	discovery := filepath.Join(dir, "discovery.json")
	for path, contents := range map[string]string{executable: "binary", manifest: `{"name":"wrong"}`, discovery: `{}`} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := Diagnose(DoctorOptions{ExecutablePath: executable, ManifestPath: manifest, DiscoveryPath: discovery, CheckChrome: func() error { return nil }, CheckMCP: func() error { return nil }})
	if result.Healthy() {
		t.Fatal("doctor accepted a mismatched Native Messaging manifest")
	}
}

func TestUninstallRemovesOnlyManagedRegistrationAndSettings(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "Google", "Chrome", "NativeMessagingHosts", "io.github.yamasaki_masahide_cyg.tabcli.json")
	settings := filepath.Join(dir, "ChromeTabOrganizer")
	unrelatedManifest := filepath.Join(filepath.Dir(manifest), "com.example.unrelated.json")
	unrelatedSettings := filepath.Join(dir, "another-product")
	developerKey := filepath.Join(settings, "extension-key.pem")
	for _, path := range []string{manifest, filepath.Join(settings, "settings.json"), filepath.Join(settings, "runtime", "default", "discovery.json"), developerKey, unrelatedManifest, filepath.Join(unrelatedSettings, "data.json")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result, err := Uninstall(UninstallOptions{ManifestPath: manifest, ProductDirectory: settings})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ManifestRemoved || !result.SettingsRemoved {
		t.Fatalf("Uninstall() = %#v", result)
	}
	for _, removed := range []string{manifest, filepath.Join(settings, "settings.json"), filepath.Join(settings, "runtime")} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("managed path still exists: %s", removed)
		}
	}
	for _, preserved := range []string{developerKey, unrelatedManifest, filepath.Join(unrelatedSettings, "data.json")} {
		if _, err := os.Stat(preserved); err != nil {
			t.Fatalf("unrelated path was changed: %s: %v", preserved, err)
		}
	}
}

func fileDigests(t *testing.T, paths ...string) map[string][sha256.Size]byte {
	t.Helper()
	result := make(map[string][sha256.Size]byte, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		result[path] = sha256.Sum256(data)
	}
	return result
}
