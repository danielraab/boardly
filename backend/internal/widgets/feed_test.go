package widgets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"boardly/internal/dashboards"
	"boardly/internal/dbtest"
	"boardly/internal/feeds"
)

type feedFixture struct {
	store *feeds.Store
	kind  dashboards.Kind
	owner string
	sub   feeds.Subscription
}

func newFeedFixture(t *testing.T) *feedFixture {
	t.Helper()
	database := dbtest.Open(t)
	store := feeds.NewStore(database, feeds.NewFetcher(true), nil, time.Hour, 100)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>Secret name</title>` +
			`<item><guid>1</guid><title>News one</title><link>https://a.example/1</link><description>The summary</description><pubDate>Sat, 07 Nov 2026 10:00:00 +0000</pubDate></item>` +
			`<item><guid>2</guid><title>Episode</title><enclosure url="https://a.example/2.mp3" type="audio/mpeg"/><pubDate>Fri, 06 Nov 2026 10:00:00 +0000</pubDate></item>` +
			`<item><guid>3</guid><title>Second show</title><enclosure url="https://a.example/3.mp3" type="audio/mpeg"/><pubDate>Thu, 05 Nov 2026 10:00:00 +0000</pubDate></item>` +
			`</channel></rss>`))
	}))
	t.Cleanup(srv.Close)
	owner := "owner-id"
	if _, err := database.Exec(`INSERT INTO users (id, email, created_at, updated_at) VALUES (?, 'o@example.org', 'x', 'x')`, owner); err != nil {
		t.Fatal(err)
	}
	sub, err := store.Subscribe(context.Background(), owner, srv.URL+"/feed?token=hidden")
	if err != nil {
		t.Fatal(err)
	}
	return &feedFixture{store: store, kind: Feed(store), owner: owner, sub: sub}
}

func feedRaw(ids []string, show string, public, summary bool) json.RawMessage {
	list, _ := json.Marshal(ids)
	return json.RawMessage(fmt.Sprintf(`{"feedIds":%s,"show":%q,"count":10,"showSummary":%v,"size":"M","showIfPublic":%v}`, list, show, summary, public))
}

func TestFeedNormalize(t *testing.T) {
	f := newFeedFixture(t)
	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = fmt.Sprint(i)
	}
	long := `"` + strings.Repeat("a", 201) + `"`
	for name, raw := range map[string]string{
		"count 0":   `{"feedIds":[],"show":"latest","count":0,"size":"M"}`,
		"count 101": `{"feedIds":[],"show":"latest","count":101,"size":"M"}`,
		"show":      `{"feedIds":[],"show":"all","count":5,"size":"M"}`,
		"size":      `{"feedIds":[],"show":"latest","count":5,"size":"XL"}`,
		"title":     `{"feedIds":[],"show":"latest","count":5,"size":"M","title":` + long + `}`,
		"repeated":  `{"feedIds":["a","a"],"show":"latest","count":5,"size":"M"}`,
		"too many":  string(feedRaw(tooMany, "latest", false, false)),
		"long id":   `{"feedIds":["` + strings.Repeat("x", 65) + `"],"show":"latest","count":5,"size":"M"}`,
		"unknown":   `{"feedIds":[],"show":"latest","count":5,"size":"M","nope":1}`,
	} {
		if _, err := f.kind.Normalize(context.Background(), f.owner, json.RawMessage(raw)); !errors.Is(err, dashboards.ErrInvalidLayout) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := f.kind.Normalize(context.Background(), f.owner, feedRaw(nil, "latest", false, false)); err != nil {
		t.Fatalf("all feeds: %v", err)
	}
	if _, err := f.kind.Normalize(context.Background(), f.owner, feedRaw(nil, "in_progress", false, false)); err != nil {
		t.Fatalf("in_progress: %v", err)
	}
}

func (f *feedFixture) resolve(t *testing.T, owner bool, raw json.RawMessage) (string, bool) {
	t.Helper()
	data, ok, err := f.kind.Resolve(context.Background(), dashboards.Viewer{OwnerID: f.owner, IsOwner: owner}, raw)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(data)
	return string(out), ok
}

func TestFeedOwnerSeesFlags(t *testing.T) {
	f := newFeedFixture(t)
	got, ok := f.resolve(t, true, feedRaw([]string{f.sub.ID}, "latest", false, false))
	if !ok || !strings.Contains(got, "News one") || !strings.Contains(got, "likedAt") || !strings.Contains(got, f.sub.ID) {
		t.Fatalf("%v %s", ok, got)
	}
}

func TestFeedVisitorFlagOff(t *testing.T) {
	f := newFeedFixture(t)
	if got, ok := f.resolve(t, false, feedRaw([]string{f.sub.ID}, "latest", false, false)); ok || got != "null" {
		t.Fatalf("%v %s", ok, got)
	}
}

func TestFeedVisitorGetsOnlyWhatIsShown(t *testing.T) {
	f := newFeedFixture(t)
	got, ok := f.resolve(t, false, feedRaw([]string{f.sub.ID}, "latest", true, false))
	if !ok {
		t.Fatal("left out")
	}
	for _, want := range []string{"News one", "https://a.example/2.mp3", "Secret name"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	for _, hidden := range []string{"token=hidden", f.sub.ID, f.sub.FeedID, "likedAt", "readLaterAt", "subscriptionId", "The summary"} {
		if strings.Contains(got, hidden) {
			t.Errorf("%q leaked in %s", hidden, got)
		}
	}
	got, _ = f.resolve(t, false, feedRaw([]string{f.sub.ID}, "latest", true, true))
	if !strings.Contains(got, "The summary") {
		t.Errorf("summary missing: %s", got)
	}
}

func TestFeedRemoved(t *testing.T) {
	f := newFeedFixture(t)
	raw := feedRaw([]string{f.sub.ID}, "latest", true, false)
	if err := f.store.Unsubscribe(context.Background(), f.owner, f.sub.ID); err != nil {
		t.Fatal(err)
	}
	if got, ok := f.resolve(t, true, raw); !ok || !strings.Contains(got, `"missing":true`) {
		t.Fatalf("%v %s", ok, got)
	}
	if _, ok := f.resolve(t, false, raw); ok {
		t.Fatal("visitors see a widget without feeds")
	}
}

func (f *feedFixture) pause(t *testing.T, id string, paused bool) {
	t.Helper()
	if _, err := f.store.Update(context.Background(), f.owner, id, feeds.Patch{Paused: &paused}); err != nil {
		t.Fatal(err)
	}
}

func TestFeedPaused(t *testing.T) {
	f := newFeedFixture(t)
	ctx := context.Background()
	other, err := f.store.Subscribe(ctx, f.owner, newOtherFeed(t))
	if err != nil {
		t.Fatal(err)
	}
	both := feedRaw([]string{f.sub.ID, other.ID}, "latest", true, false)
	f.pause(t, f.sub.ID, true)
	if got, ok := f.resolve(t, true, both); !ok || strings.Contains(got, "News one") || !strings.Contains(got, "Other news") || strings.Contains(got, `"paused":true`) {
		t.Fatalf("one of two paused: %v %s", ok, got)
	}
	all := feedRaw(nil, "latest", true, false)
	if got, _ := f.resolve(t, true, all); strings.Contains(got, "News one") || !strings.Contains(got, "Other news") {
		t.Fatalf("all feeds with one paused: %s", got)
	}

	only := feedRaw([]string{f.sub.ID}, "latest", true, false)
	if got, ok := f.resolve(t, true, only); !ok || !strings.Contains(got, `"paused":true`) || strings.Contains(got, `"missing":true`) {
		t.Fatalf("all chosen paused: %v %s", ok, got)
	}
	if _, ok := f.resolve(t, false, only); ok {
		t.Fatal("visitors see a widget with only paused feeds")
	}
	// A removed and a paused feed: the widget is paused, not missing.
	if err := f.store.Unsubscribe(ctx, f.owner, other.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.resolve(t, true, both); !strings.Contains(got, `"paused":true`) {
		t.Fatalf("removed and paused: %s", got)
	}
	// "All my feeds" with every feed paused just has no items.
	if got, ok := f.resolve(t, true, all); !ok || strings.Contains(got, `"paused":true`) || strings.Contains(got, "News one") {
		t.Fatalf("all feeds paused: %v %s", ok, got)
	}

	// Saved items of a paused feed stay, for the owner and for visitors.
	items, _, err := f.store.Items(ctx, f.owner, feeds.ItemQuery{View: feeds.ViewLiked, Limit: 10})
	if err != nil || len(items) != 0 {
		t.Fatal(items, err)
	}
	f.pause(t, f.sub.ID, false)
	latest, _, _ := f.store.Items(ctx, f.owner, feeds.ItemQuery{View: feeds.ViewLatest, Limit: 10})
	liked := true
	if _, err := f.store.SetFlags(ctx, f.owner, latest[0].ID, nil, &liked); err != nil {
		t.Fatal(err)
	}
	f.pause(t, f.sub.ID, true)
	likedRaw := feedRaw([]string{f.sub.ID}, "liked", true, false)
	for _, owner := range []bool{true, false} {
		if got, ok := f.resolve(t, owner, likedRaw); !ok || !strings.Contains(got, "News one") || strings.Contains(got, `"paused":true`) {
			t.Errorf("liked, owner %v: %v %s", owner, ok, got)
		}
	}

	// All chosen feeds removed still says missing.
	if err := f.store.Unsubscribe(ctx, f.owner, f.sub.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.resolve(t, true, only); !strings.Contains(got, `"missing":true`) || strings.Contains(got, `"paused":true`) {
		t.Fatalf("all removed: %s", got)
	}
}

// newOtherFeed serves a second feed and returns its URL.
func newOtherFeed(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>Other</title>` +
			`<item><guid>o1</guid><title>Other news</title><link>https://b.example/1</link><pubDate>Sat, 07 Nov 2026 09:00:00 +0000</pubDate></item>` +
			`</channel></rss>`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/other"
}

func TestFeedLikedView(t *testing.T) {
	f := newFeedFixture(t)
	ctx := context.Background()
	items, _, err := f.store.Items(ctx, f.owner, feeds.ItemQuery{View: feeds.ViewLatest, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	liked := true
	if _, err := f.store.SetFlags(ctx, f.owner, items[1].ID, nil, &liked); err != nil {
		t.Fatal(err)
	}
	got, _ := f.resolve(t, true, feedRaw(nil, "liked", false, false))
	if !strings.Contains(got, "Episode") || strings.Contains(got, "News one") {
		t.Fatal(got)
	}
	// Removing the feed keeps the liked item in a widget of all feeds.
	if err := f.store.Unsubscribe(ctx, f.owner, f.sub.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = f.resolve(t, true, feedRaw(nil, "liked", false, false))
	if !strings.Contains(got, "Episode") {
		t.Fatal(got)
	}
}

func TestFeedVisitorConfigHasNoSubscriptions(t *testing.T) {
	f := newFeedFixture(t)
	got, err := f.kind.VisitorConfig(feedRaw([]string{f.sub.ID}, "latest", true, false))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), f.sub.ID) || !strings.Contains(string(got), `"size":"M"`) {
		t.Fatalf("%s", got)
	}
}

func TestFeedAudioOnly(t *testing.T) {
	f := newFeedFixture(t)
	raw := json.RawMessage(`{"feedIds":[],"show":"latest","count":10,"showSummary":false,"audioOnly":true,"size":"M","showIfPublic":true}`)
	if _, err := f.kind.Normalize(context.Background(), f.owner, raw); err != nil {
		t.Fatal(err)
	}
	for _, owner := range []bool{true, false} {
		got, ok := f.resolve(t, owner, raw)
		if !ok || !strings.Contains(got, "Episode") || strings.Contains(got, "News one") {
			t.Errorf("owner %v: %v %s", owner, ok, got)
		}
	}
	// A widget saved before the setting existed shows everything.
	if got, _ := f.resolve(t, true, feedRaw(nil, "latest", false, false)); !strings.Contains(got, "News one") || !strings.Contains(got, "Episode") {
		t.Errorf("without the field: %s", got)
	}
}

func TestFeedContinueListening(t *testing.T) {
	f := newFeedFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	f.store.Now = func() time.Time { return now }
	items, _, err := f.store.Items(ctx, f.owner, feeds.ItemQuery{View: feeds.ViewLatest, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	byTitle := map[string]string{}
	for _, it := range items {
		byTitle[it.Title] = it.ID
	}
	duration := 3600
	if _, err := f.store.SaveListen(ctx, f.owner, byTitle["Second show"], 750, &duration, nil); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, err := f.store.SaveListen(ctx, f.owner, byTitle["Episode"], 30, nil, nil); err != nil {
		t.Fatal(err)
	}

	raw := feedRaw(nil, "in_progress", true, false)
	got, ok := f.resolve(t, true, raw)
	if !ok || strings.Index(got, `"Episode"`) > strings.Index(got, `"Second show"`) || strings.Index(got, `"Episode"`) < 0 {
		t.Fatalf("order: %s", got)
	}
	if !strings.Contains(got, `"positionSeconds":750`) || strings.Contains(got, "News one") {
		t.Fatalf("owner: %s", got)
	}
	got, ok = f.resolve(t, false, raw)
	if !ok || !strings.Contains(got, "Second show") {
		t.Fatalf("visitor: %v %s", ok, got)
	}
	for _, hidden := range []string{"listen", "positionSeconds", "updatedAt"} {
		if strings.Contains(got, hidden) {
			t.Errorf("%q leaked to a visitor: %s", hidden, got)
		}
	}

	finished := true
	if _, err := f.store.SaveListen(ctx, f.owner, byTitle["Episode"], 0, nil, &finished); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.resolve(t, true, raw); strings.Contains(got, `"Episode"`) {
		t.Fatalf("a finished episode stayed: %s", got)
	}
}
