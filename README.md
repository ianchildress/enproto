# enproto

A lightweight, versioned binary framing protocol in Go.

## Features

* **Magic number & version validation**: Ensures you’re speaking the right protocol and version.
* **Message-type multiplexing**: Route different payloads via a single stream.
* **Fixed-size length prefix**: 32-bit big-endian length header with a 100 MiB cap.
* **Easy-to-use `Framer` wrapper**: Encapsulates framing logic around any `io.ReadWriter`.\*\*: Encapsulates framing logic around any `io.ReadWriter`.

## Installation

```bash
go get github.com/ianchildress/enproto
```

## Usage

```go
import (
    "fmt"
    "net"

    "github.com/<your-username>/enproto"
)

func main() {
    // Dial a TCP connection
    conn, err := net.Dial("tcp", "example.com:1234")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    fr := enproto.NewFramer(conn)

    // 1) Send a message (type 0x01)
    payload := []byte("Hello, Protocol!")
    if err := fr.WriteFrame(0x01, payload); err != nil {
        panic(err)
    }

    // 2) Read the next frame
    msgType, data, err := fr.ReadFrame()
    if err != nil {
        panic(err)
    }

    switch msgType {
    case 0x01:
        fmt.Println("Received:", string(data))
    default:
        fmt.Printf("Unknown message type: %d\n", msgType)
    }
}
```

## API Reference

### Constants

* `Magic uint16` – Protocol magic number (`0xABCD`).
* `ProtocolVersion byte` – Current wire-format version.
* `maxAllowed uint32` – Maximum payload size (100 MiB).

### Types & Functions

```go
// Framer handles framing over any io.ReadWriter.
type Framer struct { /* ... */ }

// NewFramer wraps an io.ReadWriter with our framing logic.
func NewFramer(rw io.ReadWriter) *Framer

// WriteFrame writes a message type + length-prefixed payload.
func (f *Framer) WriteFrame(msgType byte, payload []byte) error

// ReadFrame reads and validates a frame, returning the message type and payload.
func (f *Framer) ReadFrame() (msgType byte, payload []byte, err error)
```

