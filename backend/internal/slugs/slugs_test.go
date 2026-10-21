package slugs

import (
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
		err      bool
	}{
		{"home", "home", false},
		{" Work ", "work", false},
		{"a", "a", false},
		{"my-board-2", "my-board-2", false},
		{strings.Repeat("a", 32), strings.Repeat("a", 32), false},
		{strings.Repeat("a", 33), "", true},
		{"", "", true},
		{"-a", "-a", false},
		{"a-", "a-", false},
		{"-", "-", false},
		{"a_b", "", true},
		{"a.b", "", true},
		{"küche", "", true},
	}
	for _, tt := range tests {
		got, err := Normalize(tt.in)
		if tt.err {
			if !errors.Is(err, ErrInvalid) {
				t.Errorf("Normalize(%q) err = %v, want ErrInvalid", tt.in, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}

func TestReservedMatchesFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/reserved.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture []string
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fixture, Reserved) {
		t.Fatalf("Reserved = %v, fixture = %v", Reserved, fixture)
	}
	for _, w := range Reserved {
		if _, err := Normalize(w); err != nil {
			t.Errorf("reserved word %q isn't a valid slug", w)
		}
	}
	if !IsReserved("api") || IsReserved("ann") {
		t.Fatal("IsReserved gives wrong answers")
	}
}

func TestGenerate(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		s := Generate()
		if len(s) != GeneratedLength || strings.Trim(s, alphabet) != "" {
			t.Fatalf("Generate() = %q", s)
		}
		seen[s] = true
	}
	if len(seen) < 100 {
		t.Fatalf("only %d distinct slugs in 100 calls", len(seen))
	}
}
