package enproto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Magic           uint16 = 0x5959
	ProtocolVersion byte   = 1

	// 100 MiB
	maxAllowed uint32 = 100 * 1024 * 1024
)

var (
	ErrBadMagic   = errors.New("invalid magic number")
	ErrBadVersion = errors.New("unsupported protocol version")
)

// Framer handles our length‐prefixed, versioned frames.
type Framer struct {
	rw io.ReadWriter
}

func NewFramer(rw io.ReadWriter) *Framer {
	return &Framer{rw: rw}
}

// WriteFrame sends a message of the given type.
func (f *Framer) WriteFrame(msgType byte, payload []byte) error {
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], Magic)
	header[2] = ProtocolVersion
	header[3] = msgType
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))

	if _, err := f.rw.Write(header); err != nil {
		return err
	}
	_, err := f.rw.Write(payload)
	return err
}

// ReadFrame reads the next frame, validates header, and returns msgType + payload.
func (f *Framer) ReadFrame() (msgType byte, payload []byte, err error) {
	header := make([]byte, 8)
	if _, err = io.ReadFull(f.rw, header); err != nil {
		return 0, nil, err
	}

	if magic := binary.BigEndian.Uint16(header[0:2]); magic != Magic {
		return 0, nil, ErrBadMagic
	}
	if version := header[2]; version != ProtocolVersion {
		return 0, nil, ErrBadVersion
	}
	msgType = header[3]

	length := binary.BigEndian.Uint32(header[4:8])
	if length > maxAllowed {
		return 0, nil, fmt.Errorf("frame too large: %d", length)
	}

	payload = make([]byte, length)
	if _, err = io.ReadFull(f.rw, payload); err != nil {
		return 0, nil, err
	}
	return msgType, payload, nil
}
