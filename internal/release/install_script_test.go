package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseInstallScriptSyntax(t *testing.T) {
	for _, path := range []string{"../../scripts/install-release.sh", "../../scripts/install-with-gh.sh"} {
		command := exec.Command("/bin/bash", "-n", path)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("bash -n %s: %v\n%s", path, err, output)
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
