package local

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/go-gost/core/recorder"
)

type captureRecorder struct {
	records [][]byte
	err     error
}

func (r *captureRecorder) Record(_ context.Context, b []byte, _ ...recorder.RecordOption) error {
	r.records = append(r.records, append([]byte(nil), b...))
	return r.err
}

func TestRecorderConnRecordsPayload(t *testing.T) {
	client, upstream := net.Pipe()
	defer client.Close()
	defer upstream.Close()

	rec := &captureRecorder{}
	conn := &recorderConn{
		Conn: client,
		recorder: recorder.RecorderObject{
			Recorder: rec,
			Options:  &recorder.Options{Direction: true},
		},
	}

	go func() {
		buf := make([]byte, 7)
		_, _ = upstream.Read(buf)
		_, _ = upstream.Write([]byte("reply"))
	}()

	if _, err := conn.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := conn.Read(buf); err != nil {
		t.Fatal(err)
	}

	if got := string(rec.records[0]); got != "<\npayload" {
		t.Fatalf("write record = %q, want %q", got, "<\npayload")
	}
	if got := string(rec.records[1]); got != ">\nreply" {
		t.Fatalf("read record = %q, want %q", got, ">\nreply")
	}
}

func TestRecorderConnRecordFailureDoesNotStopWrite(t *testing.T) {
	client, upstream := net.Pipe()
	defer client.Close()
	defer upstream.Close()

	conn := &recorderConn{
		Conn: client,
		recorder: recorder.RecorderObject{
			Recorder: &captureRecorder{err: errors.New("sink down")},
		},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 7)
		_, _ = upstream.Read(buf)
	}()

	if _, err := conn.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	<-done
}
