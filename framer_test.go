package enproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
)

// TestFramer_WriteReadFrame verifies that WriteFrame and ReadFrame work end-to-end.
func TestFramer_WriteReadFrame(t *testing.T) {
	buf := &bytes.Buffer{}
	fr := NewFramer(buf)

	msgType := byte(0x1)
	payload := []byte("test payload")

	// Write a frame
	if err := fr.WriteFrame(msgType, payload); err != nil {
		t.Fatalf("WriteFrame error: %v", err)
	}

	// Read it back
	gotType, gotPayload, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if gotType != msgType {
		t.Errorf("message type = %d; want %d", gotType, msgType)
	}

	if !bytes.Equal(gotPayload, payload) {
		t.Errorf("payload = %q; want %q", gotPayload, payload)
	}
}

// TestFramer_ReadFrame_TooLarge ensures ReadFrame rejects payloads exceeding maxAllowed.
func TestFramer_ReadFrame_TooLarge(t *testing.T) {
	buf := &bytes.Buffer{}
	// Construct header: magic, version, type, oversized length
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], Magic)
	header[2] = ProtocolVersion
	header[3] = 0x1
	binary.BigEndian.PutUint32(header[4:8], maxAllowed+1)
	buf.Write(header)

	fr := NewFramer(buf)
	_, _, err := fr.ReadFrame()
	if err == nil || !strings.Contains(err.Error(), "frame too large") {
		t.Errorf("expected frame too large error, got %v", err)
	}
}

// TestFramer_ReadFrame_BadMagic ensures ReadFrame errors on an invalid magic number.
func TestFramer_ReadFrame_BadMagic(t *testing.T) {
	buf := &bytes.Buffer{}
	// Wrong magic, correct version
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], 0xFFFF)
	header[2] = ProtocolVersion
	header[3] = 0x1
	binary.BigEndian.PutUint32(header[4:8], 0)
	buf.Write(header)

	fr := NewFramer(buf)
	_, _, err := fr.ReadFrame()
	if !errors.Is(err, ErrBadMagic) {
		t.Errorf("expected ErrBadMagic, got %v", err)
	}
}

// TestFramer_ReadFrame_BadVersion ensures ReadFrame errors on an unsupported version.
func TestFramer_ReadFrame_BadVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	// Correct magic, wrong version
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], Magic)
	header[2] = ProtocolVersion + 1
	header[3] = 0x1
	binary.BigEndian.PutUint32(header[4:8], 0)
	buf.Write(header)

	fr := NewFramer(buf)
	_, _, err := fr.ReadFrame()
	if !errors.Is(err, ErrBadVersion) {
		t.Errorf("expected ErrBadVersion, got %v", err)
	}
}
