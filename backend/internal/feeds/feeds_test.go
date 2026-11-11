package feeds

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/dbtest"
)

var ctx = context.Background()

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) add(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

type env struct {
	t     *testing.T
	s     *Store
	db    *sql.DB
	clock *clock
}

func newEnv(t *testing.T, keep int) *env {
	t.Helper()
	database := dbtest.Open(t)
	c := &clock{t: time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)}
	s := NewStore(database, NewFetcher(true), nil, time.Hour, keep)
	s.Now = c.now
	s.Jitter = func(time.Duration) time.Duration { return 0 }
	return &env{t: t, s: s, db: database, clock: c}
}

func (e *env) user(name string) string {
	e.t.Helper()
	id := uuid.NewString()
	if _, err := e.db.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		id, name+"@example.org", "2026-01-01T00:00:00.000Z", "2026-01-01T00:00:00.000Z"); err != nil {
		e.t.Fatal(err)
	}
	return id
}

func (e *env) count(query string, args ...any) int {
	e.t.Helper()
	var n int
	if err := e.db.QueryRow(query, args...).Scan(&n); err != nil {
		e.t.Fatal(err)
	}
	return n
}

// entry is one item of a served feed.
type entry struct {
	guid, title string
	age         time.Duration
}

// feedServer serves an RSS document that tests can change.
type feedServer struct {
	*httptest.Server
	mu       sync.Mutex
	entries  []entry
	hits     atomic.Int32
	status   int
	etag     string
	base     time.Time
	redirect string
	// summaries holds a description per guid.
	summaries map[string]string
}

func serveFeed(t *testing.T, base time.Time, entries ...entry) *feedServer {
	t.Helper()
	f := &feedServer{entries: entries, base: base, status: http.StatusOK}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Close)
	return f
}

func (f *feedServer) set(entries ...entry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = entries
}

func (f *feedServer) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/favicon.ico" {
		http.NotFound(w, r)
		return
	}
	f.hits.Add(1)
	if f.redirect != "" {
		http.Redirect(w, r, f.redirect, http.StatusMovedPermanently)
		return
	}
	if f.status != http.StatusOK {
		w.WriteHeader(f.status)
		return
	}
	if f.etag != "" {
		if r.Header.Get("If-None-Match") == f.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", f.etag)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><rss version="2.0"><channel><title>Example</title><link>https://site.example/</link>`)
	for _, e := range f.entries {
		fmt.Fprintf(&b, `<item><guid>%s</guid><title>%s</title><pubDate>%s</pubDate><description>%s</description></item>`,
			e.guid, e.title, f.base.Add(-e.age).Format(time.RFC1123Z), f.summaries[e.guid])
	}
	b.WriteString(`</channel></rss>`)
	w.Header().Set("Content-Type", "application/rss+xml")
	_, _ = w.Write([]byte(b.String()))
}

func titles(items []Item) string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return strings.Join(out, ",")
}

func (e *env) items(user, view string) []Item {
	e.t.Helper()
	items, _, err := e.s.Items(ctx, user, ItemQuery{View: view, Limit: 100})
	if err != nil {
		e.t.Fatal(err)
	}
	return items
}

func (e *env) subscribe(user, rawURL string) Subscription {
	e.t.Helper()
	sub, err := e.s.Subscribe(ctx, user, rawURL)
	if err != nil {
		e.t.Fatal(err)
	}
	return sub
}

func (e *env) flag(user, item string, readLater, liked *bool) {
	e.t.Helper()
	if _, err := e.s.SetFlags(ctx, user, item, readLater, liked); err != nil {
		e.t.Fatal(err)
	}
}

func ptr[T any](v T) *T { return &v }

func TestUniqueConstraints(t *testing.T) {
	e := newEnv(t, 100)
	insert := `INSERT INTO feeds (id, url, title, next_fetch_at, created_at) VALUES (?, 'https://a.example/f', 't', 'x', 'x')`
	if _, err := e.db.Exec(insert, "f1"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(insert, "f2"); err == nil {
		t.Fatal("a second feed with the same URL was stored")
	}
	item := `INSERT INTO feed_items (id, feed_id, identity, title, item_time, first_seen_at) VALUES (?, 'f1', 'g', 't', 'x', 'x')`
	if _, err := e.db.Exec(item, "i1"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(item, "i2"); err == nil {
		t.Fatal("a second item with the same identity was stored")
	}
}

func TestTwoUsersShareOneFeed(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	before := srv.hits.Load()
	sub := e.subscribe(bob, srv.URL+"/feed")
	if srv.hits.Load() != before {
		t.Fatal("a known feed was fetched again")
	}
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatalf("%d feeds", n)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 1 {
		t.Fatalf("%d items", n)
	}
	if sub.Title != "Example" || titles(e.items(bob, ViewLatest)) != "One" {
		t.Fatalf("sub %+v, items %s", sub, titles(e.items(bob, ViewLatest)))
	}
	if _, err := e.s.Subscribe(ctx, bob, srv.URL+"/feed"); !errors.Is(err, ErrAlreadySubscribed) {
		t.Fatalf("err = %v", err)
	}
}

func TestConcurrentAddsCreateOneFeed(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	var wg sync.WaitGroup
	for i := range 6 {
		user := e.user(fmt.Sprintf("u%d", i))
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := e.s.Subscribe(ctx, user, srv.URL+"/feed"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatalf("%d feeds", n)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_subscriptions`); n != 6 {
		t.Fatalf("%d subscriptions", n)
	}
}

func TestRedirectFindsTheSameFeed(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	redirector := httptest.NewServer(http.RedirectHandler(srv.URL+"/feed", http.StatusFound))
	defer redirector.Close()
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, redirector.URL+"/old")
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatalf("%d feeds", n)
	}
}

func TestSubscribeErrors(t *testing.T) {
	e := newEnv(t, 100)
	ann := e.user("ann")
	if _, err := e.s.Subscribe(ctx, ann, "ftp://example.org/"); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("err = %v", err)
	}
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>x</title></head><body>hi</body></html>`))
	}))
	defer page.Close()
	if _, err := e.s.Subscribe(ctx, ann, page.URL); !errors.Is(err, ErrNotAFeed) {
		t.Fatalf("err = %v", err)
	}
	dead := httptest.NewServer(http.NotFoundHandler())
	defer dead.Close()
	if _, err := e.s.Subscribe(ctx, ann, dead.URL); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v", err)
	}
}

func TestPrivateAddressIsBlocked(t *testing.T) {
	e := newEnv(t, 100)
	e.s.Fetcher = NewFetcher(false)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	if _, err := e.s.Subscribe(ctx, e.user("ann"), srv.URL+"/feed"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v", err)
	}
	if srv.hits.Load() != 0 {
		t.Fatal("connected to a private address")
	}
}

func TestDiscovery(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	page := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><link rel="stylesheet" href="a.css"><link rel="alternate" type="application/rss+xml" href="%s/feed"></head></html>`, srv.URL)
	}))
	defer page.Close()
	sub := e.subscribe(e.user("ann"), page.URL)
	want, _ := normalizeURL(srv.URL + "/feed")
	if sub.URL != want {
		t.Fatalf("url = %s", sub.URL)
	}
}

func TestRefreshKeepsIDAndFlags(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, srv.URL+"/feed")
	first := e.items(ann, ViewLatest)[0]
	e.flag(ann, first.ID, ptr(true), nil)

	srv.set(entry{"1", "One renamed", time.Hour}, entry{"2", "Two", time.Minute})
	e.clock.add(2 * time.Hour)
	if n, err := e.s.RefreshDue(ctx); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	items := e.items(ann, ViewLatest)
	if titles(items) != "Two,One renamed" || items[1].ID != first.ID || items[1].ReadLaterAt == nil {
		t.Fatalf("%+v", items)
	}
	if len(e.items(bob, ViewReadLater)) != 0 || e.items(bob, ViewLatest)[1].ReadLaterAt != nil {
		t.Fatal("Ann's flag leaked to Bob")
	}
}

func TestRefreshLengthensSummary(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	if _, err := e.db.Exec(`UPDATE feed_items SET summary = ?`, strings.Repeat("a", 500)); err != nil {
		t.Fatal(err)
	}

	srv.mu.Lock()
	srv.summaries = map[string]string{"1": strings.Repeat("a", 1500)}
	srv.mu.Unlock()
	e.clock.add(2 * time.Hour)
	if _, err := e.s.RefreshDue(ctx); err != nil {
		t.Fatal(err)
	}
	items := e.items(ann, ViewLatest)
	if len(items) != 1 || items[0].Summary == nil || len(*items[0].Summary) != maxSummaryLength {
		t.Fatalf("%+v", items)
	}
}

func TestConditionalRequestAndInterval(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	srv.etag = `"v1"`
	e.subscribe(e.user("ann"), srv.URL+"/feed")
	hits := srv.hits.Load()
	e.clock.add(30 * time.Minute)
	if n, _ := e.s.RefreshDue(ctx); n != 0 || srv.hits.Load() != hits {
		t.Fatal("refreshed before the cycle was over")
	}
	e.clock.add(31 * time.Minute)
	if n, _ := e.s.RefreshDue(ctx); n != 1 {
		t.Fatal("not refreshed after the cycle")
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 1 {
		t.Fatalf("%d items", n)
	}
	var next string
	_ = e.db.QueryRow(`SELECT next_fetch_at FROM feeds`).Scan(&next)
	if want := db.FormatTime(e.clock.now().Add(time.Hour)); next != want {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func TestNoSubscribersNoRefresh(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann := e.user("ann")
	sub := e.subscribe(ann, srv.URL+"/feed")
	if err := e.s.Unsubscribe(ctx, ann, sub.ID); err != nil {
		t.Fatal(err)
	}
	e.clock.add(3 * time.Hour)
	if n, _ := e.s.RefreshDue(ctx); n != 0 {
		t.Fatal("refreshed a feed without subscribers")
	}
}

func TestBackoffAndFailing(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann := e.user("ann")
	sub := e.subscribe(ann, srv.URL+"/feed")
	srv.status = http.StatusInternalServerError
	for i, wait := range []time.Duration{2 * time.Hour, 4 * time.Hour, 8 * time.Hour} {
		var next string
		_ = e.db.QueryRow(`SELECT next_fetch_at FROM feeds`).Scan(&next)
		due, _ := db.ParseTime(next)
		e.clock.t = due
		if n, _ := e.s.RefreshDue(ctx); n != 1 {
			t.Fatalf("round %d: not refreshed", i)
		}
		_ = e.db.QueryRow(`SELECT next_fetch_at FROM feeds`).Scan(&next)
		if want := db.FormatTime(due.Add(wait)); next != want {
			t.Fatalf("round %d: next = %s, want %s", i, next, want)
		}
	}
	got, err := e.s.Get(ctx, ann, sub.ID)
	if err != nil || !got.Failing {
		t.Fatalf("%+v %v", got, err)
	}
	if len(e.items(ann, ViewLatest)) != 1 {
		t.Fatal("items were lost")
	}
}

func TestRetryAfterIsHonoured(t *testing.T) {
	e := newEnv(t, 100)
	calls := atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/rss+xml")
			_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>x</title><item><guid>1</guid><title>a</title></item></channel></rss>`))
			return
		}
		w.Header().Set("Retry-After", "86400")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	e.subscribe(e.user("ann"), srv.URL)
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	var next string
	_ = e.db.QueryRow(`SELECT next_fetch_at FROM feeds`).Scan(&next)
	if want := db.FormatTime(e.clock.now().Add(24 * time.Hour)); next != want {
		t.Fatalf("next = %s, want %s", next, want)
	}
}

func many(n int) []entry {
	out := make([]entry, n)
	for i := range out {
		out[i] = entry{fmt.Sprint(i), fmt.Sprintf("T%d", i), time.Duration(i+1) * time.Minute}
	}
	return out
}

func TestOnlyNewestEntriesAreStored(t *testing.T) {
	e := newEnv(t, 3)
	srv := serveFeed(t, e.clock.now(), many(10)...)
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	if got := titles(e.items(ann, ViewLatest)); got != "T0,T1,T2" {
		t.Fatal(got)
	}
	var ids string
	_ = e.db.QueryRow(`SELECT group_concat(id) FROM (SELECT id FROM feed_items ORDER BY id)`).Scan(&ids)
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	var after string
	_ = e.db.QueryRow(`SELECT group_concat(id) FROM (SELECT id FROM feed_items ORDER BY id)`).Scan(&after)
	if ids != after {
		t.Fatal("refreshing replaced the stored items")
	}
}

func TestCleanup(t *testing.T) {
	e := newEnv(t, 2)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", 3 * time.Minute}, entry{"2", "B", 2 * time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, srv.URL+"/feed")
	b := e.items(ann, ViewLatest)[1] // the older one, "A"
	if b.Title != "A" {
		t.Fatal(b.Title)
	}
	e.flag(ann, b.ID, ptr(true), ptr(true))
	e.flag(bob, b.ID, nil, ptr(true))

	srv.set(entry{"3", "C", time.Minute}, entry{"2", "B", 2 * time.Minute})
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE title = 'A'`); n != 1 {
		t.Fatal("a flagged item was deleted")
	}
	if len(e.items(ann, ViewLiked)) != 1 {
		t.Fatal("flagged item left the view")
	}
	// Bob's flag keeps it after Ann removes hers.
	e.flag(ann, b.ID, ptr(false), ptr(false))
	if _, err := e.s.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE title = 'A'`); n != 1 {
		t.Fatal("deleted an item another user flagged")
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_item_flags WHERE user_id = ?`, ann); n != 0 {
		t.Fatal("a flags row without flags stayed")
	}
	e.flag(bob, b.ID, nil, ptr(false))
	_, _ = e.s.Cleanup(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE title = 'A'`); n != 0 {
		t.Fatal("an unflagged old item stayed")
	}
}

func TestLoweredLimitApplies(t *testing.T) {
	e := newEnv(t, 5)
	srv := serveFeed(t, e.clock.now(), many(5)...)
	e.subscribe(e.user("ann"), srv.URL+"/feed")
	e.s.Keep = 2
	_, _ = e.s.Cleanup(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 2 {
		t.Fatalf("%d items", n)
	}
}

func TestUnsubscribedFeedKeepsFlaggedItems(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", time.Minute}, entry{"2", "B", 2 * time.Minute})
	ann := e.user("ann")
	sub := e.subscribe(ann, srv.URL+"/feed")
	items := e.items(ann, ViewLatest)
	e.flag(ann, items[0].ID, nil, ptr(true))
	if err := e.s.Unsubscribe(ctx, ann, sub.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.s.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatal("the feed of a flagged item was deleted")
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 1 {
		t.Fatal("unflagged items of an unsubscribed feed stayed")
	}
	liked := e.items(ann, ViewLiked)
	if len(liked) != 1 || liked[0].Feed.Title != "Example" || liked[0].Feed.SubscriptionID != nil {
		t.Fatalf("%+v", liked)
	}
	if len(e.items(ann, ViewLatest)) != 0 {
		t.Fatal("latest shows an unsubscribed feed")
	}
	e.flag(ann, liked[0].ID, nil, ptr(false))
	_, _ = e.s.Cleanup(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 0 {
		t.Fatal("an unused feed stayed")
	}
}

func (e *env) search(user string, q ItemQuery) ([]Item, string) {
	e.t.Helper()
	if q.View == "" {
		q.View = ViewLatest
	}
	if q.Limit == 0 {
		q.Limit = 100
	}
	items, next, err := e.s.Items(ctx, user, q)
	if err != nil {
		e.t.Fatal(err)
	}
	return items, next
}

func TestSearch(t *testing.T) {
	e := newEnv(t, 100)
	a := serveFeed(t, e.clock.now(),
		entry{"1", "Rust in production", time.Minute},
		entry{"2", "Go in production", 2 * time.Minute},
		entry{"3", "Ärger im Büro", 3 * time.Minute},
		entry{"4", "Plain", 4 * time.Minute})
	a.summaries = map[string]string{"4": "All about Kubernetes"}
	b := serveFeed(t, e.clock.now(), entry{"1", "Rust on B", 30 * time.Second})
	ann := e.user("ann")
	subA := e.subscribe(ann, a.URL+"/feed")
	e.subscribe(ann, b.URL+"/feed")

	for q, want := range map[string]string{
		"production rust": "Rust in production",
		"kubernetes":      "Plain",
		"ärger":           "Ärger im Büro",
		"ÄRGER büro":      "Ärger im Büro",
		"  ":              "Rust on B,Rust in production,Go in production,Ärger im Büro,Plain",
		"missing":         "",
	} {
		if items, _ := e.search(ann, ItemQuery{Search: q}); titles(items) != want {
			t.Errorf("%q: %s", q, titles(items))
		}
	}

	for _, it := range e.items(ann, ViewLatest) {
		if strings.HasPrefix(it.Title, "Rust") {
			e.flag(ann, it.ID, nil, ptr(true))
		}
	}
	if items, _ := e.search(ann, ItemQuery{View: ViewLiked, Search: "rust", SubscriptionIDs: []string{subA.ID}}); titles(items) != "Rust in production" {
		t.Fatalf("liked: %s", titles(items))
	}
	if items, _ := e.search(ann, ItemQuery{View: ViewReadLater, Search: "rust"}); len(items) != 0 {
		t.Fatalf("read later: %s", titles(items))
	}

	first, next := e.search(ann, ItemQuery{Search: "production", Limit: 1})
	if titles(first) != "Rust in production" || next == "" {
		t.Fatalf("first page: %s %q", titles(first), next)
	}
	second, last := e.search(ann, ItemQuery{Search: "production", Limit: 1, Cursor: next})
	if titles(second) != "Go in production" || last != "" {
		t.Fatalf("second page: %s %q", titles(second), last)
	}
}

func TestFlagAccess(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	item := e.items(ann, ViewLatest)[0]
	if _, err := e.s.SetFlags(ctx, bob, item.ID, ptr(true), nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_item_flags`); n != 0 {
		t.Fatal("a flag was set")
	}
}

func TestIndependentFlags(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", time.Minute})
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	id := e.items(ann, ViewLatest)[0].ID
	e.flag(ann, id, ptr(true), nil)
	e.flag(ann, id, nil, ptr(true))
	e.flag(ann, id, ptr(false), nil)
	if len(e.items(ann, ViewLiked)) != 1 || len(e.items(ann, ViewReadLater)) != 0 {
		t.Fatal("flags aren't independent")
	}
}

func TestPagination(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), many(5)...)
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	page, next, err := e.s.Items(ctx, ann, ItemQuery{View: ViewLatest, Limit: 2})
	if err != nil || titles(page) != "T0,T1" || next == "" {
		t.Fatalf("%s %q %v", titles(page), next, err)
	}
	page, next, err = e.s.Items(ctx, ann, ItemQuery{View: ViewLatest, Limit: 3, Cursor: next})
	if err != nil || titles(page) != "T2,T3,T4" || next != "" {
		t.Fatalf("%s %q %v", titles(page), next, err)
	}
	if _, _, err := e.s.Items(ctx, ann, ItemQuery{View: "nope", Limit: 2}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := e.s.Items(ctx, ann, ItemQuery{View: ViewLatest, Limit: 2, Cursor: "!!"}); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("err = %v", err)
	}
}

func TestMergeOnPermanentRedirect(t *testing.T) {
	e := newEnv(t, 100)
	a := serveFeed(t, e.clock.now(), entry{"shared", "Shared", time.Hour}, entry{"a", "OnlyA", 2 * time.Hour})
	b := serveFeed(t, e.clock.now(), entry{"shared", "Shared", time.Hour}, entry{"b", "OnlyB", 3 * time.Hour})
	ann, bob, cid := e.user("ann"), e.user("bob"), e.user("cid")
	subA := e.subscribe(ann, a.URL+"/feed")
	e.clock.add(time.Minute)
	e.subscribe(ann, b.URL+"/feed")
	e.subscribe(bob, b.URL+"/feed")
	e.subscribe(cid, a.URL+"/feed")
	var sharedA, sharedB string
	_ = e.db.QueryRow(`SELECT i.id FROM feed_items i JOIN feeds f ON f.id = i.feed_id WHERE i.identity = 'shared' AND f.url LIKE ?`, a.URL+"%").Scan(&sharedA)
	_ = e.db.QueryRow(`SELECT i.id FROM feed_items i JOIN feeds f ON f.id = i.feed_id WHERE i.identity = 'shared' AND f.url LIKE ?`, b.URL+"%").Scan(&sharedB)
	e.flag(cid, sharedA, nil, ptr(true))
	e.flag(bob, sharedB, ptr(true), nil)

	a.mu.Lock()
	a.redirect = b.URL + "/feed"
	a.mu.Unlock()
	e.clock.add(2 * time.Hour)
	if _, err := e.s.RefreshDue(ctx); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatalf("%d feeds", n)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE identity = 'shared'`); n != 1 {
		t.Fatalf("%d copies of the shared item", n)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_subscriptions WHERE user_id = ?`, ann); n != 1 {
		t.Fatalf("Ann has %d subscriptions", n)
	}
	if _, err := e.s.Get(ctx, ann, subA.ID); err != nil {
		t.Fatalf("the earlier subscription is gone: %v", err)
	}
	if len(e.items(cid, ViewLiked)) != 1 || len(e.items(bob, ViewReadLater)) != 1 {
		t.Fatal("flags weren't kept")
	}
}

func TestMoveWithoutConflictUpdatesURL(t *testing.T) {
	e := newEnv(t, 100)
	target := serveFeed(t, e.clock.now(), entry{"1", "A", time.Hour})
	source := serveFeed(t, e.clock.now(), entry{"1", "A", time.Hour})
	ann := e.user("ann")
	sub := e.subscribe(ann, source.URL+"/feed")
	source.mu.Lock()
	source.redirect = target.URL + "/new"
	source.mu.Unlock()
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	got, err := e.s.Get(ctx, ann, sub.ID)
	want, _ := normalizeURL(target.URL + "/new")
	if err != nil || got.URL != want {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestImportCantChangeSharedItems(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "Real", time.Minute})
	ann, mallory := e.user("ann"), e.user("mallory")
	sub := e.subscribe(ann, srv.URL+"/feed")
	stamp := db.FormatTime(e.clock.now())
	restored, err := e.s.RestoreFlagsTx(ctx, e.db, mallory, sub.FeedID, Imported{Identity: "1", Title: "Forged", LikedAt: &stamp})
	if err != nil || !restored {
		t.Fatalf("%v %v", restored, err)
	}
	if got := titles(e.items(ann, ViewLatest)); got != "Real" {
		t.Fatalf("Ann sees %s", got)
	}
	if got := titles(e.items(mallory, ViewLiked)); got != "Real" {
		t.Fatalf("Mallory sees %s", got)
	}

	// An item the server doesn't have stays with the one who imported it.
	if _, err := e.s.RestoreFlagsTx(ctx, e.db, mallory, sub.FeedID, Imported{Identity: "x", Title: "Injected", LikedAt: &stamp}); err != nil {
		t.Fatal(err)
	}
	if got := titles(e.items(ann, ViewLatest)); got != "Real" {
		t.Fatalf("Ann sees %s", got)
	}
	if !strings.Contains(titles(e.items(mallory, ViewLiked)), "Injected") {
		t.Fatal("Mallory lost her item")
	}
	// Someone else's private row is skipped.
	bob := e.user("bob")
	if ok, err := e.s.RestoreFlagsTx(ctx, e.db, bob, sub.FeedID, Imported{Identity: "x", Title: "Other", LikedAt: &stamp}); err != nil || ok {
		t.Fatalf("%v %v", ok, err)
	}
	if strings.Contains(titles(e.items(bob, ViewLiked)), "Injected") {
		t.Fatal("Bob sees Mallory's unconfirmed item")
	}
}

func TestImportedItemsCantPushOutOthers(t *testing.T) {
	e := newEnv(t, 2)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", time.Minute}, entry{"2", "B", 2 * time.Minute})
	ann, mallory := e.user("ann"), e.user("mallory")
	sub := e.subscribe(ann, srv.URL+"/feed")
	stamp := db.FormatTime(e.clock.now())
	for i := range 5 {
		if _, err := e.s.RestoreFlagsTx(ctx, e.db, mallory, sub.FeedID, Imported{Identity: fmt.Sprint("x", i), Title: "Fake", PublishedAt: "2026-11-08T11:59:59Z", LikedAt: &stamp}); err != nil {
			t.Fatal(err)
		}
	}
	_, _ = e.s.Cleanup(ctx)
	if got := titles(e.items(ann, ViewLatest)); got != "A,B" {
		t.Fatalf("Ann sees %s", got)
	}
}

func TestImportedItemConfirmedByFetch(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "Real", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	sub := e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, srv.URL+"/feed")
	stamp := db.FormatTime(e.clock.now())
	if _, err := e.s.RestoreFlagsTx(ctx, e.db, ann, sub.FeedID, Imported{Identity: "2", Title: "From the file", LikedAt: &stamp}); err != nil {
		t.Fatal(err)
	}
	srv.set(entry{"1", "Real", time.Minute}, entry{"2", "From the feed", 2 * time.Minute})
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	if got := titles(e.items(bob, ViewLatest)); !strings.Contains(got, "From the feed") {
		t.Fatalf("Bob sees %s", got)
	}
	if got := titles(e.items(ann, ViewLiked)); got != "From the feed" {
		t.Fatalf("Ann's like: %s", got)
	}
}

func TestImportedFeedIsCheckedBeforeSharing(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "Real", time.Minute})
	mallory, bob := e.user("mallory"), e.user("bob")
	id, err := e.s.EnsureFeedTx(ctx, e.db, srv.URL+"/feed")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.s.SubscribeTx(ctx, e.db, mallory, id, nil, nil); err != nil {
		t.Fatal(err)
	}
	if srv.hits.Load() != 0 {
		t.Fatal("the import fetched the feed")
	}
	u, _ := url.Parse(srv.URL)
	var title string
	_ = e.db.QueryRow(`SELECT title FROM feeds`).Scan(&title)
	if title != u.Hostname() {
		t.Fatalf("title = %q", title)
	}
	e.subscribe(bob, srv.URL+"/feed")
	if srv.hits.Load() == 0 {
		t.Fatal("a feed nobody has fetched was trusted")
	}
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 1 {
		t.Fatalf("%d feeds", n)
	}
}

func TestDeletingAnAccount(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "A", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	sub := e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, srv.URL+"/feed")
	e.flag(ann, e.items(ann, ViewLatest)[0].ID, nil, ptr(true))
	stamp := db.FormatTime(e.clock.now())
	if _, err := e.s.RestoreFlagsTx(ctx, e.db, ann, sub.FeedID, Imported{Identity: "x", Title: "Mine", LikedAt: &stamp}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.db.Exec(`DELETE FROM users WHERE id = ?`, ann); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_subscriptions WHERE user_id = ?`, ann); n != 0 {
		t.Fatal("subscriptions stayed")
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_item_flags`); n != 0 {
		t.Fatal("flags stayed")
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE identity = 'x'`); n != 0 {
		t.Fatal("an item only Ann imported stayed")
	}
	if got := titles(e.items(bob, ViewLatest)); got != "A" {
		t.Fatalf("Bob sees %s", got)
	}
}

func TestAudioOnlyAppliesBeforeTheLimit(t *testing.T) {
	e := newEnv(t, 100)
	entries := ""
	for i, audio := range []bool{false, true, false, true, true} {
		enclosure := ""
		if audio {
			enclosure = fmt.Sprintf(`<enclosure url="https://a.example/%d.mp3" type="audio/mpeg"/>`, i)
		}
		entries += fmt.Sprintf(`<item><guid>%d</guid><title>T%d</title>%s<pubDate>%s</pubDate></item>`,
			i, i, enclosure, e.clock.now().Add(-time.Duration(i+1)*time.Minute).Format(time.RFC1123Z))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<rss version="2.0"><channel><title>Mixed</title>%s</channel></rss>`, entries)
	}))
	defer srv.Close()
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	items, next, err := e.s.Items(ctx, ann, ItemQuery{View: ViewLatest, AudioOnly: true, Limit: 2})
	if err != nil || titles(items) != "T1,T3" || next == "" {
		t.Fatalf("%s %q %v", titles(items), next, err)
	}
	items, _, err = e.s.Items(ctx, ann, ItemQuery{View: ViewLatest, AudioOnly: true, Limit: 2, Cursor: next})
	if err != nil || titles(items) != "T4" {
		t.Fatalf("%s %v", titles(items), err)
	}
}
