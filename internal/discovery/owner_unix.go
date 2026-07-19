//go:build !windows

package discovery

import (
	"fmt"
	"os"
	"syscall"
)

func validatePermissions(info os.FileInfo) error {
	if info.Mode().Perm() != 0o600 {
		return fmt.Errorf("%w: %o", ErrUnsafePermissions, info.Mode().Perm())
	}
	return nil
}

func validateOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() {
		return ErrWrongOwner
	}
	return nil
}
