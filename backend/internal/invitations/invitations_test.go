package invitations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/groups"
	"boardly/internal/jobs"
	"boardly/internal/mail"
)

const stamp = "2026-03-01T12:00:00.000Z"

var (
	ctx = context.Background()
	ann = Actor{ID: "ann", Email: "ann@example.org"}
	bob = Actor{ID: "bob", Email: "bob@example.org"}
	eve = Actor{ID: "eve", Email: "eve@example.org"}
)

type outbox struct {
	mu   sync.Mutex
	sent []mail.Message
	fail bool
}

func (o *outbox) Send(_ context.Context, m mail.Message) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fail {
		return errors.New("smtp down")
	}
	o.sent = append(o.sent, m)
	return nil
}

type fixture struct {
	t     *testing.T
	s     *Store
	db    *sql.DB
	now   time.Time
	mails *outbox
}

func setup(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	f := &fixture{t: t, db: database, now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), mails: &outbox{}}
	f.exec(`INSERT INTO users (id, email, name, slug, created_at, updated_at) VALUES
	  ('ann', 'ann@example.org', 'Ann', 'ann', ?, ?),
	  ('bob', 'bob@example.org', NULL, 'bob-slug', ?, ?),
	  ('eve', 'eve@example.org', 'Eve', 'eve', ?, ?)`, stamp, stamp, stamp, stamp, stamp, stamp)
	f.exec(`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES
	  ('work', 'ann', 'work', 'Work', ?, ?)`, stamp, stamp)
	f.exec(`INSERT INTO user_groups (id, name, created_at, updated_at) VALUES ('family', 'Family', ?, ?)`, stamp, stamp)
	f.exec(`INSERT INTO user_group_members VALUES ('family', 'ann', 'admin', ?), ('family', 'eve', 'member', ?)`, stamp, stamp)
	base, _ := url.Parse("https://dash.example.org")
	queue := jobs.New(database)
	queue.Now = f.clock
	f.s = &Store{DB: database, Now: f.clock, Jobs: queue, Mailer: f.mails, BaseURL: base}
	return f
}

func (f *fixture) clock() time.Time { return f.now }

func (f *fixture) exec(q string, args ...any) {
	f.t.Helper()
	if _, err := f.db.Exec(q, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) count(q string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRow(q, args...).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return n
}

func (f *fixture) inviteDash(email string) Invitation {
	f.t.Helper()
	inv, err := f.s.InviteToDashboard(ctx, ann, "work", email, "read", "en")
	if err != nil {
		f.t.Fatal(err)
	}
	return inv
}

func (f *fixture) deliver(id string) {
	f.t.Helper()
	if err := f.s.Deliver(ctx, id); err != nil {
		f.t.Fatal(err)
	}
}

var linkRE = regexp.MustCompile(`/invitation\?token=([A-Za-z0-9_-]+)`)

func (f *fixture) lastToken() string {
	f.t.Helper()
	if len(f.mails.sent) == 0 {
		f.t.Fatal("no mail")
	}
	m := linkRE.FindStringSubmatch(f.mails.sent[len(f.mails.sent)-1].Text)
	if m == nil {
		f.t.Fatalf("no link in %q", f.mails.sent[len(f.mails.sent)-1].Text)
	}
	return m[1]
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Errorf("error = %v, want %v", got, want)
	}
}

func TestCreateErrors(t *testing.T) {
	f := setup(t)
	_, err := f.s.InviteToDashboard(ctx, bob, "work", "x@example.org", "read", "en")
	wantErr(t, err, ErrNotFound)
	_, err = f.s.InviteToDashboard(ctx, ann, "work", "not an address", "read", "en")
	wantErr(t, err, ErrInvalidEmail)
	_, err = f.s.InviteToDashboard(ctx, ann, "work", "x@example.org", "edit", "en")
	wantErr(t, err, ErrInvalidLevel)
	_, err = f.s.InviteToDashboard(ctx, ann, "work", " ANN@example.org ", "read", "en")
	wantErr(t, err, ErrCannotSelf)
	_, err = f.s.InviteToGroup(ctx, bob, "family", "x@example.org", "", "en")
	wantErr(t, err, groups.ErrNotFound)
	_, err = f.s.InviteToGroup(ctx, eve, "family", "x@example.org", "", "en")
	wantErr(t, err, groups.ErrForbidden)
	_, err = f.s.InviteToGroup(ctx, ann, "family", "x@example.org", "owner", "en")
	wantErr(t, err, groups.ErrInvalidRole)
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 0 {
		t.Errorf("invitations = %d", n)
	}
}

func TestCreateAndAlreadyInvited(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash(" Bob@Example.org ")
	if inv.Email != "bob@example.org" || inv.Level != "read" || inv.Role != "" {
		t.Fatalf("invitation %+v", inv)
	}
	if want := f.now.Add(TTL).UTC().Format("2006-01-02T15:04:05.000Z"); inv.ExpiresAt != want {
		t.Errorf("expires %s, want %s", inv.ExpiresAt, want)
	}
	_, err := f.s.InviteToDashboard(ctx, ann, "work", "bob@example.org", "read", "en")
	wantErr(t, err, ErrAlreadyInvited)

	g, err := f.s.InviteToGroup(ctx, ann, "family", "bob@example.org", "", "en")
	if err != nil || g.Role != groups.RoleMember {
		t.Fatalf("group invitation %+v %v", g, err)
	}
	if n := f.count(`SELECT COUNT(*) FROM jobs WHERE kind = 'invitation_mail'`); n != 2 {
		t.Errorf("jobs = %d, want 2", n)
	}
	var payload string
	if err := f.db.QueryRow(`SELECT payload FROM jobs LIMIT 1`).Scan(&payload); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(payload, "example.org") {
		t.Errorf("job payload contains the address: %s", payload)
	}
}

func TestExpiredInvitationDoesNotBlockANewOne(t *testing.T) {
	f := setup(t)
	f.inviteDash("x@example.org")
	f.now = f.now.Add(TTL + time.Minute)
	f.inviteDash("x@example.org")
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 1 {
		t.Errorf("invitations = %d, want 1", n)
	}
}

func TestDeliverToRegisteredUser(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.deliver(inv.ID)
	if len(f.mails.sent) != 1 {
		t.Fatalf("mails = %d", len(f.mails.sent))
	}
	m := f.mails.sent[0]
	if m.To != "bob@example.org" || !strings.Contains(m.Text, "Ann") || !strings.Contains(m.Text, "Work") {
		t.Errorf("mail %+v", m)
	}
	if strings.Contains(m.Text, "ann@example.org") || strings.Contains(m.HTML, "ann@example.org") {
		t.Error("mail contains the inviter's address")
	}
	token := f.lastToken()
	if n := f.count(`SELECT COUNT(*) FROM invitations WHERE token_hash = ?`, hashToken(token)); n != 1 {
		t.Error("the hash of the token isn't stored")
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations WHERE token_hash = ?`, token); n != 0 {
		t.Error("the plain token is stored")
	}
}

func TestDeliverRetryReplacesToken(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.deliver(inv.ID)
	first := f.lastToken()
	f.deliver(inv.ID)
	second := f.lastToken()
	if first == second {
		t.Fatal("the retry reused the token")
	}
	_, err := f.s.Preview(ctx, bob, first)
	wantErr(t, err, ErrInvitationNotFound)
	if _, err := f.s.Preview(ctx, bob, second); err != nil {
		t.Errorf("new token: %v", err)
	}
}

func TestNoticeToUnknownAddress(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("new@example.org")
	f.deliver(inv.ID)
	if len(f.mails.sent) != 1 {
		t.Fatalf("mails = %d", len(f.mails.sent))
	}
	m := f.mails.sent[0]
	for _, part := range []string{m.Subject, m.Text, m.HTML} {
		for _, secret := range []string{"Ann", "ann@example.org", "Work", "/invitation"} {
			if strings.Contains(part, secret) {
				t.Errorf("notice contains %q: %s", secret, part)
			}
		}
	}
	if !strings.Contains(m.Text, "within 14 days") {
		t.Errorf("notice doesn't name the 14 days: %s", m.Text)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations WHERE id = ? AND token_hash IS NULL`, inv.ID); n != 1 {
		t.Error("a notice must not create a token")
	}
}

func TestNoticeClosedRegistration(t *testing.T) {
	f := setup(t)
	f.s.RegistrationClosed = true
	f.deliver(f.inviteDash("new@example.org").ID)
	if strings.Contains(f.mails.sent[0].Text, "sign up") {
		t.Errorf("notice suggests signing up: %s", f.mails.sent[0].Text)
	}
}

func TestOneNoticePerDay(t *testing.T) {
	f := setup(t)
	first := f.inviteDash("new@example.org")
	f.deliver(first.ID)
	group, err := f.s.InviteToGroup(ctx, ann, "family", "new@example.org", "", "en")
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(group.ID)
	if len(f.mails.sent) != 1 {
		t.Fatalf("mails = %d, want 1", len(f.mails.sent))
	}

	// Revoking doesn't reset the limit.
	if err := f.s.RevokeDashboard(ctx, "ann", "work", first.ID); err != nil {
		t.Fatal(err)
	}
	f.deliver(f.inviteDash("new@example.org").ID)
	if len(f.mails.sent) != 1 {
		t.Fatalf("after revoking: mails = %d, want 1", len(f.mails.sent))
	}

	// A held-back invitation can still be accepted after signing up.
	f.now = f.now.Add(time.Hour)
	f.exec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('new', 'new@example.org', 'new', ?, ?)`, stamp, stamp)
	list, err := f.s.ListOpen(ctx, "new@example.org")
	if err != nil || len(list) != 2 {
		t.Fatalf("open %+v %v", list, err)
	}
	if _, err := f.s.Accept(ctx, Actor{ID: "new", Email: "new@example.org"}, "", group.ID); err != nil {
		t.Errorf("accept: %v", err)
	}
}

func TestNoticeAgainAfterADay(t *testing.T) {
	f := setup(t)
	first := f.inviteDash("new@example.org")
	f.deliver(first.ID)
	f.now = f.now.Add(NoticeInterval + time.Minute)
	f.deliver(first.ID)
	if len(f.mails.sent) != 2 {
		t.Fatalf("mails = %d, want 2", len(f.mails.sent))
	}
}

func TestFailedNoticeIsNotCounted(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("new@example.org")
	f.mails.fail = true
	if err := f.s.Deliver(ctx, inv.ID); err == nil {
		t.Fatal("no error")
	}
	f.mails.fail = false
	f.deliver(inv.ID)
	if len(f.mails.sent) != 1 {
		t.Fatalf("mails = %d, want 1", len(f.mails.sent))
	}
}

func TestDeliverSkipsRevoked(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	if err := f.s.RevokeDashboard(ctx, "ann", "work", inv.ID); err != nil {
		t.Fatal(err)
	}
	f.deliver(inv.ID)
	if len(f.mails.sent) != 0 {
		t.Fatalf("mails = %d", len(f.mails.sent))
	}
}

func TestRunMailJob(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	payload, _ := json.Marshal(jobs.InvitationMailPayload{InvitationID: inv.ID})
	if err := f.s.RunMailJob(ctx, jobs.Job{Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if len(f.mails.sent) != 1 {
		t.Fatalf("mails = %d", len(f.mails.sent))
	}
}

func TestAcceptDashboardByToken(t *testing.T) {
	f := setup(t)
	f.deliver(f.inviteDash("bob@example.org").ID)
	token := f.lastToken()

	p, err := f.s.Preview(ctx, bob, token)
	if err != nil || p.Type != TypeDashboard || p.Title != "Work" || p.InviterName != "Ann" || p.Level != "read" {
		t.Fatalf("preview %+v %v", p, err)
	}

	got, err := f.s.Accept(ctx, bob, token, "")
	if err != nil || got.Type != TypeDashboard || got.OwnerSlug != "ann" || got.Slug != "work" {
		t.Fatalf("accept %+v %v", got, err)
	}
	if n := f.count(`SELECT COUNT(*) FROM shares WHERE user_id = 'bob' AND dashboard_id = 'work' AND owner_id = 'ann' AND level = 'read'`); n != 1 {
		t.Errorf("shares = %d", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 0 {
		t.Errorf("invitations = %d", n)
	}
	_, err = f.s.Accept(ctx, bob, token, "")
	wantErr(t, err, ErrInvitationNotFound)
}

func TestAcceptGroupKeepsRole(t *testing.T) {
	f := setup(t)
	inv, err := f.s.InviteToGroup(ctx, ann, "family", "bob@example.org", groups.RoleAdmin, "en")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.s.Accept(ctx, bob, "", inv.ID)
	if err != nil || got.Type != TypeGroup || got.GroupID != "family" {
		t.Fatalf("accept %+v %v", got, err)
	}
	if n := f.count(`SELECT COUNT(*) FROM user_group_members WHERE user_id = 'bob' AND role = 'admin'`); n != 1 {
		t.Errorf("member rows = %d", n)
	}

	// An invitation with another role leaves an existing membership alone.
	f.exec(`UPDATE user_group_members SET role = 'admin' WHERE user_id = 'eve'`)
	inv, err = f.s.InviteToGroup(ctx, ann, "family", "eve@example.org", groups.RoleMember, "en")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Accept(ctx, eve, "", inv.ID); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM user_group_members WHERE user_id = 'eve' AND role = 'admin'`); n != 1 {
		t.Error("the role of an existing member changed")
	}
}

func TestAcceptExistingShareSucceeds(t *testing.T) {
	f := setup(t)
	f.exec(`INSERT INTO shares (id, owner_id, dashboard_id, user_id, level, created_at) VALUES ('s', 'ann', 'work', 'bob', 'read', ?)`, stamp)
	inv := f.inviteDash("bob@example.org")
	if _, err := f.s.Accept(ctx, bob, "", inv.ID); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM shares WHERE id = 's'`); n != 1 {
		t.Error("existing share changed")
	}
	if n := f.count(`SELECT COUNT(*) FROM shares`); n != 1 {
		t.Error("second share created")
	}
}

func TestAcceptWrongAccount(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.deliver(inv.ID)
	token := f.lastToken()

	_, err := f.s.Preview(ctx, eve, token)
	wantErr(t, err, ErrWrongAccount)
	_, err = f.s.Accept(ctx, eve, token, "")
	wantErr(t, err, ErrWrongAccount)
	_, err = f.s.Accept(ctx, eve, "", inv.ID)
	wantErr(t, err, ErrInvitationNotFound)
	if n := f.count(`SELECT COUNT(*) FROM shares`); n != 0 {
		t.Errorf("shares = %d", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 1 {
		t.Errorf("invitations = %d", n)
	}
}

func TestAcceptByIDUsesUpTheLink(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.deliver(inv.ID)
	token := f.lastToken()
	if _, err := f.s.Accept(ctx, bob, "", inv.ID); err != nil {
		t.Fatal(err)
	}
	_, err := f.s.Accept(ctx, bob, token, "")
	wantErr(t, err, ErrInvitationNotFound)
}

func TestAcceptExpired(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.deliver(inv.ID)
	token := f.lastToken()
	f.now = f.now.Add(TTL + time.Minute)
	_, err := f.s.Accept(ctx, bob, token, "")
	wantErr(t, err, ErrInvitationNotFound)
	_, err = f.s.Accept(ctx, bob, "", inv.ID)
	wantErr(t, err, ErrInvitationNotFound)
	_, err = f.s.Preview(ctx, bob, token)
	wantErr(t, err, ErrInvitationNotFound)
}

func TestOwnerScheduledForDeletion(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	f.exec(`UPDATE users SET delete_after = ? WHERE id = 'ann'`, stamp)
	_, err := f.s.Accept(ctx, bob, "", inv.ID)
	wantErr(t, err, ErrInvitationNotFound)
	if list, err := f.s.ListOpen(ctx, bob.Email); err != nil || len(list) != 0 {
		t.Fatalf("open %+v %v", list, err)
	}
}

func TestListOpen(t *testing.T) {
	f := setup(t)
	dash := f.inviteDash("new@example.org")
	f.now = f.now.Add(time.Minute)
	group, err := f.s.InviteToGroup(ctx, ann, "family", "new@example.org", "", "en")
	if err != nil {
		t.Fatal(err)
	}
	f.inviteDash("other@example.org")

	f.exec(`INSERT INTO users (id, email, slug, created_at, updated_at) VALUES ('new', 'new@example.org', 'new', ?, ?)`, stamp, stamp)
	list, err := f.s.ListOpen(ctx, "new@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != group.ID || list[1].ID != dash.ID {
		t.Fatalf("list %+v", list)
	}
	if list[1].Type != TypeDashboard || list[1].Title != "Work" || list[1].InviterName != "Ann" || list[1].Level != "read" || list[1].ExpiresAt == "" {
		t.Errorf("dashboard entry %+v", list[1])
	}
	if list[0].Type != TypeGroup || list[0].Title != "Family" || list[0].Role != "member" {
		t.Errorf("group entry %+v", list[0])
	}
	if n := f.count(`SELECT COUNT(*) FROM shares`); n != 0 {
		t.Errorf("signing up gave access: %d shares", n)
	}

	if list, err := f.s.ListOpen(ctx, "carol@example.org"); err != nil || len(list) != 0 {
		t.Fatalf("carol %+v %v", list, err)
	}
	f.now = f.now.Add(TTL + time.Hour)
	if list, err := f.s.ListOpen(ctx, "new@example.org"); err != nil || len(list) != 0 {
		t.Fatalf("expired %+v %v", list, err)
	}
}

func TestInviterNameFallsBackToSlug(t *testing.T) {
	f := setup(t)
	f.exec(`INSERT INTO dashboards (id, user_id, slug, title, created_at, updated_at) VALUES ('bobs', 'bob', 'b', 'Bobs', ?, ?)`, stamp, stamp)
	inv, err := f.s.InviteToDashboard(ctx, bob, "bobs", "eve@example.org", "read", "en")
	if err != nil {
		t.Fatal(err)
	}
	f.deliver(inv.ID)
	if !strings.Contains(f.mails.sent[0].Text, "bob-slug") {
		t.Errorf("mail %s", f.mails.sent[0].Text)
	}
}

func TestRevoke(t *testing.T) {
	f := setup(t)
	inv := f.inviteDash("bob@example.org")
	wantErr(t, f.s.RevokeDashboard(ctx, "bob", "work", inv.ID), ErrNotFound)
	wantErr(t, f.s.RevokeDashboard(ctx, "ann", "work", "nope"), ErrNotFound)
	if err := f.s.RevokeDashboard(ctx, "ann", "work", inv.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.s.ListForDashboard(ctx, "work"); len(list) != 0 {
		t.Errorf("list %+v", list)
	}

	g, err := f.s.InviteToGroup(ctx, ann, "family", "bob@example.org", "", "en")
	if err != nil {
		t.Fatal(err)
	}
	wantErr(t, f.s.RevokeGroup(ctx, "eve", "family", g.ID), groups.ErrForbidden)
	wantErr(t, f.s.RevokeGroup(ctx, "bob", "family", g.ID), groups.ErrNotFound)
	if err := f.s.RevokeGroup(ctx, "ann", "family", g.ID); err != nil {
		t.Fatal(err)
	}
}

func TestListForHidesExpired(t *testing.T) {
	f := setup(t)
	f.inviteDash("bob@example.org")
	f.now = f.now.Add(TTL + time.Minute)
	if list, err := f.s.ListForDashboard(ctx, "work"); err != nil || len(list) != 0 {
		t.Fatalf("list %+v %v", list, err)
	}
}

func TestSweep(t *testing.T) {
	f := setup(t)
	f.deliver(f.inviteDash("new@example.org").ID)
	f.now = f.now.Add(NoticeInterval + time.Minute)
	f.inviteDash("keep@example.org")
	if err := f.s.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitation_notices`); n != 0 {
		t.Errorf("notices = %d, want 0", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 2 {
		t.Errorf("open invitations were swept: %d", n)
	}
	f.now = f.now.Add(TTL + 25*time.Hour)
	if err := f.s.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 0 {
		t.Errorf("invitations = %d, want 0", n)
	}
}

func TestDeleteForAddress(t *testing.T) {
	f := setup(t)
	f.deliver(f.inviteDash("new@example.org").ID)
	f.inviteDash("keep@example.org")
	if err := DeleteForAddress(ctx, f.db, "new@example.org"); err != nil {
		t.Fatal(err)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations WHERE email = 'new@example.org'`); n != 0 {
		t.Errorf("invitations = %d", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitation_notices`); n != 0 {
		t.Errorf("notices = %d", n)
	}
	if n := f.count(`SELECT COUNT(*) FROM invitations`); n != 1 {
		t.Errorf("others' invitations removed: %d", n)
	}
}

func TestGroupInvitationSurvivesInviterLeaving(t *testing.T) {
	f := setup(t)
	f.exec(`UPDATE user_group_members SET role = 'admin' WHERE user_id = 'eve'`)
	inv, err := f.s.InviteToGroup(ctx, ann, "family", "bob@example.org", "", "en")
	if err != nil {
		t.Fatal(err)
	}
	store := groups.NewStore(f.db)
	if err := store.RemoveMember(ctx, "ann", "family", "ann"); err != nil {
		t.Fatal(err)
	}
	if err := f.s.RevokeGroup(ctx, "eve", "family", inv.ID); err != nil {
		t.Errorf("remaining admin can't revoke: %v", err)
	}
}
