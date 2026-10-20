package aaguid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/providers.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func aaguid(s string) []byte {
	id := uuid.MustParse(s)
	return id[:]
}

const gpm = "ea9b8d66-4d01-1d21-3ce4-b6b48cb575d4"

func loaded(t *testing.T) *Directory {
	t.Helper()
	d := New("", filepath.Join(t.TempDir(), "aaguids.json"))
	if err := os.WriteFile(d.Path, fixture(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.Load(); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLookup(t *testing.T) {
	d := loaded(t)

	p := d.Lookup(aaguid(gpm))
	if p == nil || p.Name != "Google Password Manager" {
		t.Fatalf("Lookup(GPM) = %+v", p)
	}
	if p.IconLight == nil || *p.IconLight != "data:image/svg+xml;base64,bGlnaHQ=" || p.IconDark == nil || *p.IconDark != "data:image/svg+xml;base64,ZGFyaw==" {
		t.Fatalf("icons = %v, %v", p.IconLight, p.IconDark)
	}

	// Upper-case keys in the list are normalized.
	if p := d.Lookup(aaguid("fbfc3007-154e-4ecc-8c0b-6e020557d7bd")); p == nil || p.Name != "iCloud Keychain" || p.IconLight != nil {
		t.Fatalf("Lookup(iCloud) = %+v", p)
	}
	// Icons that aren't data: image URIs are dropped.
	if p := d.Lookup(aaguid("bada5566-a7aa-401f-bd96-45619a55120d")); p == nil || p.IconLight != nil {
		t.Fatalf("Lookup(1Password) = %+v", p)
	}

	for name, id := range map[string][]byte{
		"all zero": make([]byte, 16),
		"unknown":  aaguid("00000000-0000-0000-0000-000000000001"),
		"short":    {1, 2, 3},
		"nil":      nil,
	} {
		if p := d.Lookup(id); p != nil {
			t.Errorf("Lookup(%s) = %+v, want nil", name, p)
		}
	}

	var none *Directory
	if none.Lookup(aaguid(gpm)) != nil {
		t.Fatal("nil directory must know no providers")
	}
}

func TestNormalize(t *testing.T) {
	for _, in := range []string{gpm, "EA9B8D66-4D01-1D21-3CE4-B6B48CB575D4", "ea9b8d664d011d213ce4b6b48cb575d4"} {
		if got, err := normalize(in); err != nil || got != gpm {
			t.Errorf("normalize(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := normalize("not-a-uuid"); err == nil {
		t.Error("expected an error")
	}
}

type upstream struct {
	srv      *httptest.Server
	body     []byte
	status   int
	requests atomic.Int32
	lastINM  atomic.Value
}

func newUpstream(t *testing.T, body []byte) *upstream {
	u := &upstream{body: body, status: http.StatusOK}
	u.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.requests.Add(1)
		u.lastINM.Store(r.Header.Get("If-None-Match"))
		if r.Header.Get("If-None-Match") == `"v1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"v1"`)
		w.WriteHeader(u.status)
		_, _ = w.Write(u.body)
	}))
	t.Cleanup(u.srv.Close)
	return u
}

func TestRefreshFreshDownload(t *testing.T) {
	up := newUpstream(t, fixture(t))
	d := New(up.srv.URL, filepath.Join(t.TempDir(), "data", "aaguids.json"))

	if err := d.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if d.Lookup(aaguid(gpm)) == nil {
		t.Fatal("provider not known after download")
	}
	if etag, err := os.ReadFile(d.Path + ".etag"); err != nil || string(etag) != `"v1"` {
		t.Fatalf("etag = %q, %v", etag, err)
	}
	// A restart picks up the cached copy.
	again := New(up.srv.URL, d.Path)
	if err := again.Load(); err != nil || again.Lookup(aaguid(gpm)) == nil {
		t.Fatalf("cached copy not loaded: %v", err)
	}
}

func TestRefreshNotModified(t *testing.T) {
	up := newUpstream(t, fixture(t))
	d := New(up.srv.URL, filepath.Join(t.TempDir(), "aaguids.json"))
	if err := d.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.Stat(d.Path)

	up.body = []byte(`{"broken":`) // would fail validation if it were sent
	if err := d.Refresh(context.Background()); err != nil {
		t.Fatalf("304: %v", err)
	}
	if got := up.lastINM.Load(); got != `"v1"` {
		t.Fatalf("If-None-Match = %v", got)
	}
	if d.Lookup(aaguid(gpm)) == nil {
		t.Fatal("list lost after 304")
	}
	after, _ := os.Stat(d.Path)
	if after.Size() != before.Size() {
		t.Fatal("cache file changed on 304")
	}
}

func TestRefreshMalformedKeepsCache(t *testing.T) {
	d := loaded(t)
	cached, _ := os.ReadFile(d.Path)

	for name, body := range map[string]string{
		"not JSON":   `<html>rate limited</html>`,
		"wrong type": `["a", "b"]`,
		"empty":      `{}`,
		"bad key":    `{"nope": {"name": "X"}}`,
		"no name":    `{"` + gpm + `": {"icon_light": "data:image/png;base64,AA=="}}`,
	} {
		t.Run(name, func(t *testing.T) {
			up := newUpstream(t, []byte(body))
			d.URL = up.srv.URL
			if err := d.Refresh(context.Background()); err == nil {
				t.Fatal("expected an error")
			}
			if d.Lookup(aaguid(gpm)) == nil {
				t.Fatal("in-memory list replaced")
			}
			if now, _ := os.ReadFile(d.Path); string(now) != string(cached) {
				t.Fatal("cache file replaced")
			}
		})
	}
}

func TestRefreshSizeCap(t *testing.T) {
	up := newUpstream(t, fixture(t))
	d := New(up.srv.URL, filepath.Join(t.TempDir(), "aaguids.json"))
	d.MaxSize = 100
	if err := d.Refresh(context.Background()); err == nil {
		t.Fatal("expected an error for an oversized list")
	}
	if _, err := os.Stat(d.Path); !os.IsNotExist(err) {
		t.Fatal("oversized list was cached")
	}
}

func TestUnreachableWithoutCache(t *testing.T) {
	up := newUpstream(t, nil)
	url := up.srv.URL
	up.srv.Close()

	d := New(url, filepath.Join(t.TempDir(), "aaguids.json"))
	if err := d.Load(); err != nil {
		t.Fatalf("Load without cache: %v", err)
	}
	if err := d.Refresh(context.Background()); err == nil {
		t.Fatal("expected a download error")
	}
	if d.Len() != 0 || d.Lookup(aaguid(gpm)) != nil {
		t.Fatal("expected an empty list")
	}
}

func TestRunSkipsFreshCache(t *testing.T) {
	up := newUpstream(t, fixture(t))
	d := loaded(t)
	d.URL = up.srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { d.Run(ctx); close(done) }()
	cancel()
	<-done
	if n := up.requests.Load(); n != 0 {
		t.Fatalf("requests = %d, want 0 for a fresh cache", n)
	}
	if d.Lookup(aaguid(gpm)) == nil {
		t.Fatal("cache not loaded")
	}
}

func TestRunRefreshesStaleCache(t *testing.T) {
	up := newUpstream(t, []byte(`{"`+gpm+`": {"name": "Renamed Manager"}}`))
	d := loaded(t)
	d.URL = up.srv.URL
	old := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(d.Path, old, old); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx)

	deadline := time.Now().Add(5 * time.Second)
	for {
		if p := d.Lookup(aaguid(gpm)); p != nil && p.Name == "Renamed Manager" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("stale list was not refreshed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
