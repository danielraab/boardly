package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"boardly/internal/offline"
)

func TestOfflineCopyAPI(t *testing.T) {
	s := newServer(t)
	ann, bob := s.signIn(t, "ann@example.org"), s.signIn(t, "bob@example.org")
	tok := s.createToken(t, ann, nil)
	link := s.create(t, ann, map[string]any{"kind": "link", "title": "Go", "url": "https://go.dev/tour"})
	folder := s.create(t, ann, map[string]any{"kind": "folder", "title": "Work"})
	base := "/api/favourites/" + link.ID + "/offline"

	// Pending right after the create.
	rec := s.req("GET", base, "", withCookie(ann))
	if c := decodeJSONBody[offline.Copy](t, rec); rec.Code != http.StatusOK || c.Status != offline.StatusPending || c.HasArticle {
		t.Fatalf("pending: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"capturedAt":null`) || !strings.Contains(rec.Body.String(), `"error":null`) {
		t.Fatalf("pending JSON = %s", rec.Body)
	}
	if rec := s.req("GET", base+"/article", "", withCookie(ann)); rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
		t.Fatalf("article before the capture: %d %s", rec.Code, rec.Body)
	}

	// A captured copy.
	jpeg := []byte("\xff\xd8\xff\xe0fake")
	if ok, err := s.offline.Record(t.Context(), link.ID, "https://go.dev/tour", offline.Result{
		Title: "A Tour of Go", Article: []byte("<!doctype html><p>Hello</p>"), Screenshot: jpeg,
	}); err != nil || !ok {
		t.Fatalf("record = %v, %v", ok, err)
	}
	rec = s.req("GET", base, "", withCookie(ann))
	c := decodeJSONBody[offline.Copy](t, rec)
	if c.Status != offline.StatusReady || !c.HasArticle || !c.HasScreenshot || c.CapturedAt == nil || *c.Title != "A Tour of Go" {
		t.Fatalf("ready: %s", rec.Body)
	}
	rec = s.req("GET", base+"/article", "", withCookie(ann))
	if rec.Code != http.StatusOK || rec.Body.String() != "<!doctype html><p>Hello</p>" {
		t.Fatalf("article: %d %s", rec.Code, rec.Body)
	}
	for k, want := range map[string]string{
		"Content-Type":            "text/html; charset=utf-8",
		"Content-Security-Policy": "sandbox allow-popups allow-popups-to-escape-sandbox; default-src 'none'; img-src http: https: data:; style-src 'unsafe-inline'; frame-ancestors 'self'",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Cache-Control":           "private, no-cache",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("article %s = %q, want %q", k, got, want)
		}
	}
	rec = s.req("GET", base+"/screenshot", "", withCookie(ann))
	if rec.Code != http.StatusOK || rec.Body.String() != string(jpeg) {
		t.Fatalf("screenshot: %d", rec.Code)
	}
	for k, want := range map[string]string{
		"Content-Type": "image/jpeg", "Content-Security-Policy": "sandbox", "X-Content-Type-Options": "nosniff", "Cache-Control": "private, no-cache",
	} {
		if got := rec.Header().Get(k); got != want {
			t.Errorf("screenshot %s = %q, want %q", k, got, want)
		}
	}

	// Refresh: pending again, the parts stay until the new capture.
	rec = s.req("POST", base, "", withCookie(ann))
	if c := decodeJSONBody[offline.Copy](t, rec); rec.Code != http.StatusAccepted || c.Status != offline.StatusPending || !c.HasArticle {
		t.Fatalf("refresh: %d %s", rec.Code, rec.Body)
	}
	var setPreview bool
	if err := s.db.QueryRow(`SELECT json_extract(payload, '$.setPreview') FROM jobs WHERE kind = 'offline_capture' AND dedup_key = ?`, link.ID).Scan(&setPreview); err != nil || setPreview {
		t.Fatalf("refresh job setPreview = %v, %v", setPreview, err)
	}

	// Only the owner, only links, only with a session.
	for _, path := range []string{base, base + "/article", base + "/screenshot"} {
		if rec := s.req("GET", path, "", withCookie(bob)); rec.Code != http.StatusNotFound {
			t.Errorf("bob GET %s: %d", path, rec.Code)
		}
		if rec := s.req("GET", path, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("signed out GET %s: %d", path, rec.Code)
		}
		if rec := s.req("GET", path, "", withBearer(tok.Secret)); rec.Code != http.StatusUnauthorized {
			t.Errorf("app token GET %s: %d", path, rec.Code)
		}
	}
	for _, id := range []string{folder.ID, "missing"} {
		if rec := s.req("GET", "/api/favourites/"+id+"/offline", "", withCookie(ann)); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: %d", id, rec.Code)
		}
		if rec := s.req("POST", "/api/favourites/"+id+"/offline", "", withCookie(ann)); rec.Code != http.StatusNotFound {
			t.Errorf("POST %s: %d", id, rec.Code)
		}
	}
	if rec := s.req("POST", base, "", withCookie(bob)); rec.Code != http.StatusNotFound {
		t.Errorf("bob POST: %d", rec.Code)
	}
	if rec := s.req("POST", base, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed out POST: %d", rec.Code)
	}
}

func TestOfflineCopyStatusNone(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://go.dev/doc"})
	if _, err := s.db.Exec(`DELETE FROM offline_copies`); err != nil {
		t.Fatal(err)
	}
	rec := s.req("GET", "/api/favourites/"+link.ID+"/offline", "", withCookie(ann))
	if c := decodeJSONBody[offline.Copy](t, rec); rec.Code != http.StatusOK || c.Status != offline.StatusNone {
		t.Fatalf("none: %d %s", rec.Code, rec.Body)
	}
}

func TestOfflineRefreshRateLimit(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	link := s.create(t, ann, map[string]any{"kind": "link", "url": "https://go.dev/doc"})
	path := fmt.Sprintf("/api/favourites/%s/offline", link.ID)
	for i := range offlineRefreshBurst {
		if rec := s.req("POST", path, "", withCookie(ann)); rec.Code != http.StatusAccepted {
			t.Fatalf("refresh %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	rec := s.req("POST", path, "", withCookie(ann))
	if rec.Code != http.StatusTooManyRequests || errorCode(t, rec) != "rate_limited" || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("31st refresh: %d %s", rec.Code, rec.Body)
	}
}
