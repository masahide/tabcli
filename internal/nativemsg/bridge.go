package nativemsg

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/masahide/tabcli/internal/tools"
)

type Bridge struct {
	reader io.Reader
	writer io.Writer

	writeMu   sync.Mutex
	pendingMu sync.Mutex
	pending   map[string]chan Envelope
}

func NewBridge(reader io.Reader, writer io.Writer) *Bridge {
	return &Bridge{reader: reader, writer: writer, pending: make(map[string]chan Envelope)}
}

func (bridge *Bridge) Run(ctx context.Context) error {
	for {
		payload, err := ReadFrame(bridge.reader)
		if err != nil {
			if errors.Is(err, io.EOF) && ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		var envelope Envelope
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return fmt.Errorf("decode Native Messaging envelope: %w", err)
		}
		if err := envelope.Validate(); err != nil {
			return err
		}
		bridge.pendingMu.Lock()
		responseChannel := bridge.pending[envelope.ID]
		bridge.pendingMu.Unlock()
		if responseChannel != nil {
			select {
			case responseChannel <- envelope:
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		if envelope.Operation == OperationHandshake {
			if err := bridge.writeEnvelope(HandshakeResponse(envelope)); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("unexpected Native Messaging operation %q", envelope.Operation)
	}
}

func (bridge *Bridge) Call(ctx context.Context, operation string, input any, output any) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	id, err := randomID()
	if err != nil {
		return err
	}
	responseChannel := make(chan Envelope, 1)
	bridge.pendingMu.Lock()
	bridge.pending[id] = responseChannel
	bridge.pendingMu.Unlock()
	defer func() {
		bridge.pendingMu.Lock()
		delete(bridge.pending, id)
		bridge.pendingMu.Unlock()
	}()
	if err := bridge.writeEnvelope(Envelope{
		ProtocolVersion: ProtocolVersion,
		ID:              id,
		Operation:       operation,
		Payload:         payload,
	}); err != nil {
		return err
	}
	select {
	case response := <-responseChannel:
		if response.Error != nil {
			toolError := tools.NewError(tools.ErrorCode(response.Error.Code), response.Error.Message)
			toolError.Details = response.Error.Details
			return toolError
		}
		if output == nil || len(response.Payload) == 0 {
			return nil
		}
		return json.Unmarshal(response.Payload, output)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bridge *Bridge) writeEnvelope(envelope Envelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	bridge.writeMu.Lock()
	defer bridge.writeMu.Unlock()
	return WriteFrame(bridge.writer, payload)
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
