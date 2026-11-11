package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"boardly/internal/feeds"
)

func feedHost(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head></head><body>none</body></html>`))
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>` + strings.TrimPrefix(r.URL.Path, "/") + `</title>` +
			`<item><guid>1</guid><title>One</title><link>https://a.example/1</link></item></channel></rss>`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func (s *server) addFeed(t *testing.T, cookie, url string) feeds.Subscription {
	t.Helper()
	rec := s.req("POST", "/api/feeds", mustJSON(map[string]any{"url": url}), withCookie(cookie))
	if rec.Code != http.StatusCreated {
		t.Fatalf("add %s: %d %s", url, rec.Code, rec.Body)
	}
	return decodeJSONBody[feeds.Subscription](t, rec)
}

type feedPage struct {
	Items      []feeds.Item `json:"items"`
	NextCursor *string      `json:"nextCursor"`
}

func TestFeedsAPI(t *testing.T) {
	s := newServer(t)
	host := feedHost(t)
	ann := s.signIn(t, "ann@example.org")
	bob := s.signIn(t, "bob@example.org")

	if rec := s.req("GET", "/api/feeds", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}
	if rec := s.req("GET", "/api/feed-items", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no session: %d", rec.Code)
	}

	first := s.addFeed(t, ann, host.URL+"/first")
	second := s.addFeed(t, ann, host.URL+"/second")

	for name, c := range map[string]struct {
		method, path, body, cookie string
		status                     int
		code                       string
	}{
		"bad scheme":    {"POST", "/api/feeds", `{"url":"ftp://x.example/"}`, ann, 400, "invalid_url"},
		"no url":        {"POST", "/api/feeds", `{}`, ann, 400, "invalid_url"},
		"unknown field": {"POST", "/api/feeds", `{"url":"https://x.example/","x":1}`, ann, 400, "invalid_request"},
		"not a feed":    {"POST", "/api/feeds", `{"url":"` + host.URL + `/page"}`, ann, 400, "feed_not_found"},
		"unreachable":   {"POST", "/api/feeds", `{"url":"http://127.0.0.1:1/feed"}`, ann, 400, "feed_unreachable"},
		"duplicate":     {"POST", "/api/feeds", `{"url":"` + host.URL + `/first"}`, ann, 409, "already_subscribed"},
		"foreign patch": {"PATCH", "/api/feeds/" + first.ID, `{"customTitle":"x"}`, bob, 404, "not_found"},
		"foreign del":   {"DELETE", "/api/feeds/" + first.ID, ``, bob, 404, "not_found"},
		"long title":    {"PATCH", "/api/feeds/" + first.ID, `{"customTitle":"` + strings.Repeat("x", 201) + `"}`, ann, 400, "invalid_title"},
		"bad order":     {"PUT", "/api/feeds/order", `["` + first.ID + `"]`, ann, 400, "invalid_order"},
		"bad view":      {"GET", "/api/feed-items?view=all", ``, ann, 400, "invalid_request"},
		"bad limit":     {"GET", "/api/feed-items?limit=101", ``, ann, 400, "invalid_request"},
		"nan limit":     {"GET", "/api/feed-items?limit=x", ``, ann, 400, "invalid_request"},
		"bad cursor":    {"GET", "/api/feed-items?cursor=!", ``, ann, 400, "invalid_request"},
		"long search":   {"GET", "/api/feed-items?q=" + strings.Repeat("x", 201), ``, ann, 400, "invalid_request"},
	} {
		rec := s.req(c.method, c.path, c.body, withCookie(c.cookie))
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.code) {
			t.Errorf("%s: %d %s", name, rec.Code, rec.Body)
		}
	}

	if rec := s.req("PUT", "/api/feeds/order", mustJSON([]string{second.ID, first.ID}), withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("order: %d %s", rec.Code, rec.Body)
	}
	list := decodeJSONBody[[]feeds.Subscription](t, s.req("GET", "/api/feeds", "", withCookie(ann)))
	if len(list) != 2 || list[0].ID != second.ID {
		t.Fatalf("%+v", list)
	}

	rec := s.req("PATCH", "/api/feeds/"+first.ID, `{"customTitle":"Mine"}`, withCookie(ann))
	if rec.Code != 200 || decodeJSONBody[feeds.Subscription](t, rec).Title != "Mine" {
		t.Fatalf("rename: %d %s", rec.Code, rec.Body)
	}
	bobs := s.addFeed(t, bob, host.URL+"/first")
	if bobs.Title != "first" || bobs.FeedID != first.FeedID {
		t.Fatalf("Bob's: %+v", bobs)
	}

	page := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?feedId="+first.ID, "", withCookie(ann)))
	if len(page.Items) != 1 || page.Items[0].Feed.Title != "Mine" || page.NextCursor != nil {
		t.Fatalf("%+v", page)
	}
	id := page.Items[0].ID
	if page := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?q=ONE&feedId="+first.ID, "", withCookie(ann))); len(page.Items) != 1 {
		t.Fatalf("search: %+v", page)
	}
	if page := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?q=two", "", withCookie(ann))); len(page.Items) != 0 {
		t.Fatalf("search without match: %+v", page)
	}
	if rec := s.req("GET", "/api/feed-items?q="+strings.Repeat("ä", 200), "", withCookie(ann)); rec.Code != 200 {
		t.Fatalf("200 characters: %d", rec.Code)
	}
	if page := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?feedId="+first.ID, "", withCookie(bob))); len(page.Items) != 0 {
		t.Fatalf("Ann's feed as Bob's filter: %+v", page)
	}

	rec = s.req("PATCH", "/api/feed-items/"+id, `{"liked":true}`, withCookie(ann))
	if rec.Code != 200 || decodeJSONBody[feeds.Item](t, rec).LikedAt == nil {
		t.Fatalf("like: %d %s", rec.Code, rec.Body)
	}
	if rec := s.req("PATCH", "/api/feed-items/"+id, `{"liked":null}`, withCookie(ann)); rec.Code != 400 {
		t.Fatalf("null flag: %d", rec.Code)
	}
	liked := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?view=liked", "", withCookie(bob)))
	if len(liked.Items) != 0 {
		t.Fatal("Ann's like shows for Bob")
	}
	// Bob has no subscription to the feed of "second", so its item is hidden.
	other := decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?feedId="+second.ID, "", withCookie(ann)))
	if rec := s.req("PATCH", "/api/feed-items/"+other.Items[0].ID, `{"liked":true}`, withCookie(bob)); rec.Code != 404 {
		t.Fatalf("foreign item: %d", rec.Code)
	}

	if rec := s.req("DELETE", "/api/feeds/"+first.ID, "", withCookie(ann)); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: %d", rec.Code)
	}
	liked = decodeJSONBody[feedPage](t, s.req("GET", "/api/feed-items?view=liked", "", withCookie(ann)))
	if len(liked.Items) != 1 || liked.Items[0].Feed.SubscriptionID != nil {
		t.Fatalf("saved item after removing the feed: %+v", liked)
	}
}

func TestFeedAddIsRateLimited(t *testing.T) {
	s := newServer(t)
	ann := s.signIn(t, "ann@example.org")
	var rec *httptest.ResponseRecorder
	for range feedAddBurst + 1 {
		rec = s.req("POST", "/api/feeds", `{"url":"http://127.0.0.1:1/x"}`, withCookie(ann))
	}
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
}
