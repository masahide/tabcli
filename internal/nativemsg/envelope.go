package nativemsg

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/masahide/tabcli/internal/buildinfo"
)

const ProtocolVersion = 3

const OperationHandshake = "handshake"

var ErrProtocolVersionMismatch = errors.New("PROTOCOL_VERSION_MISMATCH")

type ProtocolError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Envelope struct {
	ProtocolVersion int             `json:"protocolVersion"`
	ID              string          `json:"id"`
	Operation       string          `json:"operation"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	Error           *ProtocolError  `json:"error,omitempty"`
}

func (e Envelope) Validate() error {
	if e.ProtocolVersion != ProtocolVersion {
		return fmt.Errorf("%w: got %d, want %d", ErrProtocolVersionMismatch, e.ProtocolVersion, ProtocolVersion)
	}
	if e.ID == "" || e.Operation == "" {
		return errors.New("Native Messaging envelope requires id and operation")
	}
	return nil
}

func HandshakeResponse(request Envelope) Envelope {
	var handshake struct {
		ExtensionVersion string `json:"extensionVersion"`
		ProfileID        string `json:"profileId"`
	}
	accepted := json.Unmarshal(request.Payload, &handshake) == nil && handshake.ExtensionVersion != "" && handshake.ProfileID == buildinfo.ProfileID
	payload, _ := json.Marshal(map[string]any{
		"accepted":               accepted,
		"hostVersion":            buildinfo.Version,
		"profileId":              buildinfo.ProfileID,
		"minimumProtocolVersion": buildinfo.MinimumProtocolVersion,
		"maximumProtocolVersion": buildinfo.MaximumProtocolVersion,
		"updateInstructions":     "Update the extension and tabcli together, then restart Chrome.",
	})
	return Envelope{
		ProtocolVersion: ProtocolVersion,
		ID:              request.ID,
		Operation:       OperationHandshake,
		Payload:         payload,
	}
}
