package buildinfo

const (
	NativeHostName         = "io.github.yamasaki_masahide_cyg.tabcli"
	ExtensionID            = "ddgfmgclndpdobieomcjaklboinbaoel"
	AllowedExtensionOrigin = "chrome-extension://" + ExtensionID + "/"
	ProfileID              = "default"
	ProtocolVersion        = 3
	MinimumProtocolVersion = 3
	MaximumProtocolVersion = 3
)

// These values are replaced by the reproducible release entrypoint with -ldflags.
var (
	Version = "0.3.0-dev"
	Commit  = "unknown"
	BuiltAt = "unknown"
)
