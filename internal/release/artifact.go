package release

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/masahide/tabcli/internal/buildinfo"
)

func ExtensionIDFromManifestKey(key string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("decode manifest public key: %w", err)
	}
	digest := sha256.Sum256(der)
	const alphabet = "abcdefghijklmnop"
	id := make([]byte, 32)
	for index, value := range digest[:16] {
		id[index*2] = alphabet[value>>4]
		id[index*2+1] = alphabet[value&0x0f]
	}
	return string(id), nil
}

func ValidateArtifacts(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		name := strings.ToLower(entry.Name())
		extension := strings.ToLower(filepath.Ext(name))
		if extension == ".ts" || extension == ".tsx" || extension == ".pem" || strings.Contains(name, "private-key") || strings.Contains(name, "extension-key") {
			return fmt.Errorf("forbidden source or private key in artifact: %s", path)
		}
		if name == "manifest.json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var manifest struct {
				Key string `json:"key"`
			}
			if err := json.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("invalid extension manifest %s: %w", path, err)
			}
			id, err := ExtensionIDFromManifestKey(manifest.Key)
			if err != nil {
				return fmt.Errorf("manifest %s: %w", path, err)
			}
			if id != buildinfo.ExtensionID {
				return fmt.Errorf("manifest %s extension ID %s does not match %s", path, id, buildinfo.ExtensionID)
			}
		}
		if isTextArtifact(extension) {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := strings.ToLower(string(data))
			for _, marker := range []string{"-----begin private key-----", `"bearertoken"`, `"authorization":"bearer `} {
				if strings.Contains(text, marker) {
					return fmt.Errorf("secret material in artifact: %s", path)
				}
			}
		}
		return nil
	})
}

func isTextArtifact(extension string) bool {
	switch extension {
	case ".json", ".js", ".mjs", ".html", ".css", ".txt", ".md", ".yaml", ".yml", ".sh", ".ps1":
		return true
	default:
		return false
	}
}
