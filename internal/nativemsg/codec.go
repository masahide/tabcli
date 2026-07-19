package nativemsg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const MaxMessageSize uint32 = 1024 * 1024

var ErrInvalidFrameLength = errors.New("invalid Native Messaging frame length")

func ReadFrame(reader io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 || length > MaxMessageSize {
		return nil, fmt.Errorf("%w: %d", ErrInvalidFrameLength, length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func WriteFrame(writer io.Writer, payload []byte) error {
	length := len(payload)
	if length == 0 || uint64(length) > uint64(MaxMessageSize) {
		return fmt.Errorf("%w: %d", ErrInvalidFrameLength, length)
	}
	if err := binary.Write(writer, binary.LittleEndian, uint32(length)); err != nil {
		return err
	}
	_, err := writer.Write(payload)
	return err
}
