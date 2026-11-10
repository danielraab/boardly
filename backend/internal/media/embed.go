package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"boardly/internal/images"
)

// Errors of embed links.
var (
	ErrEmbedUnsupported = errors.New("no provider can embed this link")
	ErrEmbedUnreachable = errors.New("provider unreachable or item unknown")
)

// MaxEmbedInfoSize bounds a provider's oEmbed answer.
const MaxEmbedInfoSize = 64 << 10

// ShapeAudio is a player of fixed height.
const ShapeAudio = "audio"

// Embed is an item of a provider, as one of its links identifies it.
type Embed struct {
	Provider string
	Shape    string
	// URL is the canonical link, the one that is stored.
	URL string
	// PlayerURL is the address of the provider's player. It is only ever
	// built here, never taken from a pasted link or an import.
	PlayerURL string
	// InfoURL is where the provider tells the item's title and cover.
	InfoURL string
}

// provider recognizes the links of one embed provider. Adding a provider
// means adding one entry to providers.
type provider func(link string) (Embed, bool)

var providers = []provider{spotify}

// ParseEmbed returns the item that link identifies. It makes no network
// request.
func ParseEmbed(link string) (Embed, error) {
	link = strings.TrimSpace(link)
	for _, p := range providers {
		if e, ok := p(link); ok {
			return e, nil
		}
	}
	return Embed{}, ErrEmbedUnsupported
}

var (
	spotifyPath = regexp.MustCompile(`^/(?:intl-[a-z-]+/)?(playlist|album)/([0-9A-Za-z]{22})/?$`)
	spotifyURI  = regexp.MustCompile(`^spotify:(playlist|album):([0-9A-Za-z]{22})$`)
)

func spotify(link string) (Embed, bool) {
	m := spotifyURI.FindStringSubmatch(link)
	if m == nil {
		u, err := url.Parse(link)
		if err != nil || u.Scheme != "https" || u.Host != "open.spotify.com" || u.User != nil {
			return Embed{}, false
		}
		if m = spotifyPath.FindStringSubmatch(u.Path); m == nil {
			return Embed{}, false
		}
	}
	item := m[1] + "/" + m[2]
	canonical := "https://open.spotify.com/" + item
	return Embed{
		Provider:  "spotify",
		Shape:     ShapeAudio,
		URL:       canonical,
		PlayerURL: "https://open.spotify.com/embed/" + item,
		InfoURL:   "https://open.spotify.com/oembed?url=" + url.QueryEscape(canonical),
	}, true
}

// EmbedInfo is what a provider tells about an item.
type EmbedInfo struct {
	Title    string
	CoverURL string
}

// EmbedFetcher asks providers about items over guarded connections.
type EmbedFetcher struct {
	Client  *http.Client
	Timeout time.Duration
}

// NewEmbedFetcher returns a fetcher whose connections are guarded unless
// allowPrivate is set.
func NewEmbedFetcher(allowPrivate bool) *EmbedFetcher {
	return &EmbedFetcher{
		Client: &http.Client{
			Transport: images.GuardedTransport(allowPrivate),
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= MaxRedirects {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		Timeout: CheckTimeout,
	}
}

// Fetch reads the oEmbed answer at infoURL.
func (f *EmbedFetcher) Fetch(ctx context.Context, infoURL string) (EmbedInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, f.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, infoURL, nil)
	if err != nil {
		return EmbedInfo{}, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := f.Client.Do(req)
	if err != nil {
		return EmbedInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return EmbedInfo{}, fmt.Errorf("oEmbed answered %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxEmbedInfoSize+1))
	if err != nil {
		return EmbedInfo{}, err
	}
	if len(body) > MaxEmbedInfoSize {
		return EmbedInfo{}, errors.New("oEmbed answer too large")
	}
	var raw struct {
		Title        string `json:"title"`
		ThumbnailURL string `json:"thumbnail_url"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return EmbedInfo{}, err
	}
	return EmbedInfo{Title: raw.Title, CoverURL: raw.ThumbnailURL}, nil
}
