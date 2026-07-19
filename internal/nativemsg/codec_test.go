package nativemsg

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestCodecRoundTrip(t *testing.T) {
	want := []byte(`{"protocolVersion":3,"id":"request-1","operation":"handshake"}`)
	var stream bytes.Buffer

	if err := WriteFrame(&stream, want); err != nil {
		t.Fatalf("WriteFrame() error = %v", err)
	}

	got, err := ReadFrame(&stream)
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("ReadFrame() = %q, want %q", got, want)
	}
}

func TestReadFrameRejectsInvalidLength(t *testing.T) {
	tests := []struct {
		name   string
		length uint32
	}{
		{name: "empty", length: 0},
		{name: "larger than native messaging limit", length: MaxMessageSize + 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stream bytes.Buffer
			if err := binary.Write(&stream, binary.LittleEndian, tt.length); err != nil {
				t.Fatal(err)
			}

			_, err := ReadFrame(&stream)
			if !errors.Is(err, ErrInvalidFrameLength) {
				t.Fatalf("ReadFrame() error = %v, want %v", err, ErrInvalidFrameLength)
			}
		})
	}
}

func TestReadFrameReportsDisconnect(t *testing.T) {
	t.Run("before header", func(t *testing.T) {
		_, err := ReadFrame(bytes.NewReader(nil))
		if !errors.Is(err, io.EOF) {
			t.Fatalf("ReadFrame() error = %v, want EOF", err)
		}
	})

	t.Run("during payload", func(t *testing.T) {
		var stream bytes.Buffer
		if err := binary.Write(&stream, binary.LittleEndian, uint32(10)); err != nil {
			t.Fatal(err)
		}
		stream.WriteString("short")

		_, err := ReadFrame(&stream)
		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("ReadFrame() error = %v, want unexpected EOF", err)
		}
	})
}
