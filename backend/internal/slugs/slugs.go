// Package slugs validates and generates the URL slugs of users and dashboards.
package slugs

import (
	"crypto/rand"
	"errors"
	"regexp"
	"slices"
	"strings"
)

// MaxLength is the longest slug, in characters.
const MaxLength = 32

// ErrInvalid is returned for a slug that doesn't match the format.
var ErrInvalid = errors.New("invalid slug")

var format = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

// Reserved are the first path segments the web app uses or may use for its own
// routes. A user slug can't be one of them. The frontend keeps a copy that is
// checked against testdata/reserved.json, like this list.
var Reserved = []string{
	"about", "account", "admin", "api", "app", "assets", "auth", "dashboards",
	"docs", "favicon", "favourites", "health", "help", "karakeep", "login",
	"logout", "me", "new", "public", "register", "search", "settings", "share",
	"signup", "static", "tags", "u", "user", "users", "widgets",
}

// Normalize trims and lowercases s and returns ErrInvalid unless the result is
// 1–32 characters of a-z, 0-9 and "-".
func Normalize(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !format.MatchString(s) {
		return "", ErrInvalid
	}
	return s, nil
}

// IsReserved reports whether the (normalized) slug is a reserved word.
func IsReserved(s string) bool {
	return slices.Contains(Reserved, s)
}

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// GeneratedLength is the length of a slug from Generate.
const GeneratedLength = 10

// Generate returns a random slug of GeneratedLength characters from a-z0-9.
func Generate() string {
	b := make([]byte, GeneratedLength)
	for i := range b {
		b[i] = alphabet[randIndex(len(alphabet))]
	}
	return string(b)
}

// randIndex returns a uniform random number in [0, n) for n ≤ 256.
func randIndex(n int) int {
	limit := 256 - 256%n
	var buf [1]byte
	for {
		if _, err := rand.Read(buf[:]); err != nil {
			panic(err)
		}
		if int(buf[0]) < limit {
			return int(buf[0]) % n
		}
	}
}
