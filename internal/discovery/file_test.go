package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func validFile() File {
	return File{
		Endpoint:        "http://127.0.0.1:43123/mcp",
		PID:             os.Getpid(),
		InstanceID:      "instance-1",
		ProfileID:       "default",
		ProtocolVersion: 1,
		CreatedAt:       time.Unix(1_700_000_000, 0).UTC(),
		Token:           "test-token",
	}
}

func TestWriteCreatesAtomicOwnerOnlyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "discovery.json")
	want := validFile()

	if err := Write(path, want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "discovery.json" {
		t.Fatalf("atomic write left unexpected files: %v", entries)
	}

	got, err := Read(path, ReadOptions{ProtocolVersion: 1, ProcessAlive: func(int) bool { return true }})
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if got.InstanceID != want.InstanceID || got.Token != want.Token {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

func TestReadRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions differ on Windows")
	}
	dir := t.TempDir()
	realPath := filepath.Join(dir, "real.json")
	linkPath := filepath.Join(dir, "link.json")
	if err := Write(realPath, validFile()); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}

	_, err := Read(linkPath, ReadOptions{ProtocolVersion: 1, ProcessAlive: func(int) bool { return true }})
	if !errors.Is(err, ErrSymlink) {
		t.Fatalf("Read() error = %v, want %v", err, ErrSymlink)
	}
}

func TestReadRejectsStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	if err := Write(path, validFile()); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path, ReadOptions{ProtocolVersion: 1, ProcessAlive: func(int) bool { return false }})
	if !errors.Is(err, ErrStaleProcess) {
		t.Fatalf("Read() error = %v, want %v", err, ErrStaleProcess)
	}
}

func TestReadRejectsProtocolMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "discovery.json")
	if err := Write(path, validFile()); err != nil {
		t.Fatal(err)
	}

	_, err := Read(path, ReadOptions{ProtocolVersion: 2, ProcessAlive: func(int) bool { return true }})
	if !errors.Is(err, ErrProtocolVersionMismatch) {
		t.Fatalf("Read() error = %v, want %v", err, ErrProtocolVersionMismatch)
	}
}
