package pluginipc

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteFrame writes a single length-prefixed frame.
func WriteFrame(w io.Writer, payload []byte, maxFrameBytes int) error {
	if maxFrameBytes <= 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	if len(payload) > maxFrameBytes {
		return fmt.Errorf("%w: %d > %d", ErrFrameTooLarge, len(payload), maxFrameBytes)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame reads a single length-prefixed frame.
func ReadFrame(r io.Reader, maxFrameBytes int) ([]byte, error) {
	if maxFrameBytes <= 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if int(n) > maxFrameBytes {
		return nil, fmt.Errorf("%w: %d > %d", ErrFrameTooLarge, n, maxFrameBytes)
	}
	if n == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
