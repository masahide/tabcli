//go:build darwin

package install

import "os"

func ProductDirectory() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return ProductDirectoryIn(config), nil
}

func NativeMessagingManifest() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return NativeMessagingManifestIn(config), nil
}
