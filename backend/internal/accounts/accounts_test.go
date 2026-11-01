package accounts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/blobs"
	"boardly/internal/dashboards"
	"boardly/internal/db"
	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/mail"
	"boardly/internal/notes"
	"boardly/internal/users"
)

type outbox struct {
	sent []mail.Message
	fail bool
}

func (o *outbox) Send(_ context.Context, m mail.Message) error {
	if o.fail {
		return errors.New("smtp down")
	}
	o.sent = append(o.sent, m)
	return nil
}

type fixture struct {
	t     *testing.T
	ctx   context.Context
	db    *sql.DB
	now   time.Time
	s     *Service
	mails *outbox
	users *users.Store
	ann   users.User
	bob   users.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	f := &fixture{t: t, ctx: context.Background(), db: database, now: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC), mails: &outbox{}}
	f.users = users.NewStore(database)
	f.users.Now = func() time.Time { return f.now }
	var err error
	if f.ann, err = f.users.EnsureByEmail(f.ctx, database, "ann@example.org"); err != nil {
		t.Fatal(err)
	}
	if f.bob, err = f.users.EnsureByEmail(f.ctx, database, "bob@example.org"); err != nil {
		t.Fatal(err)
	}
	imageDir, _ := blobs.Open(filepath.Join(t.TempDir(), "images"))
	attachmentDir, _ := blobs.Open(filepath.Join(t.TempDir(), "attachments"))
	f.s = &Service{
		DB: database, Users: f.users, Now: func() time.Time { return f.now }, Days: 7,
		Mailer: f.mails, AppTitle: "Boardly", Link: "https://dash.example.org/",
		ImageFiles: imageDir, AttachmentFiles: attachmentDir,
	}
	return f
}

func (f *fixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.ExecContext(f.ctx, query, args...); err != nil {
		f.t.Fatalf("%s: %v", query, err)
	}
}

func (f *fixture) count(query string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRowContext(f.ctx, query, args...).Scan(&n); err != nil {
		f.t.Fatalf("%s: %v", query, err)
	}
	return n
}

// session stores a session of the user and returns it as a lookup would.
func (f *fixture) session(userID, secret, method string, created time.Time) auth.Session {
	f.t.Helper()
	h := sha256.Sum256([]byte(secret))
	f.exec(`INSERT INTO sessions (id_hash, user_id, auth_method, created_at, last_used_at, expires_at, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, '')`, h[:], userID, method, db.FormatTime(created), db.FormatTime(created),
		db.FormatTime(created.Add(30*24*time.Hour)))
	return auth.Session{Secret: secret, UserID: userID, Method: method, CreatedAt: created}
}

func (f *fixture) sessions(userID string) int {
	return f.count(`SELECT COUNT(*) FROM sessions WHERE user_id = ?`, userID)
}

func (f *fixture) schedule() users.User {
	f.t.Helper()
	sess := f.session(f.ann.ID, "laptop-"+f.now.String(), auth.MethodMagicLink, f.now)
	u, err := f.s.Schedule(f.ctx, sess, "ann@example.org", "de")
	if err != nil {
		f.t.Fatal(err)
	}
	return u
}

func TestScheduleErrors(t *testing.T) {
	f := newFixture(t)
	passkey := f.session(f.ann.ID, "passkey", auth.MethodPasskey, f.now)
	if _, err := f.s.Schedule(f.ctx, passkey, "ann@example.org", "en"); !errors.Is(err, ErrWindowClosed) {
		t.Fatalf("passkey session: err = %v", err)
	}
	old := f.session(f.ann.ID, "old", auth.MethodMagicLink, f.now.Add(-auth.DeletionWindow))
	if _, err := f.s.Schedule(f.ctx, old, "ann@example.org", "en"); !errors.Is(err, ErrWindowClosed) {
		t.Fatalf("old session: err = %v", err)
	}
	fresh := f.session(f.ann.ID, "fresh", auth.MethodMagicLink, f.now.Add(-time.Minute))
	if _, err := f.s.Schedule(f.ctx, fresh, "bob@example.org", "en"); !errors.Is(err, ErrEmailMismatch) {
		t.Fatalf("wrong email: err = %v", err)
	}
	if u, _ := f.users.Get(f.ctx, f.ann.ID); u.DeletionScheduledAt != nil || f.sessions(f.ann.ID) != 3 {
		t.Fatal("a rejected request changed something")
	}
	u, err := f.s.Schedule(f.ctx, fresh, "  ANN@Example.org ", "fr")
	if err != nil {
		t.Fatalf("case and space must be ignored: %v", err)
	}
	if want := f.now.Add(7 * 24 * time.Hour); u.DeletionScheduledAt == nil || !u.DeletionScheduledAt.Equal(want) {
		t.Fatalf("DeletionScheduledAt = %v, want %v", u.DeletionScheduledAt, want)
	}
	if locale := f.count(`SELECT COUNT(*) FROM users WHERE id = ? AND deletion_locale = 'en'`, f.ann.ID); locale != 1 {
		t.Fatal("unsupported locale must be stored as en")
	}
	if _, err := f.s.Schedule(f.ctx, fresh, "ann@example.org", "en"); !errors.Is(err, ErrScheduled) {
		t.Fatalf("twice: err = %v", err)
	}
}

func TestScheduleEndsSessionsAndRevokesTokens(t *testing.T) {
	f := newFixture(t)
	tokens := &apptokens.Store{DB: f.db, Now: func() time.Time { return f.now }}
	tok, _, err := tokens.Create(f.ctx, f.ann.ID, apptokens.NewToken{Name: "Phone", App: apptokens.AppMemos})
	if err != nil {
		t.Fatal(err)
	}
	bobsToken, _, _ := tokens.Create(f.ctx, f.bob.ID, apptokens.NewToken{Name: "Bob", App: apptokens.AppMemos})
	f.session(f.ann.ID, "phone", auth.MethodPasskey, f.now)
	f.session(f.bob.ID, "bob", auth.MethodPasskey, f.now)
	laptop := f.session(f.ann.ID, "laptop", auth.MethodMagicLink, f.now)

	if _, err := f.s.Schedule(f.ctx, laptop, "ann@example.org", "en"); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte("laptop"))
	if f.sessions(f.ann.ID) != 1 || f.count(`SELECT COUNT(*) FROM sessions WHERE id_hash = ?`, h[:]) != 1 {
		t.Fatal("only the current session must be kept")
	}
	if f.sessions(f.bob.ID) != 1 {
		t.Fatal("other users' sessions must be kept")
	}
	if list, _ := tokens.List(f.ctx, f.ann.ID); list[0].ID != tok.ID || list[0].Status != apptokens.StatusRevoked {
		t.Fatalf("token = %+v, want revoked", list[0])
	}
	if list, _ := tokens.List(f.ctx, f.bob.ID); list[0].ID != bobsToken.ID || list[0].Status != apptokens.StatusActive {
		t.Fatal("other users' tokens must stay active")
	}

	// Cancelling keeps the token revoked.
	u, err := f.s.Cancel(f.ctx, f.ann.ID)
	if err != nil || u.DeletionScheduledAt != nil {
		t.Fatalf("cancel = %+v, %v", u, err)
	}
	if list, _ := tokens.List(f.ctx, f.ann.ID); list[0].Status != apptokens.StatusRevoked {
		t.Fatal("cancelling must not restore tokens")
	}
	if _, err := f.s.Cancel(f.ctx, f.ann.ID); !errors.Is(err, ErrNotScheduled) {
		t.Fatalf("cancel twice: err = %v", err)
	}
}

func TestReminder(t *testing.T) {
	f := newFixture(t)
	f.schedule()
	start := f.now

	f.now = start.Add(6*24*time.Hour - time.Minute)
	if n, _, err := f.s.Sweep(f.ctx); err != nil || n != 0 || len(f.mails.sent) != 0 {
		t.Fatalf("before day 6: reminded %d, %v", n, err)
	}

	f.now = start.Add(6 * 24 * time.Hour)
	f.mails.fail = true
	if _, _, err := f.s.Sweep(f.ctx); err == nil {
		t.Fatal("a failed send must be reported")
	}
	f.mails.fail = false
	f.now = f.now.Add(time.Hour)
	if n, _, err := f.s.Sweep(f.ctx); err != nil || n != 1 || len(f.mails.sent) != 1 {
		t.Fatalf("retry: reminded %d, %v, sent %d", n, err, len(f.mails.sent))
	}
	m := f.mails.sent[0]
	if m.To != "ann@example.org" || !strings.Contains(m.Subject, "8. November 2026, 10:00 UTC") || !strings.Contains(m.Text, "keine weiteren E-Mails") {
		t.Fatalf("reminder = %+v", m)
	}
	f.now = f.now.Add(time.Hour)
	if n, _, _ := f.s.Sweep(f.ctx); n != 0 || len(f.mails.sent) != 1 {
		t.Fatal("the reminder must be sent only once")
	}

	// Cancelling and scheduling again allows a new reminder.
	if _, err := f.s.Cancel(f.ctx, f.ann.ID); err != nil {
		t.Fatal(err)
	}
	f.schedule()
	f.now = f.now.Add(6 * 24 * time.Hour)
	if n, _, _ := f.s.Sweep(f.ctx); n != 1 || len(f.mails.sent) != 2 {
		t.Fatalf("after rescheduling: reminded %d", n)
	}
}

func TestReminderCancelled(t *testing.T) {
	f := newFixture(t)
	f.schedule()
	f.now = f.now.Add(3 * 24 * time.Hour)
	if _, err := f.s.Cancel(f.ctx, f.ann.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(4 * 24 * time.Hour)
	if n, d, err := f.s.Sweep(f.ctx); err != nil || n != 0 || d != 0 || len(f.mails.sent) != 0 {
		t.Fatalf("reminded %d, deleted %d, %v", n, d, err)
	}
}

func TestReminderShortGracePeriods(t *testing.T) {
	f := newFixture(t)
	f.s.Days = 1
	f.schedule()
	if n, d, _ := f.s.Sweep(f.ctx); n != 1 || d != 0 {
		t.Fatalf("1 day: reminded %d, deleted %d", n, d)
	}

	f = newFixture(t)
	f.s.Days = 0
	f.schedule()
	if n, d, _ := f.s.Sweep(f.ctx); n != 0 || d != 1 || len(f.mails.sent) != 0 {
		t.Fatalf("0 days: reminded %d, deleted %d", n, d)
	}
}

func TestDeleteAccount(t *testing.T) {
	f := newFixture(t)
	ctx := f.ctx
	ann, bob := f.ann.ID, f.bob.ID
	now := db.FormatTime(f.now)
	ptr := func(s string) *string { return &s }

	img, fav, att := blobs.NewID(), blobs.NewID(), blobs.NewID()

	// Favourites with tags.
	favs := favourites.NewStore(f.db)
	for _, u := range []string{ann, bob} {
		if _, err := favs.Create(ctx, u, favourites.NewNode{Kind: favourites.KindLink, URL: ptr("https://example.org/" + u), Tags: []string{"news"}}); err != nil {
			t.Fatal(err)
		}
	}
	// Dashboards with a section and a widget.
	dash := dashboards.NewStore(f.db)
	annDash, err := dash.Create(ctx, ann, dashboards.Fields{Title: "Home", Slug: "home", IsPublic: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dash.Create(ctx, bob, dashboards.Fields{Title: "Home", Slug: "home"}); err != nil {
		t.Fatal(err)
	}
	f.exec(`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width) VALUES ('s1', ?, NULL, 'section', 0, 6)`, annDash.ID)
	f.exec(`INSERT INTO dashboard_widgets (id, dashboard_id, parent_id, kind, position, width) VALUES ('w1', ?, 's1', 'link', 0, 2)`, annDash.ID)
	// Nested note categories, a note in the child with an attachment.
	ns := notes.NewStore(f.db)
	parent, err := ns.CreateCategory(ctx, ann, notes.NewCategory{Title: "Work"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := ns.CreateCategory(ctx, ann, notes.NewCategory{Title: "Projects", ParentID: &parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := ns.CreateCategory(ctx, ann, notes.NewCategory{Title: "Boardly", ParentID: &child.ID})
	if err != nil {
		t.Fatal(err)
	}
	note, err := ns.CreateNote(ctx, ann, notes.NewNote{Title: ptr("Plan"), CategoryID: &grandchild.ID})
	if err != nil {
		t.Fatal(err)
	}
	f.exec(`UPDATE users SET note_category_id = ? WHERE id = ?`, child.ID, ann)
	f.exec(`INSERT INTO attachments (id, user_id, name, content_type, size, created_at) VALUES (?, ?, 'a.pdf', 'application/pdf', 3, ?)`, att, ann, now)
	f.exec(`INSERT INTO note_attachments (note_id, attachment_id, position) VALUES (?, ?, 0)`, note.ID, att)
	if err := f.s.AttachmentFiles.Write(att, []byte("pdf")); err != nil {
		t.Fatal(err)
	}
	// An uploaded image used as a media source icon, and a shared favicon.
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 3, ?)`, img, ann, now)
	f.exec(`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'favicon', NULL, 'image/png', 3, ?)`, fav, now)
	f.exec(`INSERT INTO favicons (origin, image_id, fetched_at) VALUES ('https://example.org', ?, ?)`, fav, now)
	for _, id := range []string{img, fav} {
		if err := f.s.ImageFiles.Write(id, []byte("png")); err != nil {
			t.Fatal(err)
		}
	}
	f.exec(`INSERT INTO media_sources (id, user_id, kind, title, url, icon_image_id, position, created_at, updated_at)
		VALUES ('m1', ?, 'stream', 'Radio', 'https://radio.example.org', ?, 0, ?, ?)`, ann, img, now, now)
	// Passkey, app token, sign-in tokens, jobs.
	f.exec(`INSERT INTO passkeys (id, user_id, credential_id, public_key, aaguid, sign_count, transports, backup_eligible, backup_state, created_at)
		VALUES ('pk1', ?, x'01', x'02', x'03', 0, '[]', 0, 0, ?)`, ann, now)
	tokens := &apptokens.Store{DB: f.db, Now: func() time.Time { return f.now }}
	if _, _, err := tokens.Create(ctx, ann, apptokens.NewToken{Name: "Phone", App: apptokens.AppMemos}); err != nil {
		t.Fatal(err)
	}
	for i, email := range []string{"ann@example.org", "bob@example.org"} {
		f.exec(`INSERT INTO magic_link_tokens (token_hash, email, locale, created_at, expires_at, request_ip) VALUES (?, ?, 'en', ?, ?, '127.0.0.1')`,
			[]byte{byte(i)}, email, now, now)
	}
	f.exec(`INSERT INTO jobs (kind, payload, created_at) VALUES ('preview_fetch', json_object('userId', ?, 'nodeId', 'n', 'url', 'u'), ?)`, ann, now)
	f.exec(`INSERT INTO jobs (kind, payload, created_at) VALUES ('preview_fetch', json_object('userId', ?, 'nodeId', 'n', 'url', 'u'), ?)`, bob, now)
	f.exec(`INSERT INTO jobs (kind, dedup_key, payload, created_at) VALUES ('favicon', 'https://example.org', '{"url":"https://example.org"}', ?)`, now)

	f.schedule()
	f.now = f.now.Add(7*24*time.Hour - time.Second)
	if _, d, err := f.s.Sweep(ctx); err != nil || d != 0 {
		t.Fatalf("not yet due: deleted %d, %v", d, err)
	}
	if _, err := f.users.Get(ctx, ann); err != nil {
		t.Fatal("account must exist before its deletion time")
	}

	bobNodes := f.count(`SELECT COUNT(*) FROM favourite_nodes WHERE user_id = ?`, bob)
	f.now = f.now.Add(time.Second)
	if _, d, err := f.s.Sweep(ctx); err != nil || d != 1 {
		t.Fatalf("due: deleted %d, %v", d, err)
	}

	if _, err := f.users.Get(ctx, ann); !errors.Is(err, users.ErrNotFound) {
		t.Fatalf("user still exists: %v", err)
	}
	for _, table := range []string{"sessions", "passkeys", "app_tokens", "dashboards", "favourite_nodes", "tags",
		"notes", "note_categories", "attachments", "media_sources", "images"} {
		if n := f.count(`SELECT COUNT(*) FROM `+table+` WHERE user_id = ?`, ann); n != 0 {
			t.Errorf("%s: %d rows of the deleted user left", table, n)
		}
	}
	if n := f.count(`SELECT COUNT(*) FROM dashboard_widgets`); n != 0 {
		t.Errorf("dashboard_widgets: %d rows left", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM note_attachments`); n != 0 {
		t.Errorf("note_attachments: %d rows left", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM magic_link_tokens WHERE email = 'ann@example.org'`); n != 0 {
		t.Error("sign-in tokens of the deleted user left")
	}
	if n := f.count(`SELECT COUNT(*) FROM jobs`); n != 2 {
		t.Errorf("jobs = %d, want Bob's preview and the favicon job", n)
	}
	for _, path := range []string{f.s.ImageFiles.Path(img), f.s.AttachmentFiles.Path(att)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%s still exists", path)
		}
	}

	// Others are untouched.
	if _, err := os.Stat(f.s.ImageFiles.Path(fav)); err != nil {
		t.Errorf("favicon file: %v", err)
	}
	if n := f.count(`SELECT COUNT(*) FROM favicons WHERE image_id = ?`, fav); n != 1 {
		t.Error("favicon must be kept")
	}
	if n := f.count(`SELECT COUNT(*) FROM favourite_nodes WHERE user_id = ?`, bob); n != bobNodes {
		t.Error("Bob's favourites changed")
	}
	if n := f.count(`SELECT COUNT(*) FROM dashboards WHERE user_id = ?`, bob); n != 1 {
		t.Error("Bob's dashboard is gone")
	}
	if n := f.count(`SELECT COUNT(*) FROM magic_link_tokens`); n != 1 {
		t.Error("Bob's sign-in token is gone")
	}

	// Slug and email are free again.
	if _, err := f.users.SetSlug(ctx, bob, f.ann.Slug); err != nil {
		t.Fatalf("taking the old slug: %v", err)
	}
	again, err := f.users.EnsureByEmail(ctx, f.db, "ann@example.org")
	if err != nil || again.ID == ann || again.DeletionScheduledAt != nil {
		t.Fatalf("new account = %+v, %v", again, err)
	}
}

func TestDeleteLastUser(t *testing.T) {
	f := newFixture(t)
	f.exec(`DELETE FROM users WHERE id = ?`, f.bob.ID)
	f.s.Days = 0
	f.schedule()
	if _, d, err := f.s.Sweep(f.ctx); err != nil || d != 1 || f.count(`SELECT COUNT(*) FROM users`) != 0 {
		t.Fatalf("deleted %d, %v", d, err)
	}
}

func TestFormatDate(t *testing.T) {
	at := time.Date(2026, 3, 8, 9, 5, 0, 0, time.FixedZone("CET", 3600))
	if got := FormatDate("en", at); got != "8 March 2026, 08:05 UTC" {
		t.Errorf("en = %q", got)
	}
	if got := FormatDate("de", at); got != "8. März 2026, 08:05 UTC" {
		t.Errorf("de = %q", got)
	}
}
