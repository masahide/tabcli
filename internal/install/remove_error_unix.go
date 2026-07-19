//go:build !windows

package install

import (
	"errors"
	"syscall"
)

func isDirectoryNotEmpty(err error) bool { return errors.Is(err, syscall.ENOTEMPTY) }
