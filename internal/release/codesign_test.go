package release

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAdHocSignDarwinBinaryIsValidAndReproducible(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("codesign verification requires macOS")
	}

	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var signed [][]byte
	for _, directory := range []string{"first", "second"} {
		path := filepath.Join(t.TempDir(), directory, "tabcli-darwin-"+runtime.GOARCH)
		if err := copyFile(source, path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := adHocSignDarwinBinary(path, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		output, err := exec.Command("/usr/bin/codesign", "--display", "--verbose=4", path).CombinedOutput()
		if err != nil {
			t.Fatalf("codesign details: %v: %s", err, output)
		}
		for _, marker := range [][]byte{
			[]byte("Identifier=" + tabcliCodeSignIdentifier),
			[]byte("Signature=adhoc"),
			[]byte("TeamIdentifier=not set"),
		} {
			if !bytes.Contains(output, marker) {
				t.Fatalf("codesign details missing %q: %s", marker, output)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		signed = append(signed, data)
	}
	if !bytes.Equal(signed[0], signed[1]) {
		t.Fatal("ad-hoc signatures differ for identical inputs")
	}
}
