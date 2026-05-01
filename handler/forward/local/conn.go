package local

import (
	"bytes"
	"context"
	"encoding/hex"
	"net"
	"time"

	"github.com/go-gost/core/recorder"
)

type recorderConn struct {
	net.Conn
	recorder recorder.RecorderObject
}

func (c *recorderConn) record(ctx context.Context, direction byte, b []byte) {
	if len(b) == 0 || c.recorder.Recorder == nil {
		return
	}

	var buf bytes.Buffer
	if c.recorder.Options != nil && c.recorder.Options.Direction {
		buf.WriteByte(direction)
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

	_ = c.recorder.Recorder.Record(ctx, buf.Bytes())
}

func (c *recorderConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	c.record(context.Background(), '>', b[:n])
	return
}

func (c *recorderConn) Write(b []byte) (int, error) {
	c.record(context.Background(), '<', b)
	return c.Conn.Write(b)
}
