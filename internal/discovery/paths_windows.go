//go:build windows

package discovery

import (
	"errors"
	"os"
)

func DefaultPath() (string, error) {
	root := os.Getenv("LOCALAPPDATA")
	if root == "" {
		return "", errors.New("LOCALAPPDATA is not set")
	}
	return WindowsPathIn(root), nil
}
