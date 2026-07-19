package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/masahide/tabcli/internal/buildinfo"
)

type nativeManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

type DoctorOptions struct {
	ExecutablePath string
	ManifestPath   string
	DiscoveryPath  string
	CheckRegistry  func() error
	CheckChrome    func() error
	CheckMCP       func() error
}

type DoctorCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type DoctorResult struct {
	Checks             []DoctorCheck `json:"checks"`
	UpdateInstructions string        `json:"updateInstructions,omitempty"`
}

func (result DoctorResult) Healthy() bool {
	for _, check := range result.Checks {
		if !check.OK {
			return false
		}
	}
	return true
}

// Diagnose performs only reads and caller-supplied read-only probes.
func Diagnose(options DoctorOptions) DoctorResult {
	result := DoctorResult{UpdateInstructions: "Update the extension and tabcli together, then restart Chrome if protocol versions differ."}
	for _, check := range []struct {
		name string
		path string
	}{{"executable", options.ExecutablePath}, {"native_manifest", options.ManifestPath}} {
		_, err := os.Lstat(check.path)
		if err == nil && check.name == "native_manifest" {
			err = validateInstalledManifest(check.path, options.ExecutablePath)
		}
		result.Checks = append(result.Checks, DoctorCheck{Name: check.name, OK: err == nil, Message: diagnosticMessage(err)})
	}
	if options.CheckRegistry != nil {
		err := options.CheckRegistry()
		result.Checks = append(result.Checks, DoctorCheck{Name: "registry", OK: err == nil, Message: diagnosticMessage(err)})
	}
	_, discoveryErr := os.Lstat(options.DiscoveryPath)
	result.Checks = append(result.Checks, DoctorCheck{Name: "discovery", OK: discoveryErr == nil, Message: diagnosticMessage(discoveryErr)})
	for _, check := range []struct {
		name string
		fn   func() error
	}{{"chrome", options.CheckChrome}, {"mcp", options.CheckMCP}} {
		err := errors.New("probe unavailable")
		if check.fn != nil {
			err = check.fn()
		}
		result.Checks = append(result.Checks, DoctorCheck{Name: check.name, OK: err == nil, Message: diagnosticMessage(err)})
	}
	return result
}

func validateInstalledManifest(path, executable string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest nativeManifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return fmt.Errorf("invalid Native Messaging manifest: %w", err)
	}
	if manifest.Name != buildinfo.NativeHostName || manifest.Type != "stdio" || manifest.Path != executable ||
		len(manifest.AllowedOrigins) != 1 || manifest.AllowedOrigins[0] != buildinfo.AllowedExtensionOrigin {
		return errors.New("Native Messaging manifest does not match this tabcli and extension")
	}
	return nil
}

func diagnosticMessage(err error) string {
	if err == nil {
		return "ok"
	}
	return err.Error()
}

type UninstallOptions struct {
	ManifestPath     string
	ProductDirectory string
}

type UninstallResult struct {
	ManifestRemoved     bool `json:"manifestRemoved"`
	RegistrationRemoved bool `json:"registrationRemoved,omitempty"`
	SettingsRemoved     bool `json:"settingsRemoved"`
}

func Uninstall(options UninstallOptions) (UninstallResult, error) {
	if filepath.Base(options.ManifestPath) != buildinfo.NativeHostName+".json" {
		return UninstallResult{}, errors.New("refusing to remove an unmanaged Native Messaging manifest")
	}
	if filepath.Base(options.ProductDirectory) != buildinfo.ProductDirectoryName {
		return UninstallResult{}, errors.New("refusing to remove an unmanaged settings directory")
	}
	result := UninstallResult{}
	removed, err := unregisterPlatform(options.ManifestPath)
	if err != nil {
		return result, err
	}
	result.RegistrationRemoved = removed
	if err := os.Remove(options.ManifestPath); err == nil {
		result.ManifestRemoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	for _, managed := range []string{filepath.Join(options.ProductDirectory, "runtime"), filepath.Join(options.ProductDirectory, "settings.json")} {
		if err := os.RemoveAll(managed); err != nil {
			return result, err
		}
		if _, err := os.Stat(managed); !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("managed settings remain at %s", managed)
		}
	}
	result.SettingsRemoved = true
	// Remove the product directory only when no developer or future-version files remain.
	if err := os.Remove(options.ProductDirectory); err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) {
		return result, err
	}
	return result, nil
}

func CurrentExecutable() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	return filepath.Abs(resolved)
}

type InstallResult struct {
	ManifestPath string `json:"manifestPath"`
	RegistryKey  string `json:"registryKey,omitempty"`
}

func Install(executable string) (InstallResult, error) {
	manifestPath, err := NativeMessagingManifest()
	if err != nil {
		return InstallResult{}, err
	}
	previousManifest, readErr := os.ReadFile(manifestPath)
	hadPreviousManifest := readErr == nil
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return InstallResult{}, readErr
	}
	if err := WriteManifest(manifestPath, executable); err != nil {
		return InstallResult{}, err
	}
	registryKey, err := registerPlatform(manifestPath)
	if err != nil {
		if hadPreviousManifest {
			_ = os.WriteFile(manifestPath, previousManifest, 0o600)
		} else {
			_ = os.Remove(manifestPath)
		}
		return InstallResult{}, err
	}
	return InstallResult{ManifestPath: manifestPath, RegistryKey: registryKey}, nil
}

func WriteManifest(manifestPath, executable string) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("tabcli path must be absolute")
	}
	info, err := os.Stat(executable)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("tabcli path is a directory")
	}
	directory := filepath.Dir(manifestPath)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".native-manifest-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return err
	}
	manifest := nativeManifest{
		Name: buildinfo.NativeHostName, Description: "tabcli Native Messaging Host",
		Path: executable, Type: "stdio", AllowedOrigins: []string{buildinfo.AllowedExtensionOrigin},
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceFile(temporaryPath, manifestPath); err != nil {
		return err
	}
	committed = true
	return nil
}
