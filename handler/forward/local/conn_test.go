package local

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/go-gost/core/recorder"
)

type failingRecorder struct{}

func (failingRecorder) Record(context.Context, []byte, ...recorder.RecordOption) error {
	return errors.New("sink down")
}

func TestRecorderConnRequiredRecordFailureStopsWrite(t *testing.T) {
	client, upstream := net.Pipe()
	defer client.Close()
	defer upstream.Close()

	conn := &recorderConn{
		Conn: client,
		recorder: recorder.RecorderObject{
			Recorder: failingRecorder{},
		},
		required: true,
	}

	if _, err := conn.Write([]byte("response")); err == nil {
		t.Fatal("Write error = nil, want raw recorder failure")
	}
}

func TestRecorderConnRequiredRecordFailureStopsAfterRead(t *testing.T) {
	client, upstream := net.Pipe()
	defer client.Close()
	defer upstream.Close()

	conn := &recorderConn{
		Conn: client,
		recorder: recorder.RecorderObject{
			Recorder: failingRecorder{},
		},
		required: true,
	}

	go func() {
		_, _ = upstream.Write([]byte("request"))
	}()

	buf := make([]byte, 7)
	n, err := conn.Read(buf)
	if err == nil {
		t.Fatal("Read error = nil, want raw recorder failure")
	}
	if n != 7 {
		t.Fatalf("Read n = %d, want 7", n)
	}
}
