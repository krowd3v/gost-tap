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
// Record prefix (lines, optional — each part appears only when configured):
//
//	sid:<sid>\n               when sid is non-empty (always included here)
//	<direction>[timestamp]\n  direction: '<' client -> upstream, '>' upstream -> client
//	                          timestamp formatted per Options.TimestampFormat
//
// Followed by the payload - raw bytes, or hex.Dump(bytes) when Options.Hexdump.
//
// The SID prefix exists so the receiver can join raw-byte records with the
// parent recorder.service.handler JSON (which contains the same sid).
// Without it, correlation relies on time-window + proxy-node heuristics.
type recorderConn struct {
	net.Conn
	recorder recorder.RecorderObject
	sid      string
}

func newRecorderConn(c net.Conn, rec recorder.RecorderObject, sid string) net.Conn {
	return &recorderConn{Conn: c, recorder: rec, sid: sid}
}

// writePrefix emits the optional sid/direction/timestamp header lines
// into buf. Returns the buffer after preamble so the caller can append payload.
func (c *recorderConn) writePrefix(buf *bytes.Buffer, direction byte) {
	if c.sid != "" {
		buf.WriteString("sid:")
		buf.WriteString(c.sid)
		buf.WriteByte('\n')
	}
	var hasMeta bool
	if c.recorder.Options != nil && c.recorder.Options.Direction {
		buf.WriteByte(direction)
		hasMeta = true
	}
	if c.recorder.Options != nil && c.recorder.Options.TimestampFormat != "" {
		buf.WriteString(time.Now().Format(c.recorder.Options.TimestampFormat))
		hasMeta = true
	}
	if hasMeta {
		buf.WriteByte('\n')
	}
}

func (c *recorderConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)

	if n > 0 && c.recorder.Recorder != nil {
		var buf bytes.Buffer
		c.writePrefix(&buf, '<')
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
		c.writePrefix(&buf, '>')
		if c.recorder.Options != nil && c.recorder.Options.Hexdump {
			buf.WriteString(hex.Dump(b))
		} else {
			buf.Write(b)
		}
		c.recorder.Recorder.Record(context.Background(), buf.Bytes())
	}
	return c.Conn.Write(b)
}
