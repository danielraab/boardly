package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	"boardly/internal/dbtest"
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
