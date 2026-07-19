package nativemsg

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestEnvelopeVersionAndHandshake(t *testing.T) {
	requestPayload, _ := json.Marshal(map[string]any{"extensionVersion": "0.1.0", "profileId": "default"})
	request := Envelope{ProtocolVersion: ProtocolVersion, ID: "handshake-1", Operation: OperationHandshake, Payload: requestPayload}
	if err := request.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	response := HandshakeResponse(request)
	if response.ID != request.ID || response.ProtocolVersion != ProtocolVersion {
		t.Fatalf("HandshakeResponse() = %#v", response)
	}
	var payload struct {
		Accepted               bool   `json:"accepted"`
		HostVersion            string `json:"hostVersion"`
		ProfileID              string `json:"profileId"`
		MinimumProtocolVersion int    `json:"minimumProtocolVersion"`
		MaximumProtocolVersion int    `json:"maximumProtocolVersion"`
		UpdateInstructions     string `json:"updateInstructions"`
	}
	if err := json.Unmarshal(response.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Accepted || payload.HostVersion == "" || payload.ProfileID != "default" || payload.MinimumProtocolVersion > ProtocolVersion || payload.MaximumProtocolVersion < ProtocolVersion || payload.UpdateInstructions == "" {
		t.Fatalf("handshake payload = %#v", payload)
	}
}

func TestHandshakeRejectsAnotherProfile(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"extensionVersion": "0.1.0", "profileId": "other"})
	response := HandshakeResponse(Envelope{ProtocolVersion: ProtocolVersion, ID: "handshake-2", Operation: OperationHandshake, Payload: payload})
	var result struct {
		Accepted bool `json:"accepted"`
	}
	if err := json.Unmarshal(response.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Accepted {
		t.Fatal("host accepted a non-default profile")
	}
}

func TestEnvelopeRejectsProtocolMismatch(t *testing.T) {
	err := (Envelope{ProtocolVersion: ProtocolVersion + 1, ID: "1", Operation: "snapshot"}).Validate()
	if !errors.Is(err, ErrProtocolVersionMismatch) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrProtocolVersionMismatch)
	}
}
