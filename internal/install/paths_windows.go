//go:build windows

package install

import (
	"errors"
	"os"
)

func localAppData() (string, error) {
	value := os.Getenv("LOCALAPPDATA")
	if value == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return value, nil
}

func ProductDirectory() (string, error) {
	root, err := localAppData()
	if err != nil {
		return "", err
	}
	return WindowsProductDirectoryIn(root), nil
}

func NativeMessagingManifest() (string, error) {
	root, err := localAppData()
	if err != nil {
		return "", err
	}
	return WindowsNativeMessagingManifestIn(root), nil
}
