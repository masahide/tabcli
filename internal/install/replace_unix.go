//go:build !windows

package install

import "os"

func replaceFile(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
