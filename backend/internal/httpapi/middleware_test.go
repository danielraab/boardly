package httpapi

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

// captureLog sends the default logger to a buffer for the rest of the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// requestLines returns the request log lines in buf.
func requestLines(buf *bytes.Buffer) []string {
	var lines []string
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, "msg=request ") {
			lines = append(lines, l)
		}
	}
	return lines
}

// statusHandler answers /200, /404 and /500 with that status.
var statusHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/404":
		w.WriteHeader(http.StatusNotFound)
	case "/500":
		w.WriteHeader(http.StatusInternalServerError)
	}
})

func TestLogRequestsModes(t *testing.T) {
	tests := []struct {
		name, mode string
		want       []string
	}{
		{"all", "all", []string{"status=200", "status=404", "status=500"}},
		{"unset", "", []string{"status=200", "status=404", "status=500"}},
		{"errors", "errors", []string{"status=500"}},
		{"none", "none", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureLog(t)
			h := logRequests(tt.mode, statusHandler)
			for _, path := range []string{"/200", "/404", "/500"} {
				do(t, h, http.MethodGet, path)
			}
			lines := requestLines(buf)
			if len(lines) != len(tt.want) {
				t.Fatalf("got %d request lines, want %d:\n%s", len(lines), len(tt.want), buf)
			}
			for i, want := range tt.want {
				if !strings.Contains(lines[i], want) {
					t.Fatalf("line %d = %q, want %s", i, lines[i], want)
				}
			}
		})
	}
}

func TestLogRequestsWebAppFiles(t *testing.T) {
	buf := captureLog(t)
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb, LogRequests: "all"})
	do(t, h, http.MethodGet, "/")
	do(t, h, http.MethodGet, "/assets/index-abc123.js")
	if lines := requestLines(buf); len(lines) != 2 {
		t.Fatalf("got %d request lines, want 2:\n%s", len(lines), buf)
	}
}

func TestLogRequestsSkipsHealthyHealthCheck(t *testing.T) {
	buf := captureLog(t)
	h := New(Deps{DB: fakeDB{}, Web: fakeWeb, LogRequests: "all"})
	do(t, h, http.MethodGet, "/api/health")
	if lines := requestLines(buf); len(lines) != 0 {
		t.Fatalf("healthy health check was logged:\n%s", buf)
	}
}

func TestLogRequestsFailingHealthCheck(t *testing.T) {
	for _, mode := range []string{"all", "errors"} {
		t.Run(mode, func(t *testing.T) {
			buf := captureLog(t)
			h := New(Deps{DB: fakeDB{err: errors.New("closed")}, Web: fakeWeb, LogRequests: mode})
			do(t, h, http.MethodGet, "/api/health")
			lines := requestLines(buf)
			if len(lines) != 1 || !strings.Contains(lines[0], "status=503") {
				t.Fatalf("want one request line with status 503:\n%s", buf)
			}
		})
	}
}

func TestLogRequestsNoneKeepsOtherLogs(t *testing.T) {
	buf := captureLog(t)
	h := logRequests("none", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.ErrorContext(r.Context(), "request failed", "error", "boom")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	do(t, h, http.MethodGet, "/x")
	if lines := requestLines(buf); len(lines) != 0 {
		t.Fatalf("request was logged:\n%s", buf)
	}
	if !strings.Contains(buf.String(), `msg="request failed"`) {
		t.Fatalf("handler's own log line is missing:\n%s", buf)
	}
}
