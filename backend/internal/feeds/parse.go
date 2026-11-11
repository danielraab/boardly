package feeds

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
	xhtml "golang.org/x/net/html"
)

// Limits of what is stored of a feed.
const (
	maxTitleLength      = 500
	maxSummaryLength    = 1000
	untitledTitleLength = 100
)

// strict removes all markup, and the content of scripts and styles.
var strict = bluemonday.StrictPolicy()

// Parsed is a feed document, reduced to what is stored.
type Parsed struct {
	Title   string
	SiteURL *string
	Items   []ParsedItem
}

// ParsedItem is one entry of a feed document.
type ParsedItem struct {
	Identity string
	Title    string
	Link     *string
	Summary  *string
	Time     time.Time
	Audio    *Audio
}

// Audio is the file of a podcast episode.
type Audio struct {
	URL             string  `json:"url"`
	Type            *string `json:"type"`
	DurationSeconds *int    `json:"durationSeconds"`
}

// Parse reads a feed document and keeps the newest keep entries. Times are
// never later than now, and entries without a time count as seen now, in
// the order of the document.
func Parse(body []byte, keep int, now time.Time) (*Parsed, error) {
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotAFeed, err)
	}
	out := &Parsed{Title: cut(plainText(feed.Title), maxTitleLength)}
	if link := httpLink(feed.Link); link != nil {
		out.SiteURL = link
	}
	seen := make(map[string]bool, len(feed.Items))
	for i, entry := range feed.Items {
		if entry == nil {
			continue
		}
		item, ok := parseItem(entry, now.Add(-time.Duration(i)*time.Millisecond))
		if !ok || seen[item.Identity] {
			continue
		}
		seen[item.Identity] = true
		out.Items = append(out.Items, item)
	}
	slices.SortStableFunc(out.Items, func(a, b ParsedItem) int { return cmp.Compare(b.Time.UnixNano(), a.Time.UnixNano()) })
	if len(out.Items) > keep {
		out.Items = out.Items[:keep]
	}
	return out, nil
}

func parseItem(entry *gofeed.Item, seen time.Time) (ParsedItem, bool) {
	title := plainText(entry.Title)
	summary := plainText(entry.Description)
	if summary == "" {
		summary = plainText(entry.Content)
	}
	if title == "" && summary == "" {
		return ParsedItem{}, false
	}
	if title == "" {
		title = cut(summary, untitledTitleLength)
	}
	item := ParsedItem{Title: cut(title, maxTitleLength), Link: httpLink(entry.Link), Time: seen, Audio: parseAudio(entry)}
	if summary != "" {
		s := cut(summary, maxSummaryLength)
		item.Summary = &s
	}
	var when *time.Time
	if entry.PublishedParsed != nil {
		when = entry.PublishedParsed
	} else if entry.UpdatedParsed != nil {
		when = entry.UpdatedParsed
	}
	if when != nil && when.Before(seen) {
		item.Time = *when
	}
	item.Identity = identity(entry, title)
	return item, true
}

// identity is what makes an entry the same one on the next fetch: its GUID,
// else its link, else a hash of its title and date.
func identity(entry *gofeed.Item, title string) string {
	if id := strings.TrimSpace(entry.GUID); id != "" {
		return id
	}
	if link := strings.TrimSpace(entry.Link); link != "" {
		return link
	}
	sum := sha256.Sum256([]byte(title + "\x00" + entry.Published + "\x00" + entry.Updated))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// plainText removes markup, decodes entities and folds whitespace.
func plainText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(strict.Sanitize(s))), " ")
}

func cut(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	return string([]rune(s)[:limit])
}

// httpLink returns s when it is an absolute http or https URL.
func httpLink(s string) *string {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil
	}
	return &s
}

var audioExtensions = map[string]bool{
	".mp3": true, ".m4a": true, ".m4b": true, ".aac": true, ".ogg": true,
	".oga": true, ".opus": true, ".wav": true, ".flac": true,
}

// genericType is an enclosure type that says nothing about the content.
func genericType(t string) bool {
	return t == "" || t == "application/octet-stream" || t == "binary/octet-stream" || t == "application/x-download"
}

func parseAudio(entry *gofeed.Item) *Audio {
	for _, enc := range entry.Enclosures {
		if enc == nil {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(enc.URL))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}
		mediaType := strings.ToLower(strings.TrimSpace(strings.SplitN(enc.Type, ";", 2)[0]))
		if !strings.HasPrefix(mediaType, "audio/") && !(genericType(mediaType) && audioExtensions[strings.ToLower(path.Ext(u.Path))]) {
			continue
		}
		u.Scheme = "https"
		audio := &Audio{URL: u.String()}
		if mediaType != "" {
			audio.Type = &mediaType
		}
		if entry.ITunesExt != nil {
			audio.DurationSeconds = parseDuration(entry.ITunesExt.Duration)
		}
		return audio
	}
	return nil
}

// parseDuration reads an iTunes duration: seconds, MM:SS or HH:MM:SS.
func parseDuration(s string) *int {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) == 0 || len(parts) > 3 {
		return nil
	}
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil
		}
		total = total*60 + n
	}
	if total == 0 {
		return nil
	}
	return &total
}

// isHTML tells whether a response is a web page rather than a feed.
func isHTML(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml") {
		return true
	}
	head := bytes.ToLower(bytes.TrimSpace(body))
	if len(head) > 256 {
		head = head[:256]
	}
	return bytes.Contains(head, []byte("<!doctype html")) || bytes.Contains(head, []byte("<html"))
}

// discover returns the first feed a page advertises through
// <link rel="alternate">, resolved against base.
func discover(body []byte, base *url.URL) (string, bool) {
	tokens := xhtml.NewTokenizer(bytes.NewReader(body))
	for {
		switch tokens.Next() {
		case xhtml.ErrorToken:
			return "", false
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			name, hasAttr := tokens.TagName()
			if string(name) == "body" {
				return "", false
			}
			if string(name) != "link" || !hasAttr {
				continue
			}
			attrs := map[string]string{}
			for {
				key, value, more := tokens.TagAttr()
				attrs[string(key)] = string(value)
				if !more {
					break
				}
			}
			if !strings.Contains(" "+strings.ToLower(attrs["rel"])+" ", " alternate ") || !feedType(attrs["type"]) {
				continue
			}
			ref, err := url.Parse(strings.TrimSpace(attrs["href"]))
			if err != nil || attrs["href"] == "" {
				continue
			}
			target := base.ResolveReference(ref)
			if target.Scheme == "http" || target.Scheme == "https" {
				return target.String(), true
			}
		}
	}
}

func feedType(t string) bool {
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(t, ";", 2)[0])) {
	case "application/rss+xml", "application/atom+xml", "application/feed+json":
		return true
	}
	return false
}
