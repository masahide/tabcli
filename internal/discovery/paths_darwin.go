//go:build darwin

package discovery

import "os"

func DefaultPath() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return PathIn(config), nil
}
