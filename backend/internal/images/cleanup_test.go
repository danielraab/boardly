package images

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSweep(t *testing.T) {
	ctx := context.Background()
	s, ann := newStore(t)
	clock := &fakeClock{now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	s.Now = clock.Now

	save := func(source, user string) string {
		t.Helper()
		img, err := s.Save(ctx, source, user, pngData)
		if err != nil {
			t.Fatal(err)
		}
		return img.ID
	}
	unused := save(SourceUpload, ann)
	referenced := save(SourceUpload, ann)
	favicon := save(SourceFavicon, "")

	// A widget refers to an image by "imageId", nested in its config.
	const ts = "2026-01-01T00:00:00.000Z"
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES ('d1', ?, 'home', 'Home', ?, ?)`, []any{ann, ts, ts}},
		{`INSERT INTO dashboard_widgets (id, dashboard_id, kind, position, width, config) VALUES ('s1', 'd1', 'section', 0, 6, '{}')`, nil},
		{`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width, config)
		  VALUES ('w1', 'd1', 's1', 'link', 0, 2, json_object('height', 'auto', 'config', json_object('imageId', ?)))`, []any{referenced}},
	} {
		if _, err := s.DB.Exec(q.query, q.args...); err != nil {
			t.Fatal(err)
		}
	}

	clock.now = clock.now.Add(23 * time.Hour)
	fresh := save(SourceUpload, ann)
	clock.now = clock.now.Add(2 * time.Hour)

	// An orphan file from long ago, and one that was just written.
	oldOrphan := filepath.Join(s.Dir, "11111111-1111-1111-1111-111111111111")
	newOrphan := filepath.Join(s.Dir, ".tmp-123")
	for _, p := range []string{oldOrphan, newOrphan} {
		if err := os.WriteFile(p, pngData, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := clock.now.Add(-2 * time.Hour)
	if err := os.Chtimes(oldOrphan, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newOrphan, clock.now, clock.now); err != nil {
		t.Fatal(err)
	}

	n, err := s.Sweep(ctx)
	if err != nil || n != 1 {
		t.Fatalf("Sweep = %d, %v; want 1", n, err)
	}
	if _, _, err := s.Open(ctx, unused); !errors.Is(err, ErrNotFound) {
		t.Errorf("unused upload: %v", err)
	}
	for name, id := range map[string]string{"referenced": referenced, "fresh": fresh, "favicon": favicon} {
		_, f, err := s.Open(ctx, id)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		f.Close()
	}
	if _, err := os.Stat(filepath.Join(s.Dir, unused)); !os.IsNotExist(err) {
		t.Errorf("unused file still there: %v", err)
	}
	if _, err := os.Stat(oldOrphan); !os.IsNotExist(err) {
		t.Errorf("old orphan still there: %v", err)
	}
	if _, err := os.Stat(newOrphan); err != nil {
		t.Errorf("new orphan removed: %v", err)
	}
}
