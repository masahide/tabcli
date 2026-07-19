package release

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDeterministicZip(t *testing.T) {
	root := t.TempDir()
	writeArtifact(t, filepath.Join(root, "b.txt"), "b", 0o600)
	writeArtifact(t, filepath.Join(root, "a.txt"), "a", 0o600)
	stamp := time.Unix(1_700_000_000, 0)
	first, second := filepath.Join(t.TempDir(), "first.zip"), filepath.Join(t.TempDir(), "second.zip")
	for _, path := range []string{first, second} {
		if err := DeterministicZip(path, root, []string{"b.txt", "a.txt"}, stamp); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := os.ReadFile(first)
	b, _ := os.ReadFile(second)
	if !bytes.Equal(a, b) {
		t.Fatal("archives are not reproducible")
	}
}
