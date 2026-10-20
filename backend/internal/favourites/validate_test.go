package favourites

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestNormalizeFolderName(t *testing.T) {
	tests := []struct {
		in, want string
		err      error
	}{
		{"  Work ", "Work", nil},
		{"", "", ErrInvalidTitle},
		{" \t ", "", ErrInvalidTitle},
		{strings.Repeat("ä", 100), strings.Repeat("ä", 100), nil},
		{strings.Repeat("ä", 101), "", ErrInvalidTitle},
	}
	for _, tt := range tests {
		got, err := NormalizeFolderName(tt.in)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("NormalizeFolderName(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
		}
	}
}

func TestNormalizeLinkTitle(t *testing.T) {
	tests := []struct {
		in   *string
		want *string
		err  error
	}{
		{nil, nil, nil},
		{ptr("  "), nil, nil},
		{ptr(" Docs "), ptr("Docs"), nil},
		{ptr(strings.Repeat("a", 1000)), ptr(strings.Repeat("a", 1000)), nil},
		{ptr(strings.Repeat("a", 1001)), nil, ErrInvalidTitle},
	}
	for _, tt := range tests {
		got, err := NormalizeLinkTitle(tt.in)
		if !reflect.DeepEqual(got, tt.want) || !errors.Is(err, tt.err) {
			t.Errorf("NormalizeLinkTitle(%v) = %v, %v; want %v, %v", tt.in, got, err, tt.want, tt.err)
		}
	}
}

func TestNormalizeURL(t *testing.T) {
	long := "https://example.org/" + strings.Repeat("a", 2048-len("https://example.org/"))
	tests := []struct {
		in, want string
		err      error
	}{
		{" https://example.org/docs ", "https://example.org/docs", nil},
		{"http://localhost:8080", "http://localhost:8080", nil},
		{"HTTPS://Example.org", "HTTPS://Example.org", nil},
		{long, long, nil},
		{long + "a", "", ErrInvalidURL},
		{"", "", ErrInvalidURL},
		{"example", "", ErrInvalidURL},
		{"example.org/docs", "", ErrInvalidURL},
		{"javascript:alert(1)", "", ErrInvalidURL},
		{"ftp://example.org", "", ErrInvalidURL},
		{"https://", "", ErrInvalidURL},
		{"https:///path", "", ErrInvalidURL},
		{"http://[::1", "", ErrInvalidURL},
	}
	for _, tt := range tests {
		got, err := NormalizeURL(tt.in)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
		}
	}
}

func TestNormalizeTags(t *testing.T) {
	got, err := NormalizeTags([]string{" Reading ", "reading", "Ärger", "ärger", "tools"})
	if err != nil || !reflect.DeepEqual(got, []string{"Reading", "Ärger", "tools"}) {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := NormalizeTags(nil); err != nil || len(got) != 0 {
		t.Fatalf("nil: got %q, %v", got, err)
	}
	for _, bad := range [][]string{{" "}, {strings.Repeat("a", 101)}} {
		if _, err := NormalizeTags(bad); !errors.Is(err, ErrInvalidTag) {
			t.Errorf("NormalizeTags(%q) err = %v, want ErrInvalidTag", bad, err)
		}
	}
	if _, err := NormalizeTags([]string{strings.Repeat("a", 100)}); err != nil {
		t.Errorf("100 characters: %v", err)
	}
}
