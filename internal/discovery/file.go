package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/masahide/tabcli/internal/buildinfo"
)

var (
	ErrSymlink                 = errors.New("discovery file is a symlink")
	ErrStaleProcess            = errors.New("discovery process is not running")
	ErrProtocolVersionMismatch = errors.New("PROTOCOL_VERSION_MISMATCH")
	ErrUnsafePermissions       = errors.New("discovery file permissions are not owner-only")
	ErrWrongOwner              = errors.New("discovery file has a different owner")
	ErrInvalidEndpoint         = errors.New("discovery endpoint is not loopback HTTP")
)

type File struct {
	Endpoint        string    `json:"endpoint"`
	PID             int       `json:"pid"`
	InstanceID      string    `json:"instanceId"`
	ProfileID       string    `json:"profileId"`
	ProtocolVersion int       `json:"protocolVersion"`
	CreatedAt       time.Time `json:"createdAt"`
	Token           string    `json:"bearerToken"`
}

type ReadOptions struct {
	ProtocolVersion int
	ProcessAlive    func(int) bool
}

func Write(path string, file File) error {
	if err := validateEndpoint(file.Endpoint); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".discovery-*")
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
	encoder := json.NewEncoder(temporary)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(file); err != nil {
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
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	committed = true
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func Read(path string, options ReadOptions) (File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return File{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return File{}, ErrSymlink
	}
	if info.Mode().Perm() != 0o600 {
		return File{}, fmt.Errorf("%w: %o", ErrUnsafePermissions, info.Mode().Perm())
	}
	if runtime.GOOS != "windows" {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || int(stat.Uid) != os.Getuid() {
			return File{}, ErrWrongOwner
		}
	}
	handle, err := os.Open(path)
	if err != nil {
		return File{}, err
	}
	defer handle.Close()
	var file File
	decoder := json.NewDecoder(handle)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&file); err != nil {
		return File{}, err
	}
	if file.ProtocolVersion != options.ProtocolVersion {
		return File{}, fmt.Errorf("%w: got %d, want %d", ErrProtocolVersionMismatch, file.ProtocolVersion, options.ProtocolVersion)
	}
	if err := validateEndpoint(file.Endpoint); err != nil {
		return File{}, err
	}
	processAlive := options.ProcessAlive
	if processAlive == nil {
		processAlive = defaultProcessAlive
	}
	if file.PID <= 0 || !processAlive(file.PID) {
		return File{}, fmt.Errorf("%w: %d", ErrStaleProcess, file.PID)
	}
	if file.InstanceID == "" || file.Token == "" {
		return File{}, errors.New("discovery file is missing instanceId or bearerToken")
	}
	return file, nil
}

func RemoveIfInstance(path, instanceID string) error {
	file, err := Read(path, ReadOptions{ProtocolVersion: buildinfo.ProtocolVersion, ProcessAlive: func(int) bool { return true }})
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if file.InstanceID != instanceID {
		return nil
	}
	return os.Remove(path)
}

func validateEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.Path != "/mcp" || parsed.RawQuery != "" {
		return ErrInvalidEndpoint
	}
	host := parsed.Hostname()
	if host == "" || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return ErrInvalidEndpoint
	}
	if parsed.Port() == "" {
		return ErrInvalidEndpoint
	}
	return nil
}

func defaultProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
