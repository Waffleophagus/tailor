package server

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHijacker struct {
	http.ResponseWriter
	hijacked bool
}

func (f *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true
	return nil, nil, nil
}

func TestStatusRecorderForwardsHijack(t *testing.T) {
	underlying := &fakeHijacker{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}

	hj, ok := any(rec).(http.Hijacker)
	if !ok {
		t.Fatal("statusRecorder should implement http.Hijacker")
	}
	if _, _, err := hj.Hijack(); err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	if !underlying.hijacked {
		t.Fatal("expected underlying Hijack to be called")
	}
}

func TestStatusRecorderUnwrapReturnsUnderlyingWriter(t *testing.T) {
	underlying := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}
	if unwrapped, ok := any(rec).(interface{ Unwrap() http.ResponseWriter }); ok {
		if unwrapped.Unwrap() != underlying {
			t.Fatal("Unwrap should return the underlying writer")
		}
	}
}

func TestStatusRecorderHijackUnsupported(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	if _, ok := any(rec).(http.Hijacker); !ok {
		t.Fatal("statusRecorder should implement http.Hijacker")
	}
	if _, _, err := rec.Hijack(); err == nil {
		t.Fatal("expected error when underlying writer is not hijackable")
	}
}
