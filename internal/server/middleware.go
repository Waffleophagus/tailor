package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

type contextKey int

const requestIDKey contextKey = iota

// RequestIDFromContext returns the request ID attached by AccessMiddleware, if any.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack forwards to the underlying ResponseWriter so WebSocket handlers work.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.ResponseWriter does not implement http.Hijacker")
	}
	return h.Hijack()
}

// Flush forwards to the underlying ResponseWriter when supported.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying writer for middleware feature detection.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// AccessMiddleware logs API requests with method, path, status, duration, and request ID.
func AccessMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		id := newRequestID()
		ctx := withRequestID(r.Context(), id)
		r = r.WithContext(ctx)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		status := rec.status
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", id,
		}

		switch {
		case r.URL.Path == "/api/health":
			logger.Debug("api request", attrs...)
		case status >= 500:
			logger.Error("api request", attrs...)
		case status >= 400:
			logger.Warn("api request", attrs...)
		default:
			logger.Info("api request", attrs...)
		}
	})
}

func logAPIError(logger *slog.Logger, r *http.Request, status int, err error, msg string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"status", status,
		"path", r.URL.Path,
		"method", r.Method,
		"request_id", RequestIDFromContext(r.Context()),
	}
	if msg != "" {
		attrs = append(attrs, "message", msg)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	switch {
	case status >= 500:
		logger.Error("api handler error", attrs...)
	case status == http.StatusUnauthorized:
		logger.Debug("api handler error", attrs...)
	default:
		logger.Warn("api handler error", attrs...)
	}
}
