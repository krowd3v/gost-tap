package local

import (
	"bytes"
	"context"
	"encoding/hex"
	"net"
	"time"

	"github.com/go-gost/core/recorder"
)

// recorderConn mirrors every Read/Write on a net.Conn to a recorder,
// one record per I/O call.
//
// Output format per record (all parts optional, controlled by recorder Options):
//
	//	<direction byte>   '<' = bytes read from client    (client -> target)
	//	                   '>' = bytes written to client   (target -> client)
//	<timestamp>\n      formatted per Options.TimestampFormat
//	<payload>          raw bytes, or hex.Dump if Options.Hexdump is set
//
// When no Options are configured, records contain just the payload bytes.
//
// This mirrors the pattern in handler/serial/conn.go but applies to the
// generic TCP forwarder so WHOIS / DNS-over-TCP / any plaintext protocol
// can be captured without protocol-specific sniffing.
type recorderConn struct {
	net.Conn
	recorder recorder.RecorderObject
}

func newRecorderConn(c net.Conn, rec recorder.RecorderObject) net.Conn {
	return &recorderConn{Conn: c, recorder: rec}
}

func (c *recorderConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 && c.recorder.Recorder != nil {
		var buf bytes.Buffer
		if c.recorder.Options != nil && c.recorder.Options.Direction {
				buf.WriteByte('<')
		}
		if c.recorder.Options != nil && c.recorder.Options.TimestampFormat != "" {
			buf.WriteString(time.Now().Format(c.recorder.Options.TimestampFormat))
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		if c.recorder.Options != nil && c.recorder.Options.Hexdump {
			buf.WriteString(hex.Dump(b[:n]))
		} else {
			buf.Write(b[:n])
		}
		c.recorder.Recorder.Record(context.Background(), buf.Bytes())
	}

	return
}

func (c *recorderConn) Write(b []byte) (int, error) {
	if c.recorder.Recorder != nil {
		var buf bytes.Buffer
		if c.recorder.Options != nil && c.recorder.Options.Direction {
				buf.WriteByte('>')
		}
		if c.recorder.Options != nil && c.recorder.Options.TimestampFormat != "" {
			buf.WriteString(time.Now().Format(c.recorder.Options.TimestampFormat))
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		if c.recorder.Options != nil && c.recorder.Options.Hexdump {
			buf.WriteString(hex.Dump(b))
		} else {
			buf.Write(b)
		}
		c.recorder.Recorder.Record(context.Background(), buf.Bytes())
	}
	return c.Conn.Write(b)
}
