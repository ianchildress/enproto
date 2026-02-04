package enproto

import (
	"bufio"
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

var header [8]byte

var (
	ErrBadMagic   = errors.New("invalid magic number")
	ErrBadVersion = errors.New("unsupported protocol version")
)

// Framer handles our length‐prefixed, versioned frames.
type Framer struct {
	br *bufio.Reader
	bw *bufio.Writer

	rbuf []byte // reusable read payload buffer
}

func NewFramer(rw io.ReadWriter) *Framer {
	return &Framer{
		br: bufio.NewReaderSize(rw, 64*1024),
		bw: bufio.NewWriterSize(rw, 64*1024), // 64KB buffer
	}
}

// WriteFrame writes a frame and flushes immediately (compat behavior).
func (f *Framer) WriteFrame(msgType byte, payload []byte) error {
	if err := f.WriteFrameBuffered(msgType, payload); err != nil {
		return err
	}
	return f.bw.Flush()
}

// WriteFrameBuffered writes a frame to the internal buffer.
// Call Flush to ensure data is sent to the underlying writer.
func (f *Framer) WriteFrameBuffered(msgType byte, payload []byte) error {
	var header [8]byte
	binary.BigEndian.PutUint16(header[0:2], Magic)
	header[2] = ProtocolVersion
	header[3] = msgType
	binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))

	if _, err := f.bw.Write(header[:]); err != nil {
		return err
	}
	if _, err := f.bw.Write(payload); err != nil {
		return err
	}

	return nil
}

// Flush flushes the buffered writer.
func (f *Framer) Flush() error {
	return f.bw.Flush()
}

// ReadFrame reads the next frame, validates header, and returns msgType + payload.
func (f *Framer) ReadFrame() (msgType byte, payload []byte, err error) {
	var header [8]byte
	if _, err = io.ReadFull(f.br, header[:]); err != nil {
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
	if _, err = io.ReadFull(f.br, payload); err != nil {
		return 0, nil, err
	}
	return msgType, payload, nil
}

// ReadFrameSharedBuffer reads the next frame, validates header, and returns msgType + payload.
// NOTE: payload is backed by an internal reusable buffer and is only valid until
// the next ReadFrameSharedBuffer call on this Framer.
func (f *Framer) ReadFrameSharedBuffer() (msgType byte, payload []byte, err error) {
	var header [8]byte
	if _, err = io.ReadFull(f.br, header[:]); err != nil {
		return 0, nil, err
	}

	if magic := binary.BigEndian.Uint16(header[0:2]); magic != Magic {
		return 0, nil, ErrBadMagic
	}
	if version := header[2]; version != ProtocolVersion {
		return 0, nil, ErrBadVersion
	}
	msgType = header[3]

	length := int(binary.BigEndian.Uint32(header[4:8]))
	if uint32(length) > maxAllowed {
		return 0, nil, fmt.Errorf("frame too large: %d", length)
	}

	// Ensure reusable buffer is large enough.
	if cap(f.rbuf) < length {
		// Grow to at least length; optionally over-allocate to reduce future grows.
		// This is a normal heap allocation but happens rarely (only when size increases).
		newCap := cap(f.rbuf) * 2
		if newCap < length {
			newCap = length
		}
		f.rbuf = make([]byte, newCap)
	}

	payload = f.rbuf[:length]
	if _, err = io.ReadFull(f.br, payload); err != nil {
		return 0, nil, err
	}
	return msgType, payload, nil
}

// WriteBuffered returns the number of bytes currently queued in the write buffer.
func (f *Framer) WriteBuffered() int {
	if f.bw == nil {
		return 0
	}
	return f.bw.Buffered()
}

// ReadBuffered returns the number of bytes currently buffered and ready to be read
// without reading from the underlying io.Reader.
func (f *Framer) ReadBuffered() int {
	if f.br == nil {
		return 0
	}
	return f.br.Buffered()
}
