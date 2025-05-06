package enproto

import (
	"encoding/binary"
	"fmt"
	"io"
)

const maxAllowed uint32 = 100 * 1024 * 1024

func writeFrame(w io.Writer, payload []byte) error {
	// 32-bit length header (big endian). Change to Uint16 for shorter max.
	if err := binary.Write(w, binary.BigEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readFrame(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	// sanity check:
	if length > maxAllowed {
		return nil, fmt.Errorf("frame too large: %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
