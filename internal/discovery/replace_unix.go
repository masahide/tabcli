//go:build !windows

package discovery

import "os"

func replaceFile(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }
