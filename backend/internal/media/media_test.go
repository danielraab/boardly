package media

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"boardly/internal/dbtest"
	"boardly/internal/notes"
	"boardly/internal/users"
)

type fixture struct {
	t     *testing.T
	db    *sql.DB
	s     *Store
	ctx   context.Context
	user  string
	other string
	logos map[string]string // logo URL → image ID the fake fetcher returns
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
	f := &fixture{t: t, db: database, ctx: ctx, user: ann.ID, other: bob.ID, logos: map[string]string{}}
	check := func(_ context.Context, rawURL string) (string, error) {
		if rawURL == "https://bad.example/" {
			return "", ErrUnreachable
		}
		return rawURL, nil
	}
	fetchLogo := func(_ context.Context, userID, rawURL string) (string, error) {
		if id, ok := f.logos[rawURL]; ok {
			return id, nil
		}
		return "", errors.New("no logo")
	}
	f.s = NewStore(database, check, fetchLogo)
	return f
}

func (f *fixture) create(title string) Source {
	f.t.Helper()
	src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindStream, Title: title, URL: "https://stream.example/" + title})
	if err != nil {
		f.t.Fatal(err)
	}
	return src
}

func (f *fixture) titles(userID string) []string {
	f.t.Helper()
	list, err := f.s.List(f.ctx, userID)
	if err != nil {
		f.t.Fatal(err)
	}
	var out []string
	for _, s := range list {
		out = append(out, s.Title)
	}
	return out
}

func (f *fixture) image(userID string) string {
	f.t.Helper()
	id := "img-" + userID
	if _, err := f.db.ExecContext(f.ctx,
		`INSERT INTO images (id, source, user_id, content_type, size, created_at) VALUES (?, 'upload', ?, 'image/png', 1, '2026-01-01T00:00:00.000Z')`,
		id, userID); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func TestCreateAppendsAndTrims(t *testing.T) {
	f := newFixture(t)
	f.create("FM4")
	src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindStream, Title: "  Jazz Radio ", URL: "https://jazz.example/live"})
	if err != nil {
		t.Fatal(err)
	}
	if src.Title != "Jazz Radio" || src.Position != 1 || src.Kind != KindStream || src.URL != "https://jazz.example/live" {
		t.Fatalf("source = %+v", src)
	}
	if got := f.titles(f.user); len(got) != 2 || got[0] != "FM4" || got[1] != "Jazz Radio" {
		t.Fatalf("titles = %v", got)
	}
}

func TestCreateValidation(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		in   NewSource
		want error
	}{
		{NewSource{Kind: "embed", Title: "X", URL: "https://a.example/"}, ErrInvalidKind},
		{NewSource{Kind: KindStream, Title: "   ", URL: "https://a.example/"}, ErrInvalidTitle},
		{NewSource{Kind: KindStream, Title: string(make([]rune, 201)), URL: "https://a.example/"}, ErrInvalidTitle},
		{NewSource{Kind: KindStream, Title: "X", URL: "https://bad.example/"}, ErrUnreachable},
		{NewSource{Kind: KindStream, Title: "X", URL: "https://a.example/", Icon: &notes.Icon{Type: "symbol", Name: "no-such-symbol"}}, ErrInvalidIcon},
	}
	for _, c := range cases {
		if _, err := f.s.Create(f.ctx, f.user, c.in); !errors.Is(err, c.want) {
			t.Errorf("%+v: err = %v, want %v", c.in, err, c.want)
		}
	}
	if got := f.titles(f.user); len(got) != 0 {
		t.Fatalf("titles = %v, want none", got)
	}
}

func TestCreateWithLogo(t *testing.T) {
	f := newFixture(t)
	img := f.image(f.user)
	f.logos["https://logo.example/fm4.png"] = img
	src, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindStream, Title: "FM4", URL: "https://fm4.example/", LogoURL: "https://logo.example/fm4.png"})
	if err != nil {
		t.Fatal(err)
	}
	if src.Icon == nil || src.Icon.Type != notes.IconImage || src.Icon.ImageID != img {
		t.Fatalf("icon = %+v", src.Icon)
	}
	broken, err := f.s.Create(f.ctx, f.user, NewSource{Kind: KindStream, Title: "Other", URL: "https://o.example/", LogoURL: "https://logo.example/404.png"})
	if err != nil {
		t.Fatal(err)
	}
	if broken.Icon != nil {
		t.Fatalf("icon = %+v, want none for a broken logo", broken.Icon)
	}
}

func TestOtherUsersSource(t *testing.T) {
	f := newFixture(t)
	src := f.create("FM4")
	title := "Mine"
	if _, err := f.s.Update(f.ctx, f.other, src.ID, Patch{Title: &title}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update err = %v", err)
	}
	if err := f.s.Delete(f.ctx, f.other, src.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete err = %v", err)
	}
	if _, err := f.s.Get(f.ctx, f.other, src.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get err = %v", err)
	}
	if got := f.titles(f.user); len(got) != 1 || got[0] != "FM4" {
		t.Fatalf("titles = %v", got)
	}
}

func TestUpdate(t *testing.T) {
	f := newFixture(t)
	src := f.create("FM4")
	title, url := "FM4 Wien", "https://new.example/live"
	icon := &notes.Icon{Type: notes.IconSymbol, Name: "music"}
	got, err := f.s.Update(f.ctx, f.user, src.ID, Patch{Title: &title, URL: &url, Icon: &icon})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "FM4 Wien" || got.URL != url || got.Icon == nil || got.Icon.Name != "music" {
		t.Fatalf("source = %+v", got)
	}
	var none *notes.Icon
	if got, err = f.s.Update(f.ctx, f.user, src.ID, Patch{Icon: &none}); err != nil || got.Icon != nil {
		t.Fatalf("remove icon: %+v %v", got, err)
	}
	bad := "https://bad.example/"
	if _, err := f.s.Update(f.ctx, f.user, src.ID, Patch{URL: &bad}); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("bad url err = %v", err)
	}
	empty := " "
	if _, err := f.s.Update(f.ctx, f.user, src.ID, Patch{Title: &empty}); !errors.Is(err, ErrInvalidTitle) {
		t.Fatalf("empty title err = %v", err)
	}
}

func TestReorder(t *testing.T) {
	f := newFixture(t)
	a, b, c := f.create("A"), f.create("B"), f.create("C")
	if err := f.s.Reorder(f.ctx, f.user, []string{c.ID, a.ID, b.ID}); err != nil {
		t.Fatal(err)
	}
	if got := f.titles(f.user); got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Fatalf("titles = %v", got)
	}
	for _, ids := range [][]string{{a.ID, b.ID}, {a.ID, b.ID, b.ID}, {a.ID, b.ID, "unknown"}} {
		if err := f.s.Reorder(f.ctx, f.user, ids); !errors.Is(err, ErrInvalidOrder) {
			t.Errorf("%v: err = %v", ids, err)
		}
	}
	if got := f.titles(f.user); got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Fatalf("order changed after a rejected reorder: %v", got)
	}
}

func TestByIDsKeepsOrderAndSkipsMissing(t *testing.T) {
	f := newFixture(t)
	a, b := f.create("A"), f.create("B")
	got, err := f.s.ByIDs(f.ctx, f.user, []string{b.ID, "gone", a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != b.ID || got[1].ID != a.ID {
		t.Fatalf("got %+v", got)
	}
	if got, _ := f.s.ByIDs(f.ctx, f.other, []string{a.ID}); len(got) != 0 {
		t.Fatalf("another user's sources: %+v", got)
	}
}

func TestDeleteUserDeletesSources(t *testing.T) {
	f := newFixture(t)
	f.create("FM4")
	if _, err := f.db.ExecContext(f.ctx, `DELETE FROM users WHERE id = ?`, f.user); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := f.db.QueryRowContext(f.ctx, `SELECT COUNT(*) FROM media_sources`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("count = %d, err = %v", n, err)
	}
}
