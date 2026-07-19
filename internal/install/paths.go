package install

import (
	"os"
	"path/filepath"
	"runtime"
)

func ProductDirectory() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", &UnsupportedOSError{OS: runtime.GOOS}
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return ProductDirectoryIn(config), nil
}

func ProductDirectoryIn(config string) string { return filepath.Join(config, "ChromeTabOrganizer") }

func NativeMessagingManifest() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", &UnsupportedOSError{OS: runtime.GOOS}
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return NativeMessagingManifestIn(config), nil
}

func NativeMessagingManifestIn(config string) string {
	return filepath.Join(config, "Google", "Chrome", "NativeMessagingHosts", "io.github.yamasaki_masahide_cyg.tabcli.json")
}

type UnsupportedOSError struct{ OS string }

func (err *UnsupportedOSError) Error() string { return "unsupported OS " + err.OS }
