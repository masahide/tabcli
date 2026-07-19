//go:build !darwin && !windows

package install

func registerPlatform(string) (string, error) { return "", nil }

func unregisterPlatform(string) (bool, error) { return false, nil }

func RegistrationCheck(string) func() error { return nil }
