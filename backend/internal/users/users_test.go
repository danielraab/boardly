package users

import (
	"context"
	"errors"
	"strings"
	"testing"

	"famodash/internal/dbtest"
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
