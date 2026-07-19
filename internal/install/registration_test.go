package install

import (
	"errors"
	"os"
	"testing"
)

type fakeRegistrationStore struct {
	value         string
	exists        bool
	readOverride  string
	readErr       error
	writeErr      error
	deleteErr     error
	overrideAfter bool
}

func (store *fakeRegistrationStore) Read() (string, error) {
	if store.readErr != nil {
		return "", store.readErr
	}
	if !store.exists {
		return "", os.ErrNotExist
	}
	if store.overrideAfter && store.readOverride != "" {
		return store.readOverride, nil
	}
	return store.value, nil
}

func (store *fakeRegistrationStore) Write(value string) error {
	if store.writeErr != nil {
		return store.writeErr
	}
	store.value, store.exists, store.overrideAfter = value, true, true
	return nil
}

func (store *fakeRegistrationStore) Delete() error {
	if store.deleteErr != nil {
		return store.deleteErr
	}
	store.value, store.exists = "", false
	return nil
}

func TestRegisterStoreWritesAndVerifies(t *testing.T) {
	store := &fakeRegistrationStore{}
	if err := registerStore(store, `C:\manifest.json`); err != nil {
		t.Fatal(err)
	}
	if !store.exists || store.value != `C:\manifest.json` {
		t.Fatalf("store = %#v", store)
	}
}

func TestRegisterStoreRollsBackNewValueOnVerificationFailure(t *testing.T) {
	store := &fakeRegistrationStore{readOverride: `C:\wrong.json`}
	if err := registerStore(store, `C:\manifest.json`); err == nil {
		t.Fatal("registerStore accepted mismatched read-back")
	}
	if store.exists {
		t.Fatalf("new registration was not rolled back: %#v", store)
	}
}

func TestRegisterStoreRestoresPreviousValueOnVerificationFailure(t *testing.T) {
	store := &fakeRegistrationStore{value: `C:\previous.json`, exists: true, readOverride: `C:\wrong.json`}
	if err := registerStore(store, `C:\manifest.json`); err == nil {
		t.Fatal("registerStore accepted mismatched read-back")
	}
	if !store.exists || store.value != `C:\previous.json` {
		t.Fatalf("previous registration was not restored: %#v", store)
	}
}

func TestUnregisterStorePreservesUnmanagedValue(t *testing.T) {
	store := &fakeRegistrationStore{value: `C:\other.json`, exists: true}
	removed, err := unregisterStore(store, `C:\manifest.json`)
	if err == nil || removed || !store.exists {
		t.Fatalf("removed=%t err=%v store=%#v", removed, err, store)
	}
}

func TestRegisterStoreDoesNotWriteAfterReadFailure(t *testing.T) {
	store := &fakeRegistrationStore{readErr: errors.New("access denied")}
	if err := registerStore(store, `C:\manifest.json`); err == nil || store.exists {
		t.Fatalf("err=%v store=%#v", err, store)
	}
}
