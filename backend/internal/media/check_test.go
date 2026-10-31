package media

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// routes serves the path → handler pairs for host stream.test: https on port
// 443 and, when plain is set, http on port 80.
type routes struct {
	https map[string]http.HandlerFunc
	plain map[string]http.HandlerFunc
}

func audio(contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte("audio bytes"))
	}
}

func text(contentType, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}
}

func redirect(to string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, to, http.StatusFound) }
}

func mux(handlers map[string]http.HandlerFunc) http.Handler {
	m := http.NewServeMux()
	for p, h := range handlers {
		m.Handle(p, h)
	}
	return m
}

// newChecker returns a checker whose requests to *.test go to local servers.
func newChecker(t *testing.T, r routes) *Checker {
	t.Helper()
	secure := httptest.NewTLSServer(mux(r.https))
	t.Cleanup(secure.Close)
	plain := httptest.NewServer(mux(r.plain))
	t.Cleanup(plain.Close)
	dialer := &net.Dialer{}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			target := secure.Listener.Addr().String()
			if port == "80" {
				target = plain.Listener.Addr().String()
			}
			return dialer.DialContext(ctx, network, target)
		},
	}
	return &Checker{Client: &http.Client{Transport: transport}, Timeout: 5 * time.Second}
}

func TestCheck(t *testing.T) {
	c := newChecker(t, routes{
		https: map[string]http.HandlerFunc{
			"/live.mp3":       audio("audio/mpeg"),
			"/radio.aac":      audio("audio/aac; charset=utf-8"),
			"/listen.pls":     text("audio/x-scpls", "[playlist]\nNumberOfEntries=2\nFile1=https://stream.test/radio.aac\nFile2=https://stream.test/live.mp3\n"),
			"/listen.m3u":     text("text/plain", "#EXTM3U\n#EXTINF:-1,Radio\n/live.mp3\n"),
			"/hls.m3u8":       text("application/vnd.apple.mpegurl", "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=128000\nlow.m3u8\n"),
			"/http-only.pls":  text("audio/x-scpls", "File1=http://old.test/stream\n"),
			"/dead.pls":       text("audio/x-scpls", "File1=https://stream.test/missing\n"),
			"/nested.m3u":     text("audio/x-mpegurl", "https://stream.test/listen.pls\n"),
			"/page":           text("text/html", "<html></html>"),
			"/to-http":        redirect("http://stream.test/live.mp3"),
			"/to-https":       redirect("https://stream.test/live.mp3"),
			"/old.test/":      http.NotFound,
			"/ogg":            audio("application/ogg"),
			"/octet.pls":      text("application/octet-stream", "File1=https://stream.test/live.mp3\n"),
			"/twin/live.mp3":  audio("audio/mpeg"),
			"/missing-twin/x": http.NotFound,
		},
		plain: map[string]http.HandlerFunc{
			"/": audio("audio/mpeg"),
		},
	})
	ok := []struct{ in, want string }{
		{"https://stream.test/live.mp3", "https://stream.test/live.mp3"},
		{"  https://stream.test/radio.aac ", "https://stream.test/radio.aac"},
		{"http://stream.test/twin/live.mp3", "https://stream.test/twin/live.mp3"},
		{"https://stream.test/listen.pls", "https://stream.test/radio.aac"},
		{"https://stream.test/listen.m3u", "https://stream.test/live.mp3"},
		{"https://stream.test/hls.m3u8", "https://stream.test/hls.m3u8"},
		{"https://stream.test/to-https", "https://stream.test/to-https"},
		{"https://stream.test/ogg", "https://stream.test/ogg"},
		{"https://stream.test/octet.pls", "https://stream.test/live.mp3"},
	}
	for _, c2 := range ok {
		got, err := c.Check(context.Background(), c2.in)
		if err != nil || got != c2.want {
			t.Errorf("%s: got %q, %v; want %q", c2.in, got, err, c2.want)
		}
	}
	bad := []struct {
		in   string
		want error
	}{
		{"ftp://stream.test/live.mp3", ErrInvalidURL},
		{"stream.test/live.mp3", ErrInvalidURL},
		{"https://", ErrInvalidURL},
		{"http://stream.test/missing-twin/x", ErrNotHTTPS},
		{"https://stream.test/http-only.pls", ErrNotHTTPS},
		{"https://stream.test/to-http", ErrNotHTTPS},
		{"https://stream.test/page", ErrUnreachable},
		{"https://stream.test/missing", ErrUnreachable},
		{"https://stream.test/dead.pls", ErrUnreachable},
		{"https://stream.test/nested.m3u", ErrUnreachable},
	}
	for _, b := range bad {
		if _, err := c.Check(context.Background(), b.in); !errors.Is(err, b.want) {
			t.Errorf("%s: err = %v, want %v", b.in, err, b.want)
		}
	}
}

func TestCheckBlocksPrivateAddresses(t *testing.T) {
	var hit bool
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		audio("audio/mpeg")(w, r)
	}))
	defer srv.Close()
	c := NewChecker(false)
	if _, err := c.Check(context.Background(), srv.URL+"/live.mp3"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
	if hit {
		t.Fatal("the server on a loopback address was contacted")
	}
}

func TestCheckTimeout(t *testing.T) {
	c := newChecker(t, routes{https: map[string]http.HandlerFunc{
		"/slow": func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() },
	}})
	c.Timeout = 100 * time.Millisecond
	start := time.Now()
	_, err := c.Check(context.Background(), "https://stream.test/slow")
	if !errors.Is(err, ErrUnreachable) || time.Since(start) > 2*time.Second {
		t.Fatalf("err = %v after %v", err, time.Since(start))
	}
}

func TestPlaylistEntries(t *testing.T) {
	got := playlistEntries("pls", []byte("\ufeff[playlist]\nfile1 = https://a.test/1\nTitle1=A\nFile2=https://a.test/2\n"))
	if strings.Join(got, " ") != "https://a.test/1 https://a.test/2" {
		t.Fatalf("pls = %v", got)
	}
	got = playlistEntries("m3u", []byte("#EXTM3U\r\n\r\n#EXTINF:-1,A\r\nhttps://a.test/1\r\n"))
	if strings.Join(got, " ") != "https://a.test/1" {
		t.Fatalf("m3u = %v", got)
	}
}
