package jobs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"boardly/internal/dbtest"
)

type payload struct {
	URL string `json:"url"`
}

func newQueue(t *testing.T) *Queue {
	t.Helper()
	qu := New(dbtest.Open(t))
	qu.PollInterval = 20 * time.Millisecond
	return qu
}

// pending returns the payload URLs of pending jobs of kind, in id order.
func pending(t *testing.T, qu *Queue, kind Kind) []string {
	t.Helper()
	rows, err := qu.DB.Query(`SELECT payload FROM jobs WHERE kind = ? AND started_at IS NULL ORDER BY id`, kind)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var p payload
		if err := json.Unmarshal([]byte(raw), &p); err != nil {
			t.Fatal(err)
		}
		out = append(out, p.URL)
	}
	return out
}

func enqueue(t *testing.T, qu *Queue, kind Kind, key, url string) {
	t.Helper()
	if err := qu.Enqueue(context.Background(), qu.DB, kind, key, payload{url}); err != nil {
		t.Fatal(err)
	}
}

func TestEnqueueCombinesPendingJobs(t *testing.T) {
	qu := newQueue(t)
	// Favicon jobs keep the first pending job of an origin.
	enqueue(t, qu, KindFavicon, "https://a.example", "https://a.example/1")
	enqueue(t, qu, KindFavicon, "https://a.example", "https://a.example/2")
	enqueue(t, qu, KindFavicon, "https://b.example", "https://b.example/")
	if got := pending(t, qu, KindFavicon); len(got) != 2 || got[0] != "https://a.example/1" || got[1] != "https://b.example/" {
		t.Fatalf("favicon jobs = %v", got)
	}
	// Preview fetches take the newest URL of a link.
	enqueue(t, qu, KindPreviewFetch, "node1", "https://old.example/")
	enqueue(t, qu, KindPreviewFetch, "node1", "https://new.example/")
	if got := pending(t, qu, KindPreviewFetch); len(got) != 1 || got[0] != "https://new.example/" {
		t.Fatalf("preview fetch jobs = %v", got)
	}
	// Jobs without a key are never combined.
	enqueue(t, qu, KindPreviewConvert, "", "x")
	enqueue(t, qu, KindPreviewConvert, "", "x")
	if got := pending(t, qu, KindPreviewConvert); len(got) != 2 {
		t.Fatalf("convert jobs = %v", got)
	}
}

func TestEnqueueInRolledBackTransaction(t *testing.T) {
	qu := newQueue(t)
	tx, err := qu.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := qu.Enqueue(context.Background(), tx, KindFavicon, "k", payload{"u"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if got := pending(t, qu, KindFavicon); len(got) != 0 {
		t.Fatalf("jobs after rollback = %v", got)
	}
}

func TestEnqueueSignals(t *testing.T) {
	qu := newQueue(t)
	enqueue(t, qu, KindFavicon, "", "u")
	select {
	case <-qu.signal(KindFavicon):
	default:
		t.Fatal("no wake-up signal after enqueue")
	}
	select {
	case <-qu.signal(KindPreviewFetch):
		t.Fatal("other kind was woken")
	default:
	}
}

// collect runs a pool and records the URLs of handled jobs.
type collect struct {
	mu   sync.Mutex
	urls []string
	done chan struct{}
}

func (c *collect) handler(fn func(p payload) error) Handler {
	return func(_ context.Context, job Job) error {
		var p payload
		if err := json.Unmarshal(job.Payload, &p); err != nil {
			return err
		}
		c.mu.Lock()
		c.urls = append(c.urls, p.URL)
		c.mu.Unlock()
		c.done <- struct{}{}
		if fn != nil {
			return fn(p)
		}
		return nil
	}
}

func (c *collect) wait(t *testing.T, n int) {
	t.Helper()
	for range n {
		select {
		case <-c.done:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out; handled %v", c.urls)
		}
	}
}

func waitEmpty(t *testing.T, qu *Queue) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var n int
		if err := qu.DB.QueryRow(`SELECT COUNT(*) FROM jobs`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d jobs left", n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestRunInOrderAndDeletes(t *testing.T) {
	qu := newQueue(t)
	for _, u := range []string{"1", "2", "3"} {
		enqueue(t, qu, KindPreviewConvert, "", u)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &collect{done: make(chan struct{}, 10)}
	qu.Run(ctx, KindPreviewConvert, 1, c.handler(func(p payload) error {
		if p.URL == "2" {
			return errors.New("broken")
		}
		return nil
	}))
	c.wait(t, 3)
	waitEmpty(t, qu)
	if len(c.urls) != 3 || c.urls[0] != "1" || c.urls[1] != "2" || c.urls[2] != "3" {
		t.Fatalf("order = %v", c.urls)
	}
}

func TestRunNoDoubleClaim(t *testing.T) {
	qu := newQueue(t)
	const n = 40
	for i := range n {
		enqueue(t, qu, KindFavicon, "", string(rune('a'+i%26))+string(rune('0'+i/26)))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &collect{done: make(chan struct{}, n)}
	qu.Run(ctx, KindFavicon, 4, c.handler(nil))
	c.wait(t, n)
	waitEmpty(t, qu)
	seen := map[string]bool{}
	for _, u := range c.urls {
		if seen[u] {
			t.Fatalf("job %q ran twice", u)
		}
		seen[u] = true
	}
}

func TestRunRecoversFromPanic(t *testing.T) {
	qu := newQueue(t)
	enqueue(t, qu, KindPreviewConvert, "", "boom")
	enqueue(t, qu, KindPreviewConvert, "", "fine")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &collect{done: make(chan struct{}, 10)}
	qu.Run(ctx, KindPreviewConvert, 1, c.handler(func(p payload) error {
		if p.URL == "boom" {
			panic("bad image")
		}
		return nil
	}))
	c.wait(t, 2)
	waitEmpty(t, qu)
}

func TestRunPicksUpJobCommittedLater(t *testing.T) {
	qu := newQueue(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &collect{done: make(chan struct{}, 10)}
	qu.Run(ctx, KindFavicon, 1, c.handler(nil))

	// The signal arrives before the commit; the poll finds the job.
	tx, err := qu.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := qu.Enqueue(ctx, tx, KindFavicon, "", payload{"late"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	c.wait(t, 1)
}

func TestRunLeavesInterruptedJob(t *testing.T) {
	qu := newQueue(t)
	enqueue(t, qu, KindPreviewFetch, "n1", "u")
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	qu.Run(ctx, KindPreviewFetch, 1, func(ctx context.Context, _ Job) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	})
	<-started
	cancel()
	time.Sleep(50 * time.Millisecond)
	var n int
	if err := qu.DB.QueryRow(`SELECT COUNT(*) FROM jobs WHERE started_at IS NOT NULL`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("started jobs = %d, err %v; want 1", n, err)
	}
}

func TestRecover(t *testing.T) {
	qu := newQueue(t)
	ctx := context.Background()
	const ts = "2026-01-01T00:00:00.000Z"
	insert := `INSERT INTO jobs (kind, dedup_key, payload, attempts, created_at, started_at) VALUES (?, ?, ?, ?, ?, ?)`
	for _, args := range [][]any{
		{KindFavicon, "https://a.example", `{"url":"interrupted"}`, 1, ts, ts},
		{KindPreviewConvert, nil, `{"url":"crashes"}`, MaxAttempts, ts, ts},
		{KindPreviewFetch, "n1", `{"url":"old"}`, 1, ts, ts},
		{KindPreviewFetch, "n1", `{"url":"new"}`, 0, ts, nil},
	} {
		if _, err := qu.DB.Exec(insert, args...); err != nil {
			t.Fatal(err)
		}
	}
	var logs bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	defer slog.SetDefault(old)

	if err := qu.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	if got := pending(t, qu, KindFavicon); len(got) != 1 || got[0] != "interrupted" {
		t.Fatalf("favicon jobs = %v, want the interrupted one queued again", got)
	}
	if got := pending(t, qu, KindPreviewConvert); len(got) != 0 {
		t.Fatalf("convert jobs = %v, want the crashing one dropped", got)
	}
	if got := pending(t, qu, KindPreviewFetch); len(got) != 1 || got[0] != "new" {
		t.Fatalf("preview fetch jobs = %v, want only the pending one", got)
	}
	if !bytes.Contains(logs.Bytes(), []byte("interrupted too often")) || !bytes.Contains(logs.Bytes(), []byte("crashes")) {
		t.Fatalf("log = %q, want a warning about the dropped job", logs.String())
	}

	// The interrupted job runs again.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c := &collect{done: make(chan struct{}, 10)}
	qu.Run(runCtx, KindFavicon, 1, c.handler(nil))
	c.wait(t, 1)
	if c.urls[0] != "interrupted" {
		t.Fatalf("ran %v", c.urls)
	}
}
