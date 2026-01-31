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

// TestFramer_WriteFrameBufferedFlush verifies WriteFrameBuffered and Flush work end-to-end.
func TestFramer_WriteFrameBufferedFlush(t *testing.T) {
	buf := &bytes.Buffer{}
	fr := NewFramer(buf)

	msgType := byte(0x2)
	payload := []byte("buffered payload")

	// Write buffered frame (should not appear in underlying buffer yet)
	if err := fr.WriteFrameBuffered(msgType, payload); err != nil {
		t.Fatalf("WriteFrameBuffered error: %v", err)
	}

	// Check that data is buffered but not flushed
	if buf.Len() == 0 {
		// Good, not flushed yet
	} else {
		t.Errorf("expected buffer to be empty before flush, got %d bytes", buf.Len())
	}

	// Flush the buffer
	if err := fr.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// Now read it back
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

// TestFramer_WriteBuffered tests the WriteBuffered method.
func TestFramer_WriteBuffered(t *testing.T) {
	buf := &bytes.Buffer{}
	fr := NewFramer(buf)

	// Initially no buffered data
	if buffered := fr.WriteBuffered(); buffered != 0 {
		t.Errorf("expected 0 buffered bytes initially, got %d", buffered)
	}

	msgType := byte(0x3)
	payload := []byte("test write buffered")

	// Write buffered frame
	if err := fr.WriteFrameBuffered(msgType, payload); err != nil {
		t.Fatalf("WriteFrameBuffered error: %v", err)
	}

	// Check buffered bytes (header is 8 bytes + payload length)
	expectedBuffered := 8 + len(payload)
	if buffered := fr.WriteBuffered(); buffered != expectedBuffered {
		t.Errorf("expected %d buffered bytes, got %d", expectedBuffered, buffered)
	}

	// Flush should clear the buffer
	if err := fr.Flush(); err != nil {
		t.Fatalf("Flush error: %v", err)
	}

	// After flush, should be 0 again (or close to 0)
	if buffered := fr.WriteBuffered(); buffered != 0 {
		t.Errorf("expected 0 buffered bytes after flush, got %d", buffered)
	}
}

// TestFramer_ReadBuffered tests the ReadBuffered method.
func TestFramer_ReadBuffered(t *testing.T) {
	buf := &bytes.Buffer{}
	fr := NewFramer(buf)

	// Initially no buffered data to read
	if buffered := fr.ReadBuffered(); buffered != 0 {
		t.Errorf("expected 0 buffered bytes initially, got %d", buffered)
	}

	// Write multiple frames to the buffer so reader can buffer ahead
	msgType := byte(0x4)
	payload1 := []byte("test read buffered 1")
	payload2 := []byte("test read buffered 2")

	// Frame 1
	header1 := make([]byte, 8)
	binary.BigEndian.PutUint16(header1[0:2], Magic)
	header1[2] = ProtocolVersion
	header1[3] = msgType
	binary.BigEndian.PutUint32(header1[4:8], uint32(len(payload1)))
	buf.Write(header1)
	buf.Write(payload1)

	// Frame 2
	header2 := make([]byte, 8)
	binary.BigEndian.PutUint16(header2[0:2], Magic)
	header2[2] = ProtocolVersion
	header2[3] = msgType + 1
	binary.BigEndian.PutUint32(header2[4:8], uint32(len(payload2)))
	buf.Write(header2)
	buf.Write(payload2)

	// Read the first frame - this should cause the reader to buffer the second frame
	gotType, gotPayload, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if gotType != msgType {
		t.Errorf("message type = %d; want %d", gotType, msgType)
	}

	if !bytes.Equal(gotPayload, payload1) {
		t.Errorf("payload = %q; want %q", gotPayload, payload1)
	}

	// After reading the first frame, the reader should have buffered the second frame
	expectedBuffered := 8 + len(payload2)
	if buffered := fr.ReadBuffered(); buffered != expectedBuffered {
		t.Errorf("expected %d buffered bytes after first read, got %d", expectedBuffered, buffered)
	}

	// Read the second frame
	gotType2, gotPayload2, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("ReadFrame error: %v", err)
	}

	if gotType2 != msgType+1 {
		t.Errorf("message type = %d; want %d", gotType2, msgType+1)
	}

	if !bytes.Equal(gotPayload2, payload2) {
		t.Errorf("payload = %q; want %q", gotPayload2, payload2)
	}

	// After reading everything, buffer should be empty again
	if buffered := fr.ReadBuffered(); buffered != 0 {
		t.Errorf("expected 0 buffered bytes after all reads, got %d", buffered)
	}
}
