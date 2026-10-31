package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"boardly/internal/media"
)

// fakeStreamCheck stands in for the URL check: it resolves listen.pls,
// fails URLs by their names and accepts other https URLs as they are.
func fakeStreamCheck(_ context.Context, rawURL string) (string, error) {
	switch {
	case !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://"):
		return "", media.ErrInvalidURL
	case strings.Contains(rawURL, "http-only"):
		return "", media.ErrNotHTTPS
	case strings.Contains(rawURL, "dead"):
		return "", media.ErrUnreachable
	case strings.HasSuffix(rawURL, "/listen.pls"):
		return "https://edge.example/radio.aac", nil
	}
	return "https://" + strings.TrimPrefix(strings.TrimPrefix(rawURL, "http://"), "https://"), nil
}

func (s *server) createStation(t *testing.T, cookie, title, url string) media.Source {
	t.Helper()
	rec := s.req("POST", "/api/media", mustJSON(map[string]any{"kind": "stream", "title": title, "url": url}), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create %s: %d %s", title, rec.Code, rec.Body)
	}
	return decodeJSONBody[media.Source](t, rec)
}

func (s *server) stationTitles(t *testing.T, cookie string) []string {
	t.Helper()
	rec := s.req("GET", "/api/media", "", withCookie(cookie))
	if rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	var titles []string
	for _, src := range decodeJSONBody[[]media.Source](t, rec) {
		titles = append(titles, src.Title)
	}
	return titles
}

func TestMediaCreateAndList(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	s.createStation(t, ann, "FM4", "https://orf-live.example/fm4")
	pls := s.createStation(t, ann, "Jazz", "https://radio.example/listen.pls")
	if pls.URL != "https://edge.example/radio.aac" || pls.Kind != "stream" || pls.Position != 1 {
		t.Fatalf("source = %+v", pls)
	}
	if got := s.stationTitles(t, ann); strings.Join(got, ",") != "FM4,Jazz" {
		t.Fatalf("titles = %v", got)
	}
	upgraded := s.createStation(t, ann, "Old", "http://twin.example/live")
	if upgraded.URL != "https://twin.example/live" {
		t.Fatalf("url = %q", upgraded.URL)
	}
}

func TestMediaCreateErrors(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	for _, c := range []struct {
		body map[string]any
		code string
	}{
		{map[string]any{"kind": "embed", "title": "X", "url": "https://a.example/"}, "invalid_kind"},
		{map[string]any{"kind": "stream", "title": " ", "url": "https://a.example/"}, "invalid_title"},
		{map[string]any{"kind": "stream", "title": "X", "url": "ftp://a.example/"}, "invalid_url"},
		{map[string]any{"kind": "stream", "title": "X", "url": "http://http-only.example/"}, "stream_not_https"},
		{map[string]any{"kind": "stream", "title": "X", "url": "https://dead.example/"}, "stream_unreachable"},
		{map[string]any{"kind": "stream", "title": "X", "url": "https://a.example/", "icon": map[string]any{"type": "symbol", "name": "nope"}}, "invalid_icon"},
		{map[string]any{"kind": "stream", "title": "X", "url": "https://a.example/", "extra": 1}, "invalid_request"},
	} {
		rec := s.req("POST", "/api/media", mustJSON(c.body), withCookie(ann))
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != c.code {
			t.Errorf("%v: %d %s, want %s", c.body, rec.Code, rec.Body, c.code)
		}
	}
	if got := s.stationTitles(t, ann); len(got) != 0 {
		t.Fatalf("titles = %v", got)
	}
}

func TestMediaUpdateReorderDelete(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	fm4 := s.createStation(t, ann, "FM4", "https://fm4.example/")
	jazz := s.createStation(t, ann, "Jazz", "https://jazz.example/")

	rec := s.req("PATCH", "/api/media/"+fm4.ID, `{"title":"FM4 Wien","icon":{"type":"symbol","name":"music"}}`, withCookie(ann))
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body)
	}
	if got := decodeJSONBody[media.Source](t, rec); got.Title != "FM4 Wien" || got.Icon == nil || got.Icon.Name != "music" {
		t.Fatalf("updated = %+v", got)
	}
	rec = s.req("PATCH", "/api/media/"+fm4.ID, `{"icon":null}`, withCookie(ann))
	if got := decodeJSONBody[media.Source](t, rec); rec.Code != http.StatusOK || got.Icon != nil {
		t.Fatalf("remove icon: %d %s", rec.Code, rec.Body)
	}
	rec = s.req("PATCH", "/api/media/"+fm4.ID, `{"url":"https://dead.example/"}`, withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "stream_unreachable" {
		t.Fatalf("bad url: %d %s", rec.Code, rec.Body)
	}

	rec = s.req("PUT", "/api/media/order", mustJSON([]string{jazz.ID, fm4.ID}), withCookie(ann))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reorder: %d %s", rec.Code, rec.Body)
	}
	if got := s.stationTitles(t, ann); strings.Join(got, ",") != "Jazz,FM4 Wien" {
		t.Fatalf("titles = %v", got)
	}
	rec = s.req("PUT", "/api/media/order", mustJSON([]string{jazz.ID}), withCookie(ann))
	if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "invalid_order" {
		t.Fatalf("incomplete order: %d %s", rec.Code, rec.Body)
	}

	rec = s.req("DELETE", "/api/media/"+jazz.ID, "", withCookie(ann))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body)
	}
	if got := s.stationTitles(t, ann); strings.Join(got, ",") != "FM4 Wien" {
		t.Fatalf("titles = %v", got)
	}
}

func TestMediaOtherUserAndNoSession(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")
	fm4 := s.createStation(t, ann, "FM4", "https://fm4.example/")
	for _, r := range []struct{ method, body string }{{"PATCH", `{"title":"Mine"}`}, {"DELETE", ""}} {
		rec := s.req(r.method, "/api/media/"+fm4.ID, r.body, withCookie(bob))
		if rec.Code != http.StatusNotFound || errorCode(t, rec) != "not_found" {
			t.Errorf("%s by Bob: %d %s", r.method, rec.Code, rec.Body)
		}
	}
	if got := s.stationTitles(t, bob); len(got) != 0 {
		t.Fatalf("Bob sees %v", got)
	}
	if got := s.stationTitles(t, ann); strings.Join(got, ",") != "FM4" {
		t.Fatalf("titles = %v", got)
	}
	if rec := s.req("GET", "/api/media", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
}
