package feeds

import (
	"errors"
	"testing"
	"time"
)

func (e *env) pause(user, id string, paused bool) Subscription {
	e.t.Helper()
	sub, err := e.s.Update(ctx, user, id, Patch{Paused: &paused})
	if err != nil {
		e.t.Fatal(err)
	}
	return sub
}

func TestPauseAndResume(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann, bob := e.user("ann"), e.user("bob")
	sub := e.subscribe(ann, srv.URL+"/feed")
	if sub.Paused {
		t.Fatal("a new subscription is paused")
	}
	if got := e.pause(ann, sub.ID, true); !got.Paused {
		t.Fatal("pausing didn't pause")
	}
	var first string
	_ = e.db.QueryRow(`SELECT paused_at FROM feed_subscriptions WHERE id = ?`, sub.ID).Scan(&first)
	e.clock.add(time.Hour)
	e.pause(ann, sub.ID, true)
	var again string
	_ = e.db.QueryRow(`SELECT paused_at FROM feed_subscriptions WHERE id = ?`, sub.ID).Scan(&again)
	if first == "" || again != first {
		t.Fatalf("pausing twice changed the time: %q, %q", first, again)
	}
	list, err := e.s.List(ctx, ann)
	if err != nil || len(list) != 1 || !list[0].Paused {
		t.Fatalf("%+v, %v", list, err)
	}
	if got := e.pause(ann, sub.ID, false); got.Paused {
		t.Fatal("resuming didn't resume")
	}
	if _, err := e.s.Update(ctx, bob, sub.ID, Patch{Paused: ptr(true)}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user paused Ann's feed: %v", err)
	}
}

func TestPausedFeedLeavesLatest(t *testing.T) {
	e := newEnv(t, 100)
	news := serveFeed(t, e.clock.now(), entry{"n1", "News", time.Hour}, entry{"n2", "Older news", 2 * time.Hour})
	pod := serveFeed(t, e.clock.now(), entry{"p1", "Episode", 3 * time.Hour})
	pod.audio = true
	ann, bob := e.user("ann"), e.user("bob")
	heise := e.subscribe(ann, news.URL+"/feed")
	podcast := e.subscribe(ann, pod.URL+"/feed")
	e.subscribe(bob, news.URL+"/feed")

	latest := e.items(ann, ViewLatest)
	e.flag(ann, latest[0].ID, ptr(true), ptr(true)) // "News"
	episode := latest[2]
	if _, err := e.s.SaveListen(ctx, ann, episode.ID, 60, nil, nil); err != nil {
		t.Fatal(err)
	}

	e.pause(ann, heise.ID, true)
	e.pause(ann, podcast.ID, true)
	if got := titles(e.items(ann, ViewLatest)); got != "" {
		t.Fatalf("latest shows paused feeds: %s", got)
	}
	filtered, _ := e.search(ann, ItemQuery{View: ViewLatest, SubscriptionIDs: []string{heise.ID}})
	if len(filtered) != 0 {
		t.Fatal("latest filtered by a paused feed shows its items")
	}
	if got := titles(e.items(bob, ViewLatest)); got != "News,Older news" {
		t.Fatalf("pausing changed Bob's latest: %s", got)
	}
	for _, view := range []string{ViewReadLater, ViewLiked} {
		items, _ := e.search(ann, ItemQuery{View: view, SubscriptionIDs: []string{heise.ID}})
		if len(items) != 1 || items[0].Feed.SubscriptionID == nil || *items[0].Feed.SubscriptionID != heise.ID {
			t.Fatalf("%s: %+v", view, items)
		}
	}
	for _, view := range []string{ViewListened, ViewInProgress} {
		if got := titles(e.items(ann, view)); got != "Episode" {
			t.Fatalf("%s: %s", view, got)
		}
	}
	// Items of a paused feed can still be flagged and listened to.
	e.flag(ann, latest[1].ID, nil, ptr(true))
	if _, err := e.s.SaveListen(ctx, ann, episode.ID, 120, nil, nil); err != nil {
		t.Fatal(err)
	}

	e.pause(ann, heise.ID, false)
	if got := titles(e.items(ann, ViewLatest)); got != "News,Older news" {
		t.Fatalf("resumed feed: %s", got)
	}
}

func TestPausedFeedIsStillRefreshedAndKept(t *testing.T) {
	e := newEnv(t, 100)
	srv := serveFeed(t, e.clock.now(), entry{"1", "One", time.Hour})
	ann := e.user("ann")
	sub := e.subscribe(ann, srv.URL+"/feed")
	e.pause(ann, sub.ID, true)
	srv.set(entry{"2", "Two", 0}, entry{"1", "One", time.Hour})
	e.clock.add(2 * time.Hour)
	if n, err := e.s.RefreshDue(ctx); err != nil || n != 1 {
		t.Fatalf("refreshed %d feeds: %v", n, err)
	}
	if _, err := e.s.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if n := e.count(`SELECT COUNT(*) FROM feed_items`); n != 2 {
		t.Fatalf("%d items after cleanup", n)
	}
	e.pause(ann, sub.ID, false)
	if got := titles(e.items(ann, ViewLatest)); got != "Two,One" {
		t.Fatalf("resumed feed: %s", got)
	}
}

func TestMergeKeepsPausedState(t *testing.T) {
	e := newEnv(t, 100)
	a := serveFeed(t, e.clock.now(), entry{"a", "OnlyA", time.Hour})
	b := serveFeed(t, e.clock.now(), entry{"b", "OnlyB", time.Hour})
	ann := e.user("ann")
	subA := e.subscribe(ann, a.URL+"/feed")
	e.clock.add(time.Minute)
	e.subscribe(ann, b.URL+"/feed")
	e.pause(ann, subA.ID, true)

	a.mu.Lock()
	a.redirect = b.URL + "/feed"
	a.mu.Unlock()
	e.clock.add(2 * time.Hour)
	if _, err := e.s.RefreshDue(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := e.s.Get(ctx, ann, subA.ID)
	if err != nil || !got.Paused {
		t.Fatalf("%+v, %v", got, err)
	}
}
