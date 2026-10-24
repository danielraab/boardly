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
	// MaxDescriptionLength matches Linkwarden's limit for link descriptions.
	MaxDescriptionLength = 2048
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

// NormalizeDescription trims the description. Nil, empty and
// whitespace-only descriptions become nil; descriptions longer than
// MaxDescriptionLength return ErrInvalidDescription.
func NormalizeDescription(description *string) (*string, error) {
	if description == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*description)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxDescriptionLength {
		return nil, ErrInvalidDescription
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

// SameURL reports whether a and b are the same page: they are compared without
// the fragment and without a trailing slash of the path, and with the scheme
// and host in lower case. Karakeep and Linkwarden clients use this to check
// whether a page is already saved.
func SameURL(a, b string) bool {
	return comparableURL(a) == comparableURL(b)
}

// Origin returns scheme://host[:port] of an http or https URL, lower-cased
// and without the default port, or ErrInvalidURL.
func Origin(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", ErrInvalidURL
	}
	scheme, host, port := strings.ToLower(u.Scheme), strings.ToLower(u.Hostname()), u.Port()
	if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host, nil
}

func comparableURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return raw
	}
	u.Fragment, u.RawFragment = "", ""
	u.Scheme, u.Host = strings.ToLower(u.Scheme), strings.ToLower(u.Host)
	if u.Path == "" {
		u.Path = "/"
	}
	if len(u.Path) > 1 {
		u.Path = strings.TrimSuffix(u.Path, "/")
		u.RawPath = strings.TrimSuffix(u.RawPath, "/")
	}
	return u.String()
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
