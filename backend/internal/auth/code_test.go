package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewCodeUsesTheAlphabet(t *testing.T) {
	seen := map[rune]bool{}
	for range 2000 {
		code, err := newCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != CodeLength {
			t.Fatalf("code %q has length %d", code, len(code))
		}
		for _, r := range code {
			if !strings.ContainsRune(CodeAlphabet, r) {
				t.Fatalf("code %q contains %q", code, r)
			}
			seen[r] = true
		}
	}
	// 12000 draws from 31 symbols: every symbol shows up.
	if len(seen) != len(CodeAlphabet) {
		t.Fatalf("saw %d of %d symbols", len(seen), len(CodeAlphabet))
	}
}

func TestNormalizeCode(t *testing.T) {
	for in, want := range map[string]string{
		"ABC123":      "ABC123",
		"abc 123":     "ABC123",
		" k7q-4mz\n":  "K7Q4MZ",
		"K 7 Q 4 M Z": "K7Q4MZ",
	} {
		if got := NormalizeCode(in); got != want {
			t.Errorf("NormalizeCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatCode(t *testing.T) {
	if got := FormatCode("K7Q4MZ"); got != "K7Q-4MZ" {
		t.Fatalf("FormatCode = %q", got)
	}
}

func TestCodeHashDependsOnTheLink(t *testing.T) {
	if bytes.Equal(codeHash([]byte{1}, "K7Q4MZ"), codeHash([]byte{2}, "K7Q4MZ")) {
		t.Fatal("same code hash for different links")
	}
}
