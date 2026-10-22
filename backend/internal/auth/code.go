package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strings"
	"unicode"
)

// CodeAlphabet are the characters of one-time codes: upper-case letters and
// digits without the look-alikes 0/O and 1/I/L.
const CodeAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

// CodeLength is the number of characters of a one-time code.
const CodeLength = 6

// maxCodeAttempts is how many wrong codes lock the codes of a browser and address.
const maxCodeAttempts = 5

// newCode returns a random one-time code. Each character is drawn uniformly
// from CodeAlphabet.
func newCode() (string, error) {
	n := big.NewInt(int64(len(CodeAlphabet)))
	b := make([]byte, CodeLength)
	for i := range b {
		k, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", fmt.Errorf("generating code: %w", err)
		}
		b[i] = CodeAlphabet[k.Int64()]
	}
	return string(b), nil
}

// NormalizeCode upper-cases an entered code and drops whitespace and hyphens,
// so "abc 123" and "ABC-123" both become "ABC123".
func NormalizeCode(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return unicode.ToUpper(r)
	}, s)
}

// FormatCode splits a code in two halves for reading, such as "ABC 123".
func FormatCode(code string) string {
	half := len(code) / 2
	return code[:half] + " " + code[half:]
}

// codeHash ties a normalized code to its sign-in link's token hash.
func codeHash(tokenHash []byte, code string) []byte {
	h := sha256.New()
	h.Write(tokenHash)
	h.Write([]byte(code))
	return h.Sum(nil)
}
