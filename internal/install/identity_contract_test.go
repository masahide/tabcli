package install

import (
	"os"
	"strings"
	"testing"

	"github.com/masahide/tabcli/internal/buildinfo"
)

func TestNativeHostIdentityMatchesChromeExtension(t *testing.T) {
	source, err := os.ReadFile("../../extension/src/service-worker.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), `const nativeHostName = "`+buildinfo.NativeHostName+`"`) {
		t.Fatalf("extension Native Host does not match %q", buildinfo.NativeHostName)
	}
	if buildinfo.WindowsRegistryKey != `HKCU\Software\Google\Chrome\NativeMessagingHosts\`+buildinfo.NativeHostName {
		t.Fatalf("registry key = %q", buildinfo.WindowsRegistryKey)
	}
}
