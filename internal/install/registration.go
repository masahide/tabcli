package install

import (
	"errors"
	"fmt"
	"os"
)

type registrationStore interface {
	Read() (string, error)
	Write(string) error
	Delete() error
}

func registerStore(store registrationStore, manifestPath string) error {
	previous, readErr := store.Read()
	hadPrevious := readErr == nil
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read Native Messaging registration: %w", readErr)
	}
	if err := store.Write(manifestPath); err != nil {
		return fmt.Errorf("write Native Messaging registration: %w", err)
	}
	stored, err := store.Read()
	if err == nil && stored == manifestPath {
		return nil
	}
	if hadPrevious {
		_ = store.Write(previous)
	} else {
		_ = store.Delete()
	}
	if err != nil {
		return fmt.Errorf("verify Native Messaging registration: %w", err)
	}
	return errors.New("verify Native Messaging registration: stored path does not match manifest")
}

func unregisterStore(store registrationStore, manifestPath string) (bool, error) {
	stored, err := store.Read()
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if stored != manifestPath {
		return false, errors.New("refusing to remove an unmanaged Native Messaging registry key")
	}
	if err := store.Delete(); err != nil {
		return false, err
	}
	return true, nil
}
