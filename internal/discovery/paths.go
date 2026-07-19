package discovery

import (
	"path/filepath"

	"github.com/masahide/tabcli/internal/buildinfo"
)

const DefaultProfileID = buildinfo.ProfileID

func PathIn(config string) string {
	return filepath.Join(config, buildinfo.ProductDirectoryName, "runtime", "discovery.json")
}

func WindowsPathIn(localAppData string) string {
	return filepath.Join(localAppData, buildinfo.ProductDirectoryName, "runtime", "discovery.json")
}
