package feeds

import (
	"errors"
	"testing"
	"time"
)

func serveEpisodes(t *testing.T, e *env, entries ...entry) *feedServer {
	t.Helper()
	srv := serveFeed(t, e.clock.now(), entries...)
	srv.audio = true
	return srv
}

func (e *env) listen(user, item string, position int, duration *int, finished *bool) Item {
	e.t.Helper()
	it, err := e.s.SaveListen(ctx, user, item, position, duration, finished)
	if err != nil {
		e.t.Fatal(err)
	}
	return it
}

// byTitle finds an item of the user's latest view.
func (e *env) byTitle(user, title string) Item {
	e.t.Helper()
	for _, it := range e.items(user, ViewLatest) {
		if it.Title == title {
			return it
		}
	}
	e.t.Fatalf("no item %q", title)
	return Item{}
}

func TestListenViews(t *testing.T) {
	e := newEnv(t, 100)
	a := serveEpisodes(t, e, entry{"1", "Alpha", 3 * time.Minute}, entry{"2", "Beta", 2 * time.Minute}, entry{"3", "Gamma", time.Minute}, entry{"4", "Delta", 4 * time.Minute})
	b := serveEpisodes(t, e, entry{"1", "Other", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	subA := e.subscribe(ann, a.URL+"/feed")
	e.subscribe(ann, b.URL+"/feed")
	e.subscribe(bob, a.URL+"/feed")

	e.listen(ann, e.byTitle(ann, "Alpha").ID, 600, ptr(3600), nil)
	e.clock.add(time.Minute)
	e.listen(ann, e.byTitle(ann, "Beta").ID, 100, nil, nil)
	e.clock.add(time.Minute)
	e.listen(ann, e.byTitle(ann, "Other").ID, 50, nil, nil)
	e.clock.add(time.Minute)
	gamma := e.listen(ann, e.byTitle(ann, "Gamma").ID, 900, nil, ptr(true))
	e.clock.add(time.Minute)
	e.listen(ann, e.byTitle(ann, "Delta").ID, 0, nil, nil) // marked as unplayed

	if gamma.Listen == nil || gamma.Listen.PositionSeconds != 0 || gamma.Listen.FinishedAt == nil {
		t.Fatalf("finished: %+v", gamma.Listen)
	}
	for view, want := range map[string]string{
		ViewListened:   "Delta,Gamma,Other,Beta,Alpha",
		ViewInProgress: "Other,Beta,Alpha",
		ViewFinished:   "Gamma",
	} {
		if got := titles(e.items(ann, view)); got != want {
			t.Errorf("%s: %s", view, got)
		}
	}
	if items := e.items(bob, ViewListened); len(items) != 0 {
		t.Fatalf("Bob sees Ann's history: %s", titles(items))
	}
	for _, it := range e.items(bob, ViewLatest) {
		if it.Listen != nil {
			t.Fatal("Ann's record shows for Bob")
		}
	}
	alpha := e.byTitle(ann, "Alpha")
	if alpha.Listen == nil || alpha.Listen.PositionSeconds != 600 || *alpha.Listen.DurationSeconds != 3600 {
		t.Fatalf("latest view: %+v", alpha.Listen)
	}

	first, next := e.search(ann, ItemQuery{View: ViewInProgress, Limit: 2})
	second, last := e.search(ann, ItemQuery{View: ViewInProgress, Limit: 2, Cursor: next})
	if titles(first) != "Other,Beta" || titles(second) != "Alpha" || last != "" {
		t.Fatalf("pages: %s / %s", titles(first), titles(second))
	}
	if got, _ := e.search(ann, ItemQuery{View: ViewListened, Search: "ALPHA"}); titles(got) != "Alpha" {
		t.Fatalf("search: %s", titles(got))
	}
	if got, _ := e.search(ann, ItemQuery{View: ViewInProgress, SubscriptionIDs: []string{subA.ID}}); titles(got) != "Beta,Alpha" {
		t.Fatalf("feed filter: %s", titles(got))
	}
}

func TestSaveListen(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveEpisodes(t, e, entry{"1", "Episode", time.Minute})
	news := serveFeed(t, e.clock.now(), entry{"1", "Article", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(ann, news.URL+"/feed")
	id := e.byTitle(ann, "Episode").ID

	it := e.listen(ann, id, 750, ptr(3600), nil)
	started := it.Listen.StartedAt
	e.clock.add(time.Minute)
	it = e.listen(ann, id, 0, nil, ptr(true))
	if it.Listen.FinishedAt == nil || it.Listen.PositionSeconds != 0 || *it.Listen.DurationSeconds != 3600 {
		t.Fatalf("mark played: %+v", it.Listen)
	}
	e.clock.add(time.Minute)
	it = e.listen(ann, id, 30, nil, nil)
	if it.Listen.FinishedAt != nil || it.Listen.PositionSeconds != 30 || it.Listen.StartedAt != started || it.Listen.UpdatedAt == started {
		t.Fatalf("play again: %+v", it.Listen)
	}
	it = e.listen(ann, id, 0, nil, ptr(false))
	if it.Listen.FinishedAt != nil || it.Listen.PositionSeconds != 0 {
		t.Fatalf("mark unplayed: %+v", it.Listen)
	}
	if len(e.items(ann, ViewListened)) != 1 || len(e.items(ann, ViewInProgress)) != 0 || len(e.items(ann, ViewFinished)) != 0 {
		t.Fatal("unplayed episode in the wrong views")
	}

	if err := e.s.DeleteListen(ctx, ann, id); err != nil {
		t.Fatal(err)
	}
	if err := e.s.DeleteListen(ctx, ann, id); err != nil {
		t.Fatalf("deleting again: %v", err)
	}
	if len(e.items(ann, ViewListened)) != 0 || e.byTitle(ann, "Episode").Listen != nil {
		t.Fatal("record stayed")
	}

	if _, err := e.s.SaveListen(ctx, ann, e.byTitle(ann, "Article").ID, 10, nil, nil); !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("no audio: %v", err)
	}
	for _, c := range []struct {
		position int
		duration *int
	}{{-1, nil}, {MaxListenSeconds + 1, nil}, {0, ptr(0)}, {0, ptr(MaxListenSeconds + 1)}} {
		if _, err := e.s.SaveListen(ctx, ann, id, c.position, c.duration, nil); !errors.Is(err, ErrInvalidQuery) {
			t.Errorf("%d %v: %v", c.position, c.duration, err)
		}
	}
	if _, err := e.s.SaveListen(ctx, bob, id, 10, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Bob: %v", err)
	}
	if err := e.s.DeleteListen(ctx, bob, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Bob delete: %v", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_item_listens WHERE user_id = ?`, bob); n != 0 {
		t.Fatal("Bob got a record")
	}
}

func TestListenedItemOfRemovedFeed(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveEpisodes(t, e, entry{"1", "Episode", time.Minute})
	other := serveEpisodes(t, e, entry{"1", "Unrelated", time.Minute})
	ann, bob := e.user("ann"), e.user("bob")
	sub := e.subscribe(ann, srv.URL+"/feed")
	e.subscribe(bob, other.URL+"/feed")
	id := e.byTitle(ann, "Episode").ID
	e.listen(ann, id, 60, nil, nil)
	if err := e.s.Unsubscribe(ctx, ann, sub.ID); err != nil {
		t.Fatal(err)
	}
	it := e.listen(ann, id, 90, nil, nil)
	if it.Feed.SubscriptionID != nil || it.Listen.PositionSeconds != 90 {
		t.Fatalf("%+v", it)
	}
	if _, err := e.s.SetFlags(ctx, ann, id, nil, ptr(true)); err != nil {
		t.Fatalf("patch: %v", err)
	}
	if got := titles(e.items(ann, ViewListened)); got != "Episode" {
		t.Fatalf("listened: %s", got)
	}
	unrelated := e.byTitle(bob, "Unrelated").ID
	if _, err := e.s.SetFlags(ctx, ann, unrelated, nil, ptr(true)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unrelated: %v", err)
	}
	_, _ = e.s.Cleanup(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feeds`); n != 2 {
		t.Fatal("the feed of a listened item was deleted")
	}
}

func TestListenedItemKept(t *testing.T) {
	e := newEnv(t, 2)
	srv := serveEpisodes(t, e, entry{"1", "A", 3 * time.Minute}, entry{"2", "B", 2 * time.Minute})
	ann := e.user("ann")
	e.subscribe(ann, srv.URL+"/feed")
	a := e.byTitle(ann, "A")
	e.listen(ann, a.ID, 60, nil, nil)

	srv.set(entry{"3", "C", time.Minute}, entry{"2", "B", 2 * time.Minute})
	e.clock.add(2 * time.Hour)
	_, _ = e.s.RefreshDue(ctx)
	_, _ = e.s.Cleanup(ctx)
	if got := titles(e.items(ann, ViewListened)); got != "A" {
		t.Fatalf("listened item: %q", got)
	}
	if err := e.s.DeleteListen(ctx, ann, a.ID); err != nil {
		t.Fatal(err)
	}
	_, _ = e.s.Cleanup(ctx)
	if n := e.count(`SELECT COUNT(*) FROM feed_items WHERE title = 'A'`); n != 0 {
		t.Fatal("an item removed from history stayed")
	}
}

func TestMergeKeepsLatestListen(t *testing.T) {
	e := newEnv(t, 100)
	a := serveEpisodes(t, e, entry{"shared", "Shared", time.Hour})
	b := serveEpisodes(t, e, entry{"shared", "Shared", time.Hour})
	ann, bob := e.user("ann"), e.user("bob")
	e.subscribe(ann, a.URL+"/feed")
	e.subscribe(ann, b.URL+"/feed")
	e.subscribe(bob, a.URL+"/feed")
	var sharedA, sharedB string
	_ = e.db.QueryRow(`SELECT i.id FROM feed_items i JOIN feeds f ON f.id = i.feed_id WHERE f.url LIKE ?`, a.URL+"%").Scan(&sharedA)
	_ = e.db.QueryRow(`SELECT i.id FROM feed_items i JOIN feeds f ON f.id = i.feed_id WHERE f.url LIKE ?`, b.URL+"%").Scan(&sharedB)
	e.listen(ann, sharedB, 100, nil, nil)
	e.clock.add(time.Minute)
	e.listen(ann, sharedA, 200, nil, nil) // later, on the item that is dropped
	e.listen(bob, sharedA, 300, nil, nil)

	a.mu.Lock()
	a.redirect = b.URL + "/feed"
	a.mu.Unlock()
	e.clock.add(2 * time.Hour)
	if _, err := e.s.RefreshDue(ctx); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 1 {
		t.Fatalf("%d items", n)
	}
	for user, want := range map[string]int{ann: 200, bob: 300} {
		items := e.items(user, ViewListened)
		if len(items) != 1 || items[0].Listen.PositionSeconds != want {
			t.Errorf("%s: %+v", user, items)
		}
	}
}
