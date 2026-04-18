package local

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"time"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/core/recorder"
)

// recorderConn mirrors every Read/Write on a net.Conn to a recorder,
// one record per I/O call.
//
// Record prefix (context header lines, each emitted only when known):
//
//	sid:<sid>\n               session id (matches the meta record's sid)
//	route:<route>\n           proxy node picked by the chain + target
//	                          e.g. proxy-<hash>@<ip>:<port> > whois.denic.de:43
//	svc:<service>\n           service name (GOST service config name)
//	<direction>[timestamp]\n  direction '<' client -> upstream / '>' upstream -> client
//	                          timestamp per Options.TimestampFormat (optional)
//
// Followed by the payload - raw bytes, or hex.Dump(bytes) when Options.Hexdump.
//
// All of the context lines except the trailing direction/timestamp exist so
// the receiver can join raw-byte records with the meta recorder.service.handler
// JSON and persist complete rows keyed by SID without parsing payload first.
// They are append-only text keys. Parsers should treat recognized header lines
// as preamble and the remaining bytes as payload.
type recorderConn struct {
	net.Conn
	recorder recorder.RecorderObject
	ctx      context.Context
	log      logger.Logger
	required bool
	timeout  time.Duration
	sid      string
	route    string
	service  string
}

// SetRoute updates the cached proxy-node route string. The handler calls this
// after Router.Dial resolves the chain and before forwarding starts.
func (c *recorderConn) SetRoute(route string) {
	c.route = route
}

// writePrefix emits the context header lines + direction/timestamp line.
// The payload is appended by the caller after this returns.
func (c *recorderConn) writePrefix(buf *bytes.Buffer, direction byte) {
	if c.sid != "" {
		buf.WriteString("sid:")
		buf.WriteString(c.sid)
		buf.WriteByte('\n')
	}
	if c.route != "" {
		buf.WriteString("route:")
		buf.WriteString(c.route)
		buf.WriteByte('\n')
	}
	if c.service != "" {
		buf.WriteString("svc:")
		buf.WriteString(c.service)
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
		if err := c.record(buf.Bytes()); err != nil {
			return n, err
		}
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
		if err := c.record(buf.Bytes()); err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(b)
}

func (c *recorderConn) record(b []byte) error {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	if err := c.recorder.Recorder.Record(ctx, b); err != nil {
		if c.log != nil {
			if c.required {
				c.log.Warnf("raw record required: %v", err)
			} else {
				c.log.Debugf("raw record: %v", err)
			}
		}
		if c.required {
			return fmt.Errorf("raw record: %w", err)
		}
	}
	return nil
}
