//go:build !darwin && !windows

package install

import "runtime"

func ProductDirectory() (string, error) {
	return "", &UnsupportedOSError{OS: runtime.GOOS}
}

func NativeMessagingManifest() (string, error) {
	return "", &UnsupportedOSError{OS: runtime.GOOS}
}
