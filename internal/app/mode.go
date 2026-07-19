package app

import (
	"errors"
	"strings"
)

type Mode string

const (
	ModeCLI        Mode = "cli"
	ModeNativeHost Mode = "native-host"
)

var ErrOriginNotAllowed = errors.New("extension origin is not allowed")

// DetectMode distinguishes Chrome's Native Messaging launch convention from
// ordinary user CLI arguments. Chrome passes the calling extension origin as
// the first argument on macOS.
func DetectMode(args []string, allowedOrigin string) (Mode, error) {
	if len(args) == 0 || !strings.HasPrefix(args[0], "chrome-extension://") {
		return ModeCLI, nil
	}
	if args[0] != allowedOrigin {
		return "", ErrOriginNotAllowed
	}
	return ModeNativeHost, nil
}
