package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/masahide/tabcli/internal/buildinfo"
)

const DefaultProfileID = buildinfo.ProfileID

func DefaultPath() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("unsupported OS %q", runtime.GOOS)
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return PathIn(config), nil
}

func PathIn(config string) string {
	return filepath.Join(config, "ChromeTabOrganizer", "runtime", "discovery.json")
}
