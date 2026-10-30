package attachments

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/users"
)

var pdfData = []byte("%PDF-1.7\nhello")

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// newStore returns a store with the default types, a 1 KB limit and 30 days,
// and the IDs of the users Ann and Bob.
func newStore(t *testing.T) (*Store, *fakeClock, string, string) {
	t.Helper()
	database := dbtest.Open(t)
	s, err := NewStore(database, filepath.Join(t.TempDir(), "data"), 1024, DefaultAllowed(), 30)
	if err != nil {
		t.Fatal(err)
	}
	clock := &fakeClock{now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	s.Now = clock.Now
	us := users.NewStore(database)
	ann, err := us.EnsureByEmail(context.Background(), database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := us.EnsureByEmail(context.Background(), database, "bob@example.org")
	if err != nil {
		t.Fatal(err)
	}
	return s, clock, ann.ID, bob.ID
}

// refer makes a note of the user that refers to the attachments.
func refer(t *testing.T, s *Store, userID, noteID string, ids ...string) {
	t.Helper()
	const ts = "2026-01-01T00:00:00.000Z"
	if _, err := s.DB.Exec(`INSERT OR IGNORE INTO notes (id, user_id, content, created_at, updated_at) VALUES (?, ?, '{}', ?, ?)`,
		noteID, userID, ts, ts); err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		if _, err := s.DB.Exec(`INSERT INTO note_attachments (note_id, attachment_id, position) VALUES (?, ?, ?)`, noteID, id, i); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	s, clock, ann, _ := newStore(t)

	a, err := s.Create(ctx, ann, "  report.pdf ", pdfData)
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "report.pdf" || a.ContentType != "application/pdf" || a.Size != int64(len(pdfData)) ||
		a.Keep || a.DeletedAt != nil || a.DeleteAt != nil || len(a.NoteIDs) != 0 || !a.CreatedAt.Equal(clock.now) ||
		a.URL != "/api/attachments/"+a.ID+"/report.pdf" {
		t.Fatalf("created %+v", a)
	}
	got, err := os.ReadFile(s.Files.Path(a.ID))
	if err != nil || !bytes.Equal(got, pdfData) {
		t.Fatalf("file %q, %v", got, err)
	}
	if b, err := s.Create(ctx, ann, "my file.pdf", pdfData); err != nil || b.URL != "/api/attachments/"+b.ID+"/my%20file.pdf" {
		t.Fatalf("URL with a space: %+v, %v", b, err)
	}

	for name, c := range map[string]struct {
		name string
		data []byte
		want error
	}{
		"too large":   {"big.pdf", append([]byte("%PDF-"), make([]byte, 2000)...), ErrTooLarge},
		"not allowed": {"notes.zip", []byte("PK\x03\x04rest"), ErrUnsupported},
		"fake image":  {"photo.jpg", []byte("text"), ErrUnsupported},
		"empty":       {"empty.pdf", nil, ErrUnsupported},
		"path":        {"../etc/passwd", pdfData, ErrInvalidName},
		"backslash":   {`a\b.pdf`, pdfData, ErrInvalidName},
		"control":     {"a\nb.pdf", pdfData, ErrInvalidName},
		"blank":       {"  ", pdfData, ErrInvalidName},
		"long":        {strings.Repeat("a", 252) + ".pdf", pdfData, ErrInvalidName},
	} {
		if _, err := s.Create(ctx, ann, c.name, c.data); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
	list, err := s.List(ctx, ann)
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %d attachments, %v; want 2", len(list), err)
	}
}

func TestReadLimited(t *testing.T) {
	s, _, _, _ := newStore(t)
	if _, err := s.ReadLimited(bytes.NewReader(make([]byte, 1025))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if data, err := s.ReadLimited(bytes.NewReader(make([]byte, 1024))); err != nil || len(data) != 1024 {
		t.Fatalf("read %d, %v", len(data), err)
	}
}

func TestListAndGet(t *testing.T) {
	ctx := context.Background()
	s, clock, ann, bob := newStore(t)
	first, _ := s.Create(ctx, ann, "a.pdf", pdfData)
	clock.now = clock.now.Add(time.Minute)
	second, _ := s.Create(ctx, ann, "b.pdf", pdfData)
	if _, err := s.Create(ctx, bob, "bob.pdf", pdfData); err != nil {
		t.Fatal(err)
	}
	refer(t, s, ann, "n1", first.ID)
	refer(t, s, ann, "n2", first.ID)

	list, err := s.List(ctx, ann)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != second.ID || list[1].ID != first.ID {
		t.Fatalf("List = %+v; want b.pdf, a.pdf", list)
	}
	if !slices.Equal(list[1].NoteIDs, []string{"n1", "n2"}) || len(list[0].NoteIDs) != 0 {
		t.Fatalf("NoteIDs = %v, %v", list[1].NoteIDs, list[0].NoteIDs)
	}
	if _, err := s.Get(ctx, bob, first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Bob gets Ann's attachment: %v", err)
	}
}

func TestKeep(t *testing.T) {
	ctx := context.Background()
	s, clock, ann, bob := newStore(t)
	a, _ := s.Create(ctx, ann, "a.pdf", pdfData)
	marked := clock.now
	if _, err := s.DB.Exec(`UPDATE attachments SET deleted_at = ? WHERE id = ?`, "2026-03-01T12:00:00.000Z", a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, ann, a.ID)
	if err != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(marked) || got.DeleteAt == nil || !got.DeleteAt.Equal(marked.Add(30*24*time.Hour)) {
		t.Fatalf("marked = %+v, %v", got, err)
	}

	// Keep: the mark stays as information, no deletion date.
	got, err = s.SetKeep(ctx, ann, a.ID, true)
	if err != nil || !got.Keep || got.DeletedAt == nil || got.DeleteAt != nil {
		t.Fatalf("keep = %+v, %v", got, err)
	}
	// Don't keep: the mark is removed.
	got, err = s.SetKeep(ctx, ann, a.ID, false)
	if err != nil || got.Keep || got.DeletedAt != nil || got.DeleteAt != nil {
		t.Fatalf("don't keep = %+v, %v", got, err)
	}
	if _, err := s.SetKeep(ctx, bob, a.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Bob keeps Ann's attachment: %v", err)
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	s, _, ann, bob := newStore(t)
	used, _ := s.Create(ctx, ann, "used.pdf", pdfData)
	unused, _ := s.Create(ctx, ann, "unused.pdf", pdfData)
	refer(t, s, ann, "n1", used.ID)

	if err := s.Delete(ctx, ann, used.ID); !errors.Is(err, ErrInUse) {
		t.Fatalf("delete in use: %v", err)
	}
	if _, err := s.Get(ctx, ann, used.ID); err != nil {
		t.Fatalf("attachment in use is gone: %v", err)
	}
	if err := s.Delete(ctx, bob, unused.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Bob deletes Ann's attachment: %v", err)
	}
	if err := s.Delete(ctx, ann, unused.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Open(ctx, unused.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted attachment opens: %v", err)
	}
	if _, err := os.Stat(s.Files.Path(unused.ID)); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	s, _, ann, _ := newStore(t)
	s.Allowed = append(s.Allowed, "text/html")
	pdf, _ := s.Create(ctx, ann, "a.pdf", pdfData)
	html, _ := s.Create(ctx, ann, "page.html", []byte("<script>alert(1)</script>"))

	served, f, err := s.Open(ctx, pdf.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if served.Name != "a.pdf" || served.ContentType != "application/pdf" || !served.Inline {
		t.Fatalf("pdf = %+v", served)
	}
	served, f, err = s.Open(ctx, html.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if served.ContentType != "text/html" || served.Inline {
		t.Fatalf("html = %+v", served)
	}
	if _, _, err := s.Open(ctx, "doesnotexist"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
}

func TestSweep(t *testing.T) {
	ctx := context.Background()
	s, clock, ann, _ := newStore(t)
	create := func(name string) string {
		t.Helper()
		a, err := s.Create(ctx, ann, name, pdfData)
		if err != nil {
			t.Fatal(err)
		}
		return a.ID
	}
	unused := create("unused.pdf")
	used := create("used.pdf")
	kept := create("kept.pdf")
	refer(t, s, ann, "n1", used)
	if _, err := s.SetKeep(ctx, ann, kept, true); err != nil {
		t.Fatal(err)
	}
	clock.now = clock.now.Add(23 * time.Hour)
	recent := create("recent.pdf")

	// Recent upload: not marked yet.
	clock.now = clock.now.Add(2 * time.Hour)
	marked, deleted, err := s.Sweep(ctx)
	if err != nil || marked != 1 || deleted != 0 {
		t.Fatalf("Sweep = %d, %d, %v; want 1 marked", marked, deleted, err)
	}
	markTime := clock.now
	for id, want := range map[string]bool{unused: true, used: false, kept: false, recent: false} {
		a, _ := s.Get(ctx, ann, id)
		if (a.DeletedAt != nil) != want {
			t.Errorf("%s marked = %v, want %v", a.Name, a.DeletedAt != nil, want)
		}
	}
	a, _ := s.Get(ctx, ann, unused)
	if !a.DeleteAt.Equal(markTime.Add(30 * 24 * time.Hour)) {
		t.Errorf("deleteAt = %v", a.DeleteAt)
	}

	// Kept: marked earlier, then kept; never deleted.
	if _, err := s.DB.Exec(`UPDATE attachments SET deleted_at = ? WHERE id = ?`, "2026-01-01T00:00:00.000Z", kept); err != nil {
		t.Fatal(err)
	}

	// After the period, the unused attachment is deleted; recent is marked by now.
	clock.now = markTime.Add(30 * 24 * time.Hour)
	marked, deleted, err = s.Sweep(ctx)
	if err != nil || marked != 1 || deleted != 1 {
		t.Fatalf("Sweep = %d, %d, %v; want 1 marked, 1 deleted", marked, deleted, err)
	}
	if _, _, err := s.Open(ctx, unused); !errors.Is(err, ErrNotFound) {
		t.Errorf("unused still served: %v", err)
	}
	if _, err := os.Stat(s.Files.Path(unused)); !os.IsNotExist(err) {
		t.Errorf("unused file still exists: %v", err)
	}
	if a, err := s.Get(ctx, ann, kept); err != nil || a.DeletedAt == nil {
		t.Errorf("kept = %+v, %v; want kept with its mark", a, err)
	}

	// A marked attachment that a note refers to again isn't deleted.
	refer(t, s, ann, "n2", recent)
	clock.now = clock.now.Add(31 * 24 * time.Hour)
	if _, deleted, err := s.Sweep(ctx); err != nil || deleted != 0 {
		t.Fatalf("Sweep deleted %d, %v; want 0", deleted, err)
	}
}

func TestSweepRemovesOrphanFiles(t *testing.T) {
	s, clock, _, _ := newStore(t)
	orphan := filepath.Join(string(s.Files), "11111111-1111-1111-1111-111111111111")
	if err := os.WriteFile(orphan, pdfData, 0o600); err != nil {
		t.Fatal(err)
	}
	old := clock.now.Add(-2 * time.Hour)
	if err := os.Chtimes(orphan, old, old); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Sweep(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Fatalf("orphan still exists: %v", err)
	}
}
