package apptokens

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/notes"
	"boardly/internal/users"
)

type fixture struct {
	t      *testing.T
	s      *Store
	ctx    context.Context
	now    time.Time
	user   string
	other  string
	folder string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	database := dbtest.Open(t)
	ctx := context.Background()
	u := users.NewStore(database)
	ann, err := u.EnsureByEmail(ctx, database, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := u.EnsureByEmail(ctx, database, "bob@example.org")
	if err != nil {
		t.Fatal(err)
	}
	title := "Reading"
	folder, err := favourites.NewStore(database).Create(ctx, ann.ID, favourites.NewNode{Kind: favourites.KindFolder, Title: &title})
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, ctx: ctx, now: time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC), user: ann.ID, other: bob.ID, folder: folder.ID}
	f.s = &Store{DB: database, Now: func() time.Time { return f.now }}
	return f
}

func (f *fixture) create(expires *time.Time) (Token, string) {
	f.t.Helper()
	tok, secret, err := f.s.Create(f.ctx, f.user, NewToken{Name: "Phone", App: AppKarakeep, RootID: f.folder, ExpiresAt: expires})
	if err != nil {
		f.t.Fatal(err)
	}
	return tok, secret
}

func TestSecretFormat(t *testing.T) {
	f := newFixture(t)
	tok, secret := f.create(nil)
	if !strings.HasPrefix(secret, SecretPrefix) || len(secret) < len(SecretPrefix)+32 {
		t.Errorf("secret = %q, want %q and at least 32 random characters", secret, SecretPrefix)
	}
	if tok.Prefix != secret[:12] {
		t.Errorf("prefix = %q, want %q", tok.Prefix, secret[:12])
	}
	if _, other := f.create(nil); other == secret {
		t.Error("two tokens got the same secret")
	}
}

func TestCreate(t *testing.T) {
	f := newFixture(t)
	tok, _ := f.create(nil)
	if tok.Name != "Phone" || tok.App != AppKarakeep || tok.RootID == nil || *tok.RootID != f.folder ||
		tok.Status != StatusActive || tok.ExpiresAt != nil || tok.LastUsedAt != nil || tok.RevokedAt != nil {
		t.Errorf("token = %+v", tok)
	}

	title := "Docs"
	fav := favourites.NewStore(f.s.DB)
	link, err := fav.Create(f.ctx, f.user, favourites.NewNode{ParentID: f.folder, Kind: favourites.KindLink, Title: &title, URL: ptr("https://example.org")})
	if err != nil {
		t.Fatal(err)
	}
	bobRoot, err := fav.EnsureRoot(f.ctx, f.s.DB, f.other)
	if err != nil {
		t.Fatal(err)
	}
	annRoot, err := fav.EnsureRoot(f.ctx, f.s.DB, f.user)
	if err != nil {
		t.Fatal(err)
	}
	past := f.now.Add(-time.Hour)
	future := f.now.Add(time.Hour)
	for name, tc := range map[string]struct {
		n    NewToken
		want error
	}{
		"top-level root":   {NewToken{Name: "All", App: AppKarakeep, RootID: annRoot}, nil},
		"linkwarden":       {NewToken{Name: "Tablet", App: AppLinkwarden, RootID: f.folder}, nil},
		"future expiry":    {NewToken{Name: "Soon", App: AppKarakeep, RootID: f.folder, ExpiresAt: &future}, nil},
		"trimmed empty":    {NewToken{Name: "   ", App: AppKarakeep, RootID: f.folder}, ErrInvalidName},
		"too long":         {NewToken{Name: strings.Repeat("x", 101), App: AppKarakeep, RootID: f.folder}, ErrInvalidName},
		"unknown app":      {NewToken{Name: "x", App: "pocket", RootID: f.folder}, ErrInvalidApp},
		"link as root":     {NewToken{Name: "x", App: AppKarakeep, RootID: link.ID}, ErrInvalidRoot},
		"other user root":  {NewToken{Name: "x", App: AppKarakeep, RootID: bobRoot}, ErrInvalidRoot},
		"unknown root":     {NewToken{Name: "x", App: AppKarakeep, RootID: "nope"}, ErrInvalidRoot},
		"past expiry":      {NewToken{Name: "x", App: AppKarakeep, RootID: f.folder, ExpiresAt: &past}, ErrInvalidExpiry},
		"expiry right now": {NewToken{Name: "x", App: AppKarakeep, RootID: f.folder, ExpiresAt: &f.now}, ErrInvalidExpiry},
	} {
		if _, _, err := f.s.Create(f.ctx, f.user, tc.n); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}
}

func TestListAndStatus(t *testing.T) {
	f := newFixture(t)
	expires := f.now.Add(time.Hour)
	old, _ := f.create(&expires)
	f.now = f.now.Add(time.Minute)
	revoked, _ := f.create(nil)
	if _, err := f.s.Revoke(f.ctx, f.user, revoked.ID); err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Minute)
	active, _ := f.create(nil)
	if _, _, err := f.s.Create(f.ctx, f.other, NewToken{Name: "x", App: AppKarakeep, RootID: f.folder}); !errors.Is(err, ErrInvalidRoot) {
		t.Fatalf("other user's token: err = %v", err)
	}

	f.now = expires
	list, err := f.s.List(f.ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		id     string
		status Status
	}{{active.ID, StatusActive}, {revoked.ID, StatusRevoked}, {old.ID, StatusExpired}}
	if len(list) != len(want) {
		t.Fatalf("list has %d tokens, want %d", len(list), len(want))
	}
	for i, w := range want {
		if list[i].ID != w.id || list[i].Status != w.status {
			t.Errorf("list[%d] = %s %s, want %s %s", i, list[i].ID, list[i].Status, w.id, w.status)
		}
	}
	if other, err := f.s.List(f.ctx, f.other); err != nil || len(other) != 0 {
		t.Errorf("other user's list = %v, err %v; want empty", other, err)
	}
}

func TestRevokeAndDelete(t *testing.T) {
	f := newFixture(t)
	tok, _ := f.create(nil)

	if err := f.s.Delete(f.ctx, f.user, tok.ID); !errors.Is(err, ErrActive) {
		t.Fatalf("delete active: err = %v, want ErrActive", err)
	}
	if _, err := f.s.Revoke(f.ctx, f.other, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoke other user's: err = %v, want ErrNotFound", err)
	}
	revoked, err := f.s.Revoke(f.ctx, f.user, tok.ID)
	if err != nil || revoked.Status != StatusRevoked || revoked.RevokedAt == nil || !revoked.RevokedAt.Equal(f.now) {
		t.Fatalf("revoke = %+v, err %v", revoked, err)
	}
	f.now = f.now.Add(time.Hour)
	if _, err := f.s.Revoke(f.ctx, f.user, tok.ID); !errors.Is(err, ErrInactive) {
		t.Fatalf("revoke twice: err = %v, want ErrInactive", err)
	}
	if list, _ := f.s.List(f.ctx, f.user); !list[0].RevokedAt.Equal(revoked.RevokedAt.UTC()) {
		t.Errorf("revoked_at changed to %v", list[0].RevokedAt)
	}
	if err := f.s.Delete(f.ctx, f.other, tok.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete other user's: err = %v, want ErrNotFound", err)
	}
	if err := f.s.Delete(f.ctx, f.user, tok.ID); err != nil {
		t.Fatalf("delete revoked: %v", err)
	}
	if list, _ := f.s.List(f.ctx, f.user); len(list) != 0 {
		t.Errorf("list = %v after delete, want empty", list)
	}

	// Expired tokens can't be revoked but can be deleted.
	expires := f.now.Add(time.Hour)
	exp, _ := f.create(&expires)
	f.now = expires
	if _, err := f.s.Revoke(f.ctx, f.user, exp.ID); !errors.Is(err, ErrInactive) {
		t.Fatalf("revoke expired: err = %v, want ErrInactive", err)
	}
	if err := f.s.Delete(f.ctx, f.user, exp.ID); err != nil {
		t.Fatalf("delete expired: %v", err)
	}
}

func TestAuthenticate(t *testing.T) {
	f := newFixture(t)
	expires := f.now.Add(time.Hour)
	tok, secret := f.create(&expires)

	got, err := f.s.Authenticate(f.ctx, secret, AppKarakeep)
	if err != nil || got.ID != tok.ID || got.UserID != f.user || got.LastUsedAt == nil || !got.LastUsedAt.Equal(f.now) {
		t.Fatalf("Authenticate = %+v, err %v", got, err)
	}

	// Last use is recorded at most once a minute.
	first := f.now
	f.now = f.now.Add(30 * time.Second)
	if _, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.s.List(f.ctx, f.user); !list[0].LastUsedAt.Equal(first) {
		t.Errorf("last used = %v after 30s, want %v", list[0].LastUsedAt, first)
	}
	f.now = first.Add(time.Minute)
	if _, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.s.List(f.ctx, f.user); !list[0].LastUsedAt.Equal(f.now) {
		t.Errorf("last used = %v after 1m, want %v", list[0].LastUsedAt, f.now)
	}

	for name, s := range map[string]string{
		"unknown":    SecretPrefix + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"no prefix":  strings.TrimPrefix(secret, SecretPrefix),
		"empty":      "",
		"other case": strings.ToUpper(secret[:1]) + secret[1:],
	} {
		if _, err := f.s.Authenticate(f.ctx, s, AppKarakeep); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
	if _, err := f.s.Authenticate(f.ctx, secret, AppLinkwarden); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("wrong app: err = %v, want ErrInvalidToken", err)
	}

	f.now = expires
	if _, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expired: err = %v, want ErrInvalidToken", err)
	}

	tok2, secret2 := f.create(nil)
	if _, err := f.s.Revoke(f.ctx, f.user, tok2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Authenticate(f.ctx, secret2, AppKarakeep); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("revoked: err = %v, want ErrInvalidToken", err)
	}
}

func ptr[T any](v T) *T { return &v }

func TestNoteScope(t *testing.T) {
	f := newFixture(t)
	ns := notes.NewStore(f.s.DB)
	inbox, err := ns.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: "Inbox"})
	if err != nil {
		t.Fatal(err)
	}
	bobs, err := ns.CreateCategory(f.ctx, f.other, notes.NewCategory{Title: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range map[string]struct {
		app      App
		category *string
		want     NoteScope
		err      error
	}{
		"karakeep, all notes":   {AppKarakeep, nil, NoteScopeAll, nil},
		"karakeep, category":    {AppKarakeep, &inbox.ID, NoteScopeCategory, nil},
		"linkwarden":            {AppLinkwarden, nil, NoteScopeNone, nil},
		"linkwarden, category":  {AppLinkwarden, &inbox.ID, "", ErrInvalidRootCategory},
		"other user's category": {AppKarakeep, &bobs.ID, "", ErrInvalidRootCategory},
		"unknown category":      {AppKarakeep, ptr("nope"), "", ErrInvalidRootCategory},
	} {
		tok, _, err := f.s.Create(f.ctx, f.user, NewToken{Name: "x", App: tc.app, RootID: f.folder, RootCategoryID: tc.category})
		if !errors.Is(err, tc.err) || err == nil && tok.NoteScope != tc.want {
			t.Errorf("%s: scope %q, err %v; want %q, %v", name, tok.NoteScope, err, tc.want, tc.err)
		}
	}
}

func TestUpdate(t *testing.T) {
	f := newFixture(t)
	tok, secret := f.create(nil)
	ns := notes.NewStore(f.s.DB)
	inbox, _ := ns.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: "Inbox"})
	title := "Work"
	work, err := favourites.NewStore(f.s.DB).Create(f.ctx, f.user, favourites.NewNode{Kind: favourites.KindFolder, Title: &title})
	if err != nil {
		t.Fatal(err)
	}

	category := &inbox.ID
	got, err := f.s.Update(f.ctx, f.user, tok.ID, Patch{RootID: &work.ID, RootCategoryID: &category})
	if err != nil || *got.RootID != work.ID || got.NoteScope != NoteScopeCategory || *got.RootCategoryID != inbox.ID {
		t.Fatalf("update: %+v, %v", got, err)
	}
	if auth, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); err != nil || *auth.RootID != work.ID {
		t.Fatalf("authenticate after the update: %+v, %v", auth, err)
	}
	var none *string
	if got, err := f.s.Update(f.ctx, f.user, tok.ID, Patch{RootCategoryID: &none}); err != nil || got.NoteScope != NoteScopeAll || got.RootCategoryID != nil || *got.RootID != work.ID {
		t.Fatalf("back to all notes: %+v, %v", got, err)
	}
	if _, err := f.s.Update(f.ctx, f.other, tok.ID, Patch{RootID: &work.ID}); !errors.Is(err, ErrNotFound) {
		t.Errorf("other user: err = %v", err)
	}
	if _, err := f.s.Update(f.ctx, f.user, tok.ID, Patch{RootID: ptr("nope")}); !errors.Is(err, ErrInvalidRoot) {
		t.Errorf("unknown root: err = %v", err)
	}
	if _, err := f.s.Revoke(f.ctx, f.user, tok.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Update(f.ctx, f.user, tok.ID, Patch{RootID: &f.folder}); !errors.Is(err, ErrInactive) {
		t.Errorf("revoked: err = %v", err)
	}
}

func TestDeletingRootCategoryRevokes(t *testing.T) {
	f := newFixture(t)
	ns := notes.NewStore(f.s.DB)
	home, _ := ns.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: "Home"})
	garden, _ := ns.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: "Garden", ParentID: &home.ID})
	create := func(category string, expires *time.Time) (Token, string) {
		t.Helper()
		tok, secret, err := f.s.Create(f.ctx, f.user, NewToken{Name: "x", App: AppKarakeep, RootID: f.folder, RootCategoryID: &category, ExpiresAt: expires})
		if err != nil {
			t.Fatal(err)
		}
		return tok, secret
	}
	soon := f.now.Add(time.Hour)
	rooted, secret := create(home.ID, nil)
	expired, _ := create(home.ID, &soon)
	sub, _ := create(garden.ID, nil)
	f.now = f.now.Add(2 * time.Hour)

	if err := ns.DeleteCategory(f.ctx, f.user, home.ID); err != nil {
		t.Fatal(err)
	}
	list, err := f.s.List(f.ctx, f.user)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Token{}
	for _, tok := range list {
		byID[tok.ID] = tok
	}
	if got := byID[rooted.ID]; got.Status != StatusRevoked || got.NoteScope != NoteScopeCategory || got.RootCategoryID != nil {
		t.Errorf("rooted token = %+v, want revoked with a deleted category", got)
	}
	if got := byID[expired.ID]; got.Status != StatusExpired || got.RevokedAt != nil {
		t.Errorf("expired token = %+v, want untouched", got)
	}
	if got := byID[sub.ID]; got.Status != StatusActive || *got.RootCategoryID != garden.ID {
		t.Errorf("subcategory token = %+v, want active", got)
	}
	if _, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("authenticate the revoked token: err = %v", err)
	}
}

func TestMemosTokens(t *testing.T) {
	f := newFixture(t)
	ns := notes.NewStore(f.s.DB)
	inbox, _ := ns.CreateCategory(f.ctx, f.user, notes.NewCategory{Title: "Inbox"})

	all, secret, err := f.s.Create(f.ctx, f.user, NewToken{Name: "MoeMemos", App: AppMemos})
	if err != nil || all.App != AppMemos || all.RootID != nil || all.NoteScope != NoteScopeAll {
		t.Fatalf("memos token = %+v, %v; want no root and all notes", all, err)
	}
	if auth, err := f.s.Authenticate(f.ctx, secret, AppMemos); err != nil || auth.ID != all.ID {
		t.Errorf("authenticate = %+v, %v", auth, err)
	}
	if _, err := f.s.Authenticate(f.ctx, secret, AppKarakeep); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("as Karakeep: err = %v, want ErrInvalidToken", err)
	}
	inInbox, _, err := f.s.Create(f.ctx, f.user, NewToken{Name: "Inbox", App: AppMemos, RootCategoryID: &inbox.ID})
	if err != nil || inInbox.NoteScope != NoteScopeCategory || *inInbox.RootCategoryID != inbox.ID {
		t.Errorf("with a category = %+v, %v", inInbox, err)
	}
	if _, _, err := f.s.Create(f.ctx, f.user, NewToken{Name: "x", App: AppMemos, RootID: f.folder}); !errors.Is(err, ErrInvalidRoot) {
		t.Errorf("with a root folder: err = %v, want ErrInvalidRoot", err)
	}
	if _, _, err := f.s.Create(f.ctx, f.user, NewToken{Name: "x", App: AppKarakeep}); !errors.Is(err, ErrInvalidRoot) {
		t.Errorf("Karakeep without a root folder: err = %v, want ErrInvalidRoot", err)
	}

	category := &inbox.ID
	got, err := f.s.Update(f.ctx, f.user, all.ID, Patch{RootID: ptr(""), RootCategoryID: &category})
	if err != nil || got.RootID != nil || got.NoteScope != NoteScopeCategory {
		t.Fatalf("change the category = %+v, %v", got, err)
	}
	if _, err := f.s.Update(f.ctx, f.user, all.ID, Patch{RootID: &f.folder}); !errors.Is(err, ErrInvalidRoot) {
		t.Errorf("set a root folder: err = %v, want ErrInvalidRoot", err)
	}
	// Deleting the category revokes the token, as for Karakeep.
	if err := ns.DeleteCategory(f.ctx, f.user, inbox.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.Authenticate(f.ctx, secret, AppMemos); !errors.Is(err, ErrInvalidToken) {
		t.Errorf("after deleting its category: err = %v, want ErrInvalidToken", err)
	}
}
