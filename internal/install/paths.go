package install

import (
	"path/filepath"

	"github.com/masahide/tabcli/internal/buildinfo"
)

func ProductDirectoryIn(config string) string {
	return filepath.Join(config, buildinfo.ProductDirectoryName)
}

func NativeMessagingManifestIn(config string) string {
	return filepath.Join(config, "Google", "Chrome", "NativeMessagingHosts", buildinfo.NativeManifestFileName)
}

func WindowsProductDirectoryIn(localAppData string) string {
	return filepath.Join(localAppData, buildinfo.ProductDirectoryName)
}

func WindowsNativeMessagingManifestIn(localAppData string) string {
	return filepath.Join(WindowsProductDirectoryIn(localAppData), "native-messaging", buildinfo.NativeManifestFileName)
}

type UnsupportedOSError struct{ OS string }

func (err *UnsupportedOSError) Error() string { return "unsupported OS " + err.OS }
