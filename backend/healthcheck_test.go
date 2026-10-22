package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"healthy", http.StatusOK, false},
		{"unhealthy", http.StatusServiceUnavailable, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/health" {
					t.Errorf("path = %q", r.URL.Path)
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			err := checkHealth(srv.URL+"/api/health", time.Second)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckHealthUnreachable(t *testing.T) {
	// Take a free port and close it again, so nothing listens there.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	if err := checkHealth("http://"+addr+"/api/health", time.Second); err == nil {
		t.Fatal("expected an error")
	}
}

func TestCheckHealthTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	start := time.Now()
	if err := checkHealth(srv.URL+"/api/health", 100*time.Millisecond); err == nil {
		t.Fatal("expected a timeout error")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("took %v, timeout not applied", d)
	}
}

func TestRunHealthcheckNeedsOnlyPort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	// Only BOARDLY_PORT is set; BOARDLY_BASE_URL is missing on purpose.
	lookup := func(k string) (string, bool) {
		if k == "BOARDLY_PORT" {
			return u.Port(), true
		}
		return "", false
	}
	if err := runHealthcheck(lookup); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunHealthcheckInvalidPort(t *testing.T) {
	lookup := func(k string) (string, bool) {
		if k == "BOARDLY_PORT" {
			return "abc", true
		}
		return "", false
	}
	if err := runHealthcheck(lookup); err == nil {
		t.Fatal("expected an error")
	}
}
