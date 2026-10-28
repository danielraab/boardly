package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	"boardly/internal/dbtest"
	"boardly/internal/favourites"
	"boardly/internal/notes"
)

func ptr(s string) *string { return &s }

func TestEnsureByEmailCreatesOnce(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()

	first, err := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.Email != "ann@example.org" || first.Name != nil {
		t.Fatalf("got %+v", first)
	}
	second, err := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("second call created a new account: %s != %s", second.ID, first.ID)
	}
}

func TestGetByEmailDoesNotCreate(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()

	if _, err := s.GetByEmail(ctx, s.DB, "new@example.org"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown address: err = %v, want ErrNotFound", err)
	}
	if _, err := s.GetByEmail(ctx, s.DB, "new@example.org"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("lookup created an account: err = %v", err)
	}
	ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	got, err := s.GetByEmail(ctx, s.DB, "ann@example.org")
	if err != nil || got.ID != ann.ID {
		t.Fatalf("known address: got %+v, %v", got, err)
	}
}

func TestSetName(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	u, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")

	got, err := s.SetName(ctx, u.ID, ptr("  Ann Smith "))
	if err != nil || got.Name == nil || *got.Name != "Ann Smith" {
		t.Fatalf("got %+v, %v", got, err)
	}

	got, err = s.SetName(ctx, u.ID, ptr("   "))
	if err != nil || got.Name != nil {
		t.Fatalf("whitespace should clear the name, got %+v, %v", got, err)
	}

	if _, err := s.SetName(ctx, u.ID, ptr(strings.Repeat("a", 101))); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("err = %v, want ErrInvalidName", err)
	}

	if _, err := s.SetName(ctx, "missing", nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   *string
		want *string
		err  bool
	}{
		{"nil", nil, nil, false},
		{"empty", ptr(""), nil, false},
		{"whitespace", ptr(" \t "), nil, false},
		{"trimmed", ptr(" Ann "), ptr("Ann"), false},
		{"100 characters", ptr(strings.Repeat("ä", 100)), ptr(strings.Repeat("ä", 100)), false},
		{"101 characters", ptr(strings.Repeat("a", 101)), nil, true},
		{"101 after trim", ptr(" " + strings.Repeat("a", 100) + " "), ptr(strings.Repeat("a", 100)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeName(tt.in)
			if (err != nil) != tt.err {
				t.Fatalf("err = %v", err)
			}
			if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnsureByEmailGeneratesSlug(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()

	u, err := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	if err != nil {
		t.Fatal(err)
	}
	if len(u.Slug) != 10 || strings.Trim(u.Slug, "abcdefghijklmnopqrstuvwxyz0123456789") != "" {
		t.Fatalf("slug = %q, want 10 characters from a-z0-9", u.Slug)
	}
}

func TestEnsureByEmailSlugCollisions(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	// The generator first returns a reserved word, which must be skipped.
	queue := []string{"settings", "x", "x", "x"}
	s.GenerateSlug = func() string {
		next := queue[0]
		queue = queue[1:]
		return next
	}

	var got []string
	for _, email := range []string{"a@example.org", "b@example.org", "c@example.org"} {
		u, err := s.EnsureByEmail(ctx, s.DB, email)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, u.Slug)
	}
	if strings.Join(got, ",") != "x,x-2,x-3" {
		t.Fatalf("slugs = %v, want x, x-2, x-3", got)
	}
}

func TestSetSlug(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	bob, _ := s.EnsureByEmail(ctx, s.DB, "bob@example.org")

	got, err := s.SetSlug(ctx, ann.ID, " Ann-Smith ")
	if err != nil || got.Slug != "ann-smith" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if found, err := s.GetBySlug(ctx, "ANN-SMITH"); err != nil || found.ID != ann.ID {
		t.Fatalf("GetBySlug = %+v, %v", found, err)
	}

	for in, want := range map[string]error{
		"ann-smith": ErrSlugTaken,
		"API":       ErrSlugReserved,
		"b_ob":      ErrInvalidSlug,
	} {
		if _, err := s.SetSlug(ctx, bob.ID, in); !errors.Is(err, want) {
			t.Errorf("SetSlug(%q) err = %v, want %v", in, err, want)
		}
	}
	if u, _ := s.Get(ctx, bob.ID); u.Slug != bob.Slug {
		t.Fatalf("bob's slug changed to %q", u.Slug)
	}
}

func TestUpdateIsAllOrNothing(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	bob, _ := s.EnsureByEmail(ctx, s.DB, "bob@example.org")

	name := ptr("Bob")
	if _, err := s.Update(ctx, bob.ID, Patch{Name: &name, Slug: &ann.Slug}); !errors.Is(err, ErrSlugTaken) {
		t.Fatalf("err = %v, want ErrSlugTaken", err)
	}
	if u, _ := s.Get(ctx, bob.ID); u.Name != nil {
		t.Fatalf("name = %v, want it unchanged", *u.Name)
	}
}

func TestShareFolder(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	favs := favourites.NewStore(s.DB)
	ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	bob, _ := s.EnsureByEmail(ctx, s.DB, "bob@example.org")
	folder := func(userID, parentID, title string) string {
		t.Helper()
		n, err := favs.Create(ctx, userID, favourites.NewNode{ParentID: parentID, Kind: favourites.KindFolder, Title: &title})
		if err != nil {
			t.Fatal(err)
		}
		return n.ID
	}
	set := func(id *string) (User, error) { return s.Update(ctx, ann.ID, Patch{ShareFolderID: &id}) }

	if ann.ShareFolderID != nil {
		t.Fatalf("new user: shareFolderId = %v, want nil", *ann.ShareFolderID)
	}
	inbox := folder(ann.ID, "", "Inbox")
	got, err := set(&inbox)
	if err != nil || got.ShareFolderID == nil || *got.ShareFolderID != inbox {
		t.Fatalf("own folder: got %+v, %v", got, err)
	}

	url := "https://example.org"
	link, _ := favs.Create(ctx, ann.ID, favourites.NewNode{ParentID: inbox, Kind: favourites.KindLink, URL: &url})
	bobs := folder(bob.ID, "", "Bob")
	for name, id := range map[string]string{"link": link.ID, "other user": bobs, "unknown": "nope"} {
		if _, err := set(&id); !errors.Is(err, ErrInvalidFolder) {
			t.Fatalf("%s: err = %v, want ErrInvalidFolder", name, err)
		}
	}
	if u, _ := s.Get(ctx, ann.ID); u.ShareFolderID == nil || *u.ShareFolderID != inbox {
		t.Fatalf("rejected value changed the setting: %+v", u)
	}

	root, _ := favs.EnsureRoot(ctx, s.DB, ann.ID)
	if got, err := set(&root); err != nil || got.ShareFolderID != nil {
		t.Fatalf("root: got %+v, %v, want nil", got, err)
	}
	if _, err := set(&inbox); err != nil {
		t.Fatal(err)
	}
	if got, err := set(nil); err != nil || got.ShareFolderID != nil {
		t.Fatalf("clear: got %+v, %v", got, err)
	}
}

func TestShareFolderClearedOnDelete(t *testing.T) {
	for _, tc := range []struct {
		name   string
		delete func(favs *favourites.Store, userID, work, inbox string) error
	}{
		{"folder itself", func(favs *favourites.Store, userID, _, inbox string) error {
			return favs.Delete(context.Background(), userID, inbox, favourites.DeleteKeepNone)
		}},
		{"with parent", func(favs *favourites.Store, userID, work, _ string) error {
			return favs.Delete(context.Background(), userID, work, favourites.DeleteWithChildren)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStore(dbtest.Open(t))
			ctx := context.Background()
			favs := favourites.NewStore(s.DB)
			ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
			work, _ := favs.Create(ctx, ann.ID, favourites.NewNode{Kind: favourites.KindFolder, Title: ptr("Work")})
			inbox, _ := favs.Create(ctx, ann.ID, favourites.NewNode{ParentID: work.ID, Kind: favourites.KindFolder, Title: ptr("Inbox")})
			id := &inbox.ID
			if _, err := s.Update(ctx, ann.ID, Patch{ShareFolderID: &id}); err != nil {
				t.Fatal(err)
			}
			if err := tc.delete(favs, ann.ID, work.ID, inbox.ID); err != nil {
				t.Fatal(err)
			}
			if u, _ := s.Get(ctx, ann.ID); u.ShareFolderID != nil {
				t.Fatalf("shareFolderId = %v, want nil", *u.ShareFolderID)
			}
		})
	}
}

func TestNoteCategory(t *testing.T) {
	s := NewStore(dbtest.Open(t))
	ctx := context.Background()
	ns := notes.NewStore(s.DB)
	ann, _ := s.EnsureByEmail(ctx, s.DB, "ann@example.org")
	bob, _ := s.EnsureByEmail(ctx, s.DB, "bob@example.org")
	set := func(id *string) (User, error) { return s.Update(ctx, ann.ID, Patch{NoteCategoryID: &id}) }

	if ann.NoteCategoryID != nil {
		t.Fatalf("new user: noteCategoryId = %v, want nil", *ann.NoteCategoryID)
	}
	home, _ := ns.CreateCategory(ctx, ann.ID, notes.NewCategory{Title: "Home"})
	inbox, _ := ns.CreateCategory(ctx, ann.ID, notes.NewCategory{Title: "Inbox", ParentID: &home.ID})
	got, err := set(&inbox.ID)
	if err != nil || got.NoteCategoryID == nil || *got.NoteCategoryID != inbox.ID {
		t.Fatalf("own category: got %+v, %v", got, err)
	}

	bobs, _ := ns.CreateCategory(ctx, bob.ID, notes.NewCategory{Title: "Bob"})
	for name, id := range map[string]string{"other user": bobs.ID, "unknown": "nope"} {
		if _, err := set(&id); !errors.Is(err, ErrInvalidCategory) {
			t.Fatalf("%s: err = %v, want ErrInvalidCategory", name, err)
		}
	}
	if u, _ := s.Get(ctx, ann.ID); u.NoteCategoryID == nil || *u.NoteCategoryID != inbox.ID {
		t.Fatalf("rejected value changed the setting: %+v", u)
	}

	// Deleting the parent moves Inbox up and keeps the setting.
	if err := ns.DeleteCategory(ctx, ann.ID, home.ID); err != nil {
		t.Fatal(err)
	}
	if u, _ := s.Get(ctx, ann.ID); u.NoteCategoryID == nil || *u.NoteCategoryID != inbox.ID {
		t.Fatalf("after deleting the parent: %+v", u)
	}
	if err := ns.DeleteCategory(ctx, ann.ID, inbox.ID); err != nil {
		t.Fatal(err)
	}
	if u, _ := s.Get(ctx, ann.ID); u.NoteCategoryID != nil {
		t.Fatalf("after deleting the category: noteCategoryId = %v, want nil", *u.NoteCategoryID)
	}

	other, _ := ns.CreateCategory(ctx, ann.ID, notes.NewCategory{Title: "Other"})
	if _, err := set(&other.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := set(nil); err != nil || got.NoteCategoryID != nil {
		t.Fatalf("clear: got %+v, %v", got, err)
	}
}
