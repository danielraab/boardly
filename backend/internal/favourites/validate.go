package favourites

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

// Limits, in characters after trimming. They match Karakeep's limits for list
// names and bookmark titles, so the later Karakeep API can pass data through.
const (
	MaxFolderNameLength = 100
	MaxLinkTitleLength  = 1000
	MaxURLLength        = 2048
	MaxTagNameLength    = 100
)

// NormalizeFolderName trims the name; empty names and names longer than
// MaxFolderNameLength return ErrInvalidTitle.
func NormalizeFolderName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxFolderNameLength {
		return "", ErrInvalidTitle
	}
	return trimmed, nil
}

// NormalizeLinkTitle trims the title. Nil, empty and whitespace-only titles
// become nil; titles longer than MaxLinkTitleLength return ErrInvalidTitle.
func NormalizeLinkTitle(title *string) (*string, error) {
	if title == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*title)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxLinkTitleLength {
		return nil, ErrInvalidTitle
	}
	return &trimmed, nil
}

// NormalizeURL trims the URL and checks that it is an absolute http or https
// URL with a host and at most MaxURLLength characters. Other schemes, such as
// javascript:, return ErrInvalidURL because dashboards render these URLs as links.
func NormalizeURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxURLLength {
		return "", ErrInvalidURL
	}
	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrInvalidURL
	}
	return trimmed, nil
}

// NormalizeTags trims every name and drops later duplicates that differ only
// in case. Empty names and names longer than MaxTagNameLength return ErrInvalidTag.
func NormalizeTags(names []string) ([]string, error) {
	out := make([]string, 0, len(names))
	seen := map[string]bool{}
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > MaxTagNameLength {
			return nil, ErrInvalidTag
		}
		key := tagKey(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out, nil
}

// tagKey is the case-insensitive key of an already trimmed tag name.
func tagKey(name string) string {
	return strings.ToLower(name)
}
