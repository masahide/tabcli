//go:build windows

package install

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isDirectoryNotEmpty(err error) bool { return errors.Is(err, windows.ERROR_DIR_NOT_EMPTY) }
