//go:build windows

package install

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/masahide/tabcli/internal/buildinfo"
	"golang.org/x/sys/windows/registry"
)

const chromeNativeMessagingRegistryPath = `Software\Google\Chrome\NativeMessagingHosts\` + buildinfo.NativeHostName

type windowsRegistrationStore struct{}

func (windowsRegistrationStore) Read() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, chromeNativeMessagingRegistryPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", os.ErrNotExist
	}
	if err != nil {
		return "", err
	}
	defer key.Close()
	value, _, err := key.GetStringValue("")
	return value, err
}

func (windowsRegistrationStore) Write(value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, chromeNativeMessagingRegistryPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue("", value)
}

func (windowsRegistrationStore) Delete() error {
	err := registry.DeleteKey(registry.CURRENT_USER, chromeNativeMessagingRegistryPath)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	return err
}

func registerPlatform(manifestPath string) (string, error) {
	absolute, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", err
	}
	if err := registerStore(windowsRegistrationStore{}, absolute); err != nil {
		return "", fmt.Errorf("register Native Messaging host: %w", err)
	}
	return buildinfo.WindowsRegistryKey, nil
}

func RegistrationCheck(manifestPath string) func() error {
	return func() error {
		stored, err := (windowsRegistrationStore{}).Read()
		if err != nil {
			return err
		}
		absolute, err := filepath.Abs(manifestPath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(filepath.Clean(stored), filepath.Clean(absolute)) {
			return fmt.Errorf("registry value %q does not match manifest %q", stored, absolute)
		}
		return nil
	}
}

func unregisterPlatform(manifestPath string) (bool, error) {
	absolute, err := filepath.Abs(manifestPath)
	if err != nil {
		return false, err
	}
	store := windowsRegistrationStore{}
	stored, err := store.Read()
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(filepath.Clean(stored), filepath.Clean(absolute)) {
		return false, errors.New("refusing to remove an unmanaged Native Messaging registry key")
	}
	return unregisterStore(store, stored)
}
