package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseInstallScriptSyntax(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is unavailable on this platform")
	}
	for _, path := range []string{"../../scripts/install-release.sh", "../../scripts/install-with-gh.sh"} {
		command := exec.Command(bash, "-n", path)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("bash -n %s: %v\n%s", path, err, output)
		}
	}
}

func TestWindowsInstallScriptsContainSafetyAndVerificationContracts(t *testing.T) {
	installer, err := os.ReadFile("../../scripts/install.ps1")
	if err != nil {
		t.Fatal(err)
	}
	bootstrap, err := os.ReadFile("../../scripts/install-with-gh.ps1")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"LOCALAPPDATA", "Programs\\tabcli", "Completely exit Google Chrome", "tabcli-extension.zip", "version.json"} {
		if !strings.Contains(string(installer), marker) {
			t.Errorf("install.ps1 lacks %q", marker)
		}
	}
	for _, marker := range []string{"windows-amd64.zip", "SHA256SUMS", "Get-FileHash", "install.ps1"} {
		if !strings.Contains(string(bootstrap), marker) {
			t.Errorf("install-with-gh.ps1 lacks %q", marker)
		}
	}
}

func TestWindowsReleasePublisherContainsBuildVerificationAndExplicitPublishGate(t *testing.T) {
	script, err := os.ReadFile("../../scripts/publish-windows-release.ps1")
	if err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{
		"--target\", \"windows-amd64",
		"SHA256SUMS",
		"Get-FileHash",
		"version.json",
		"[switch]$Publish",
		"\"release\", \"create\", $tag",
		"Invoke-Native \"gh\" $releaseArguments",
		"git\" @(\"push",
		"--verify-tag",
	} {
		if !strings.Contains(string(script), marker) {
			t.Errorf("publish-windows-release.ps1 lacks %q", marker)
		}
	}
}

func TestValidateExtensionVersion(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"manifest.json", "package.json"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(`{"version":"1.2.3"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := validateExtensionVersion(root, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := validateExtensionVersion(root, "1.2.4"); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("validateExtensionVersion() error = %v", err)
	}
}

func TestValidateSourceRevisionRejectsNonHeadCommit(t *testing.T) {
	err := validateSourceRevision("../..", "0000000000000000000000000000000000000000")
	if err == nil || !strings.Contains(err.Error(), "does not match source HEAD") {
		t.Fatalf("validateSourceRevision() error = %v", err)
	}
}
