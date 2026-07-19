//go:build windows

package discovery

import "os"

// Windows current-user data is protected by the user's profile ACL. Go's
// portable FileInfo does not expose the owner SID.
func validateOwner(os.FileInfo) error { return nil }
