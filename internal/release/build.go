package release

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/masahide/tabcli/internal/buildinfo"
)

type BuildConfig struct {
	Root          string
	Out           string
	Version       string
	Commit        string
	Timestamp     time.Time
	Architectures []string
	Target        string
	Stdout        io.Writer
	Stderr        io.Writer
}

type Metadata struct {
	Version              string   `json:"version"`
	Commit               string   `json:"commit"`
	BuiltAt              string   `json:"builtAt"`
	Targets              []string `json:"targets"`
	ExtensionID          string   `json:"extensionId"`
	ProtocolVersion      int      `json:"protocolVersion"`
	MinimumChromeVersion string   `json:"minimumChromeVersion"`
}

func Build(config BuildConfig) error {
	if config.Root == "" || config.Out == "" || config.Version == "" || config.Commit == "" {
		return errors.New("root, out, version, and commit are required")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return err
	}
	if err := validateSourceRevision(root, config.Commit); err != nil {
		return err
	}
	out, err := filepath.Abs(config.Out)
	if err != nil {
		return err
	}
	if out == root || !strings.HasPrefix(out, root+string(filepath.Separator)) {
		return errors.New("release output must be a directory inside the repository")
	}
	if len(config.Architectures) == 0 {
		config.Architectures = []string{"arm64", "amd64"}
	}
	if config.Timestamp.IsZero() {
		config.Timestamp = time.Unix(315532800, 0).UTC()
	}
	if config.Stdout == nil {
		config.Stdout = io.Discard
	}
	if config.Stderr == nil {
		config.Stderr = io.Discard
	}
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}

	if err := runCommand(root, config.Stdout, config.Stderr, nil, "go", "test", "./..."); err != nil {
		return err
	}
	extensionRoot := filepath.Join(root, "extension")
	if err := validateExtensionVersion(extensionRoot, config.Version); err != nil {
		return err
	}
	for _, command := range [][]string{{"npm", "ci"}, {"npm", "test"}, {"npm", "run", "typecheck"}, {"npm", "run", "build"}} {
		if err := runCommand(extensionRoot, config.Stdout, config.Stderr, nil, command[0], command[1:]...); err != nil {
			return err
		}
	}
	if config.Target == "windows-amd64" {
		return buildWindows(config, root, out, extensionRoot)
	}
	if config.Target != "" && config.Target != "darwin" {
		return fmt.Errorf("unsupported release target %q", config.Target)
	}

	binaries := make(map[string]string, len(config.Architectures))
	for _, architecture := range config.Architectures {
		if architecture != "arm64" && architecture != "amd64" {
			return fmt.Errorf("unsupported darwin architecture %q", architecture)
		}
		name := "tabcli-darwin-" + architecture
		path := filepath.Join(out, name)
		ldflags := strings.Join([]string{"-s", "-w", "-X", "github.com/masahide/tabcli/internal/buildinfo.Version=" + config.Version, "-X", "github.com/masahide/tabcli/internal/buildinfo.Commit=" + config.Commit, "-X", "github.com/masahide/tabcli/internal/buildinfo.BuiltAt=" + config.Timestamp.UTC().Format(time.RFC3339)}, " ")
		environment := []string{"CGO_ENABLED=0", "GOOS=darwin", "GOARCH=" + architecture}
		if err := runCommand(root, config.Stdout, config.Stderr, environment, "go", "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", path, "./cmd/tabcli"); err != nil {
			return err
		}
		if err := adHocSignDarwinBinary(path, config.Stdout, config.Stderr); err != nil {
			return err
		}
		binaries[architecture] = path
	}

	unpacked := filepath.Join(out, "tabcli-extension-unpacked")
	if err := copyFile(filepath.Join(extensionRoot, "manifest.json"), filepath.Join(unpacked, "manifest.json"), 0o600); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(extensionRoot, "options.html"), filepath.Join(unpacked, "options.html"), 0o600); err != nil {
		return err
	}
	distFiles, err := FilesUnder(filepath.Join(extensionRoot, "dist"))
	if err != nil {
		return err
	}
	for _, relative := range distFiles {
		if err := copyFile(filepath.Join(extensionRoot, "dist", relative), filepath.Join(unpacked, "dist", relative), 0o600); err != nil {
			return err
		}
	}
	if err := ValidateArtifacts(out); err != nil {
		return err
	}
	unpackedFiles, err := FilesUnder(unpacked)
	if err != nil {
		return err
	}
	extensionZip := filepath.Join(out, "tabcli-extension.zip")
	if err := DeterministicZip(extensionZip, unpacked, unpackedFiles, config.Timestamp); err != nil {
		return err
	}

	instructions := "tabcli (macOS, ad-hoc signed; no Developer ID or notarization)\n\n1. Extract the archive for this Mac: arm64 for Apple silicon, amd64 for Intel.\n2. Run: ./install.sh\n3. Open chrome://extensions and enable Developer mode.\n4. Choose Load unpacked and select the extension directory printed by the installer.\n5. Confirm extension ID " + buildinfo.ExtensionID + ".\n6. Reload the extension or restart Chrome.\n7. Run: ~/.local/bin/tabcli --json doctor\n8. Try: ~/.local/bin/tabcli --json list\n\nThe installer verifies the ad-hoc signature and matching CLI/extension versions before installing.\nUninstall: run ~/.local/bin/tabcli uninstall, then remove the extension in chrome://extensions.\n"
	if err := os.WriteFile(filepath.Join(out, "INSTALL.txt"), []byte(instructions), 0o600); err != nil {
		return err
	}
	metadata := Metadata{Version: config.Version, Commit: config.Commit, BuiltAt: config.Timestamp.UTC().Format(time.RFC3339), ExtensionID: buildinfo.ExtensionID, ProtocolVersion: buildinfo.ProtocolVersion, MinimumChromeVersion: "121"}
	for _, architecture := range config.Architectures {
		metadata.Targets = append(metadata.Targets, "darwin/"+architecture)
	}
	metadataData, _ := json.MarshalIndent(metadata, "", "  ")
	metadataData = append(metadataData, '\n')
	if err := os.WriteFile(filepath.Join(out, "version.json"), metadataData, 0o600); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(root, "scripts", "install-with-gh.sh"), filepath.Join(out, "install-with-gh.sh"), 0o700); err != nil {
		return err
	}

	for _, architecture := range config.Architectures {
		bundleRoot := filepath.Join(out, ".bundle-"+architecture)
		if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
			return err
		}
		if err := copyFile(binaries[architecture], filepath.Join(bundleRoot, "tabcli"), 0o700); err != nil {
			return err
		}
		for _, name := range []string{"tabcli-extension.zip", "INSTALL.txt", "version.json"} {
			if err := copyFile(filepath.Join(out, name), filepath.Join(bundleRoot, name), 0o600); err != nil {
				return err
			}
		}
		if err := copyFile(filepath.Join(root, "scripts", "install-release.sh"), filepath.Join(bundleRoot, "install.sh"), 0o700); err != nil {
			return err
		}
		if err := ValidateArtifacts(bundleRoot); err != nil {
			return fmt.Errorf("validate %s bundle: %w", architecture, err)
		}
		bundleFiles, _ := FilesUnder(bundleRoot)
		bundle := filepath.Join(out, fmt.Sprintf("tabcli-%s-darwin-%s.zip", config.Version, architecture))
		if err := DeterministicZip(bundle, bundleRoot, bundleFiles, config.Timestamp); err != nil {
			return err
		}
		if err := os.RemoveAll(bundleRoot); err != nil {
			return err
		}
	}
	if err := ValidateArtifacts(out); err != nil {
		return err
	}
	return writeChecksums(out)
}

func buildWindows(config BuildConfig, root, out, extensionRoot string) error {
	binary := filepath.Join(out, "tabcli.exe")
	ldflags := strings.Join([]string{"-s", "-w", "-X", "github.com/masahide/tabcli/internal/buildinfo.Version=" + config.Version, "-X", "github.com/masahide/tabcli/internal/buildinfo.Commit=" + config.Commit, "-X", "github.com/masahide/tabcli/internal/buildinfo.BuiltAt=" + config.Timestamp.UTC().Format(time.RFC3339)}, " ")
	environment := []string{"CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64"}
	if err := runCommand(root, config.Stdout, config.Stderr, environment, "go", "build", "-trimpath", "-buildvcs=false", "-ldflags", ldflags, "-o", binary, "./cmd/tabcli"); err != nil {
		return err
	}
	unpacked := filepath.Join(out, "tabcli-extension-unpacked")
	if err := copyExtension(extensionRoot, unpacked); err != nil {
		return err
	}
	if err := ValidateArtifacts(out); err != nil {
		return err
	}
	files, err := FilesUnder(unpacked)
	if err != nil {
		return err
	}
	if err := DeterministicZip(filepath.Join(out, "tabcli-extension.zip"), unpacked, files, config.Timestamp); err != nil {
		return err
	}
	instructions := "tabcli for Windows 11 x64\n\n1. Completely exit Google Chrome before updating.\n2. Extract this ZIP and run install.ps1 in PowerShell.\n3. Open chrome://extensions and enable Developer mode.\n4. Load the unpacked extension directory printed by the installer.\n5. Confirm extension ID " + buildinfo.ExtensionID + ".\n6. Run tabcli.exe --json doctor, then tabcli.exe --json list.\n\nThe binary is not Authenticode-signed and Windows SmartScreen may warn. The installer never terminates Chrome and does not modify PATH.\n"
	if err := os.WriteFile(filepath.Join(out, "INSTALL.txt"), []byte(instructions), 0o600); err != nil {
		return err
	}
	metadata := Metadata{
		Version: config.Version, Commit: config.Commit, BuiltAt: config.Timestamp.UTC().Format(time.RFC3339),
		Targets: []string{"windows/amd64"}, ExtensionID: buildinfo.ExtensionID,
		ProtocolVersion: buildinfo.ProtocolVersion, MinimumChromeVersion: "121",
	}
	data, _ := json.MarshalIndent(metadata, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "version.json"), append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(root, "scripts", "install.ps1"), filepath.Join(out, "install.ps1"), 0o600); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(root, "scripts", "install-with-gh.ps1"), filepath.Join(out, "install-with-gh.ps1"), 0o600); err != nil {
		return err
	}
	bundleRoot := filepath.Join(out, ".bundle-windows-amd64")
	if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"tabcli.exe", "tabcli-extension.zip", "INSTALL.txt", "install.ps1", "version.json"} {
		if err := copyFile(filepath.Join(out, name), filepath.Join(bundleRoot, name), 0o600); err != nil {
			return err
		}
	}
	if err := ValidateArtifacts(bundleRoot); err != nil {
		return err
	}
	bundleFiles, err := FilesUnder(bundleRoot)
	if err != nil {
		return err
	}
	bundle := filepath.Join(out, fmt.Sprintf("tabcli-%s-windows-amd64.zip", config.Version))
	if err := DeterministicZip(bundle, bundleRoot, bundleFiles, config.Timestamp); err != nil {
		return err
	}
	if err := os.RemoveAll(bundleRoot); err != nil {
		return err
	}
	if err := ValidateArtifacts(out); err != nil {
		return err
	}
	return writeChecksums(out)
}

func copyExtension(extensionRoot, unpacked string) error {
	if err := copyFile(filepath.Join(extensionRoot, "manifest.json"), filepath.Join(unpacked, "manifest.json"), 0o600); err != nil {
		return err
	}
	if err := copyFile(filepath.Join(extensionRoot, "options.html"), filepath.Join(unpacked, "options.html"), 0o600); err != nil {
		return err
	}
	files, err := FilesUnder(filepath.Join(extensionRoot, "dist"))
	if err != nil {
		return err
	}
	for _, relative := range files {
		if err := copyFile(filepath.Join(extensionRoot, "dist", relative), filepath.Join(unpacked, "dist", relative), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func validateSourceRevision(root, commit string) error {
	headCommand := exec.Command("git", "rev-parse", "HEAD")
	headCommand.Dir = root
	headOutput, err := headCommand.Output()
	if err != nil {
		return fmt.Errorf("resolve source HEAD: %w", err)
	}
	head := strings.TrimSpace(string(headOutput))
	if commit != head {
		return fmt.Errorf("release commit %s does not match source HEAD %s", commit, head)
	}
	statusCommand := exec.Command("git", "status", "--porcelain")
	statusCommand.Dir = root
	statusOutput, err := statusCommand.Output()
	if err != nil {
		return fmt.Errorf("inspect source worktree: %w", err)
	}
	if len(statusOutput) != 0 {
		return errors.New("release source worktree is not clean")
	}
	return nil
}

func validateExtensionVersion(extensionRoot, releaseVersion string) error {
	for _, name := range []string{"manifest.json", "package.json"} {
		data, err := os.ReadFile(filepath.Join(extensionRoot, name))
		if err != nil {
			return err
		}
		var document struct {
			Version string `json:"version"`
		}
		if err := json.Unmarshal(data, &document); err != nil {
			return fmt.Errorf("parse extension/%s: %w", name, err)
		}
		if document.Version != releaseVersion {
			return fmt.Errorf("release version %s does not match extension/%s version %s", releaseVersion, name, document.Version)
		}
	}
	return nil
}

func runCommand(directory string, stdout, stderr io.Writer, extraEnvironment []string, name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Dir = directory
	command.Env = append(os.Environ(), extraEnvironment...)
	command.Stdout, command.Stderr = stdout, stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

const tabcliCodeSignIdentifier = "io.github.masahide.tabcli.tabcli"

func adHocSignDarwinBinary(path string, stdout, stderr io.Writer) error {
	if err := runCommand(filepath.Dir(path), stdout, stderr, nil,
		"/usr/bin/codesign", "--force", "--sign", "-", "--timestamp=none",
		"--identifier", tabcliCodeSignIdentifier, path,
	); err != nil {
		return fmt.Errorf("ad-hoc sign %s: %w", filepath.Base(path), err)
	}
	if err := runCommand(filepath.Dir(path), stdout, stderr, nil,
		"/usr/bin/codesign", "--verify", "--strict", "--verbose=2", path,
	); err != nil {
		return fmt.Errorf("verify ad-hoc signature for %s: %w", filepath.Base(path), err)
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func writeChecksums(root string) error {
	files, err := FilesUnder(root)
	if err != nil {
		return err
	}
	var lines []string
	for _, relative := range files {
		if relative == "SHA256SUMS" || strings.HasPrefix(relative, "tabcli-extension-unpacked/") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		lines = append(lines, hex.EncodeToString(digest[:])+"  "+filepath.ToSlash(relative))
	}
	sort.Strings(lines)
	return os.WriteFile(filepath.Join(root, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}
