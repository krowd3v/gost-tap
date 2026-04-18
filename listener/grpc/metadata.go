package grpc

import (
	"time"

	mdata "github.com/go-gost/core/metadata"
	mdutil "github.com/go-gost/x/metadata/util"
)

const (
	defaultBacklog = 128
)

type metadata struct {
	backlog                      int
	insecure                     bool
	path                         string
	maxStreams                   uint32
	keepalive                    bool
	keepaliveMinTime             time.Duration
	keepaliveTime                time.Duration
	keepaliveTimeout             time.Duration
	keepalivePermitWithoutStream bool
	keepaliveMaxConnectionIdle   time.Duration
	mptcp                        bool
}

func (l *grpcListener) parseMetadata(md mdata.Metadata) (err error) {
	l.md.backlog = mdutil.GetInt(md, "grpc.backlog", "backlog")
	if l.md.backlog <= 0 {
		l.md.backlog = defaultBacklog
	}

	l.md.insecure = mdutil.GetBool(md, "grpc.insecure", "grpcInsecure", "insecure")
	l.md.path = mdutil.GetString(md, "grpc.path", "path")

	// Optional explicit MaxConcurrentStreams. grpc-go default is math.MaxUint32
	// (effectively unlimited) but is NOT advertised in SETTINGS, so clients
	// brief-cap at 100 until their first settings frame arrives. Setting this
	// explicitly makes the server advertise the value, eliminating that
	// transient cap for the first ~RTT after a ClientConn is established.
	if v := mdutil.GetInt(md, "grpc.maxStreams", "maxStreams"); v > 0 {
		l.md.maxStreams = uint32(v)
	}

	l.md.keepalive = mdutil.GetBool(md, "grpc.keepalive", "keepalive", "keepAlive")
	if l.md.keepalive {
		l.md.keepaliveMinTime = mdutil.GetDuration(md, "grpc.keepalive.minTime", "keepalive.minTime")
		if l.md.keepaliveMinTime <= 0 {
			l.md.keepaliveMinTime = 30 * time.Second
		}
		l.md.keepaliveTime = mdutil.GetDuration(md, "grpc.keepalive.time", "keepalive.time")
		if l.md.keepaliveTime <= 0 {
			l.md.keepaliveTime = 60 * time.Second
		}
		l.md.keepaliveTimeout = mdutil.GetDuration(md, "grpc.keepalive.timeout", "keepalive.timeout")
		if l.md.keepaliveTimeout <= 0 {
			l.md.keepaliveTimeout = 30 * time.Second
		}

		l.md.keepalivePermitWithoutStream = mdutil.GetBool(md, "grpc.keepalive.permitWithoutStream", "keepalive.permitWithoutStream")
		l.md.keepaliveMaxConnectionIdle = mdutil.GetDuration(md, "grpc.keepalive.maxConnectionIdle", "keepalive.maxConnectionIdle")
		l.md.mptcp = mdutil.GetBool(md, "mptcp")
	}

	return
}
