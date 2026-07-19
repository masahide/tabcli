//go:build !darwin && !windows

package discovery

import (
	"fmt"
	"runtime"
)

func DefaultPath() (string, error) {
	return "", fmt.Errorf("unsupported OS %q", runtime.GOOS)
}
