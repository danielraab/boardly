package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeDirectory serves a server list and a search, and records requests.
type fakeDirectory struct {
	mu       sync.Mutex
	requests []*http.Request
	servers  int // status of /json/servers
	search   int // status of the search
}

func (f *fakeDirectory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.requests = append(f.requests, r.Clone(context.Background()))
	f.mu.Unlock()
	switch r.URL.Path {
	case "/json/servers":
		if f.servers != 0 {
			w.WriteHeader(f.servers)
			return
		}
		_, _ = w.Write([]byte(`[{"name":"at1.api.radio-browser.info"}]`))
	case "/json/stations/search":
		if f.search != 0 {
			w.WriteHeader(f.search)
			return
		}
		_, _ = w.Write([]byte(`[
			{"stationuuid":"a","name":" FM4 ","url":"http://fm4.example/pls","url_resolved":"https://fm4.example/live","favicon":"https://fm4.example/logo.png","country":"Austria","tags":"alternative, indie,,rock,pop","bitrate":128},
			{"stationuuid":"b","name":"","url":"http://old.example/stream","favicon":" ","country":"","tags":"","bitrate":0}
		]`))
	default:
		http.NotFound(w, r)
	}
}

func newTestDirectory(t *testing.T, f *fakeDirectory) (*Directory, *[]string) {
	t.Helper()
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	var hosts []string
	d := NewDirectory(true)
	d.Client = srv.Client()
	d.ServersURL = srv.URL + "/json/servers"
	d.MirrorURL = func(host string) string {
		hosts = append(hosts, host)
		return srv.URL
	}
	return d, &hosts
}

func TestDirectorySearch(t *testing.T) {
	f := &fakeDirectory{}
	d, hosts := newTestDirectory(t, f)
	list, err := d.Search(context.Background(), "fm4")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("got %d stations", len(list))
	}
	fm4, old := list[0], list[1]
	if fm4.ID != "a" || fm4.Name != "FM4" || fm4.URL != "https://fm4.example/live" || !fm4.Playable ||
		fm4.LogoURL == nil || *fm4.LogoURL != "https://fm4.example/logo.png" ||
		fm4.Country == nil || *fm4.Country != "Austria" || fm4.Bitrate == nil || *fm4.Bitrate != 128 ||
		strings.Join(fm4.Tags, ",") != "alternative,indie,rock" {
		t.Fatalf("fm4 = %+v", fm4)
	}
	if old.Name != "http://old.example/stream" || old.Playable || old.LogoURL != nil || old.Country != nil ||
		old.Bitrate != nil || len(old.Tags) != 0 {
		t.Fatalf("old = %+v", old)
	}
	if strings.Join(*hosts, ",") != "at1.api.radio-browser.info" {
		t.Fatalf("mirrors = %v", *hosts)
	}

	search := f.requests[len(f.requests)-1]
	q := search.URL.Query()
	if q.Get("name") != "fm4" || q.Get("limit") != "30" || q.Get("order") != "clickcount" ||
		q.Get("reverse") != "true" || q.Get("hidebroken") != "true" {
		t.Fatalf("query = %v", q)
	}
	for _, r := range f.requests {
		if !strings.HasPrefix(r.UserAgent(), "Boardly") {
			t.Fatalf("user agent = %q", r.UserAgent())
		}
		if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("Cookie") != "" {
			t.Fatalf("forwarded headers: %v", r.Header)
		}
	}
}

func TestDirectoryCachesMirrors(t *testing.T) {
	f := &fakeDirectory{}
	d, _ := newTestDirectory(t, f)
	now := time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)
	d.Now = func() time.Time { return now }
	count := func() (n int) {
		for _, r := range f.requests {
			if r.URL.Path == "/json/servers" {
				n++
			}
		}
		return n
	}
	for range 2 {
		if _, err := d.Search(context.Background(), "fm4"); err != nil {
			t.Fatal(err)
		}
	}
	if count() != 1 {
		t.Fatalf("server list loaded %d times", count())
	}
	now = now.Add(2 * time.Hour)
	if _, err := d.Search(context.Background(), "fm4"); err != nil {
		t.Fatal(err)
	}
	if count() != 2 {
		t.Fatalf("server list loaded %d times after an hour", count())
	}
}

func TestDirectoryFallback(t *testing.T) {
	d, hosts := newTestDirectory(t, &fakeDirectory{servers: http.StatusInternalServerError})
	if _, err := d.Search(context.Background(), "fm4"); err != nil {
		t.Fatal(err)
	}
	if strings.Join(*hosts, ",") != "de1.api.radio-browser.info" {
		t.Fatalf("mirrors = %v", *hosts)
	}
}

func TestDirectoryUnreachable(t *testing.T) {
	d, _ := newTestDirectory(t, &fakeDirectory{search: http.StatusServiceUnavailable})
	if _, err := d.Search(context.Background(), "fm4"); !errors.Is(err, ErrDirectoryUnreachable) {
		t.Fatalf("err = %v", err)
	}
}
