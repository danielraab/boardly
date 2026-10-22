package apptokens

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"famodash/internal/dbtest"
	"famodash/internal/favourites"
	"famodash/internal/users"
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
	if _, err := f.s.Authenticate(f.ctx, secret, "linkwarden"); !errors.Is(err, ErrInvalidToken) {
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
