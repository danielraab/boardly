package images

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/sync/singleflight"

	"famodash/internal/db"
	"famodash/internal/favourites"
)

// Limits of the favicon fetcher.
const (
	FetchTimeout  = 10 * time.Second
	MaxPageSize   = 512 << 10
	MaxRedirects  = 5
	RefreshAfter  = 30 * 24 * time.Hour
	RetryAfter    = 24 * time.Hour
	workers       = 4
	queueSize     = 256
	maxCandidates = 4
)

// Errors returned by Favicons.Get.
var (
	ErrInvalidURL = errors.New("not an absolute http or https URL")
	ErrNoFavicon  = errors.New("no favicon found")
)

// Favicons fetches favicons and keeps one shared copy per origin.
type Favicons struct {
	Images *Store
	Client *http.Client
	Now    func() time.Time

	group singleflight.Group
	queue chan string
}

// NewFavicons returns a fetcher whose connections are guarded unless
// allowPrivate is set.
func NewFavicons(store *Store, allowPrivate bool) *Favicons {
	transport := &http.Transport{
		Proxy:                 nil, // a proxy would hide the real target from the guard
		DialContext:           guardedDialer(allowPrivate).DialContext,
		TLSHandshakeTimeout:   FetchTimeout,
		ResponseHeaderTimeout: FetchTimeout,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// via holds the requests so far, so len(via) redirects were followed.
			if len(via) > MaxRedirects {
				return fmt.Errorf("more than %d redirects", MaxRedirects)
			}
			return nil
		},
	}
	return &Favicons{Images: store, Client: client, Now: time.Now, queue: make(chan string, queueSize)}
}

// Origin returns scheme://host[:port] of an http or https URL, lower-cased
// and without the default port.
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

// cached is an origin's row in favicons.
type cached struct {
	imageID   sql.NullString
	fetchedAt sql.NullString
	failedAt  sql.NullString
}

// Get returns the image ID of the favicon of rawURL's origin, from the cache
// or by fetching it. It fails with ErrInvalidURL, ErrBlocked or ErrNoFavicon.
func (f *Favicons) Get(ctx context.Context, rawURL string) (string, error) {
	origin, err := Origin(rawURL)
	if err != nil {
		return "", err
	}
	c, err := f.load(ctx, origin)
	if err != nil {
		return "", err
	}
	now := f.Now()
	fresh := c.imageID.Valid && f.within(c.fetchedAt, now, RefreshAfter)
	backingOff := f.within(c.failedAt, now, RetryAfter)
	switch {
	case fresh:
		return c.imageID.String, nil
	case backingOff && c.imageID.Valid:
		return c.imageID.String, nil
	case backingOff:
		return "", ErrNoFavicon
	}

	// The fetch must outlive a cancelled request, because others may wait on it.
	v, err, _ := f.group.Do(origin, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), FetchTimeout)
		defer cancel()
		return f.refresh(fetchCtx, origin, rawURL, c)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

// SetIcons sets IconImageID on every link in the trees whose origin has a
// known favicon.
func (f *Favicons) SetIcons(ctx context.Context, roots ...*favourites.Node) error {
	byOrigin := map[string][]*favourites.Node{}
	var walk func(n *favourites.Node)
	walk = func(n *favourites.Node) {
		if n.Kind == favourites.KindLink && n.URL != nil {
			if origin, err := Origin(*n.URL); err == nil {
				byOrigin[origin] = append(byOrigin[origin], n)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	for _, n := range roots {
		walk(n)
	}
	origins := make([]string, 0, len(byOrigin))
	for o := range byOrigin {
		origins = append(origins, o)
	}
	ids, err := f.Lookup(ctx, origins)
	if err != nil {
		return err
	}
	for origin, id := range ids {
		for _, n := range byOrigin[origin] {
			n.IconImageID = &id
		}
	}
	return nil
}

// Lookup returns the known favicon image IDs of origins, without fetching.
func (f *Favicons) Lookup(ctx context.Context, origins []string) (map[string]string, error) {
	out := map[string]string{}
	if len(origins) == 0 {
		return out, nil
	}
	args := make([]any, len(origins))
	for i, o := range origins {
		args[i] = o
	}
	rows, err := f.Images.DB.QueryContext(ctx,
		`SELECT origin, image_id FROM favicons WHERE image_id IS NOT NULL AND origin IN (?`+
			strings.Repeat(",?", len(origins)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var origin, id string
		if err := rows.Scan(&origin, &id); err != nil {
			return nil, err
		}
		out[origin] = id
	}
	return out, rows.Err()
}

// Queue asks for rawURL's favicon in the background. It never blocks; when
// the queue is full the request is dropped and retried on a later Get.
func (f *Favicons) Queue(rawURL string) {
	select {
	case f.queue <- rawURL:
	default:
	}
}

// Run works through the queue until ctx is done.
func (f *Favicons) Run(ctx context.Context) {
	for range workers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case u := <-f.queue:
					if _, err := f.Get(ctx, u); err != nil && !errors.Is(err, ErrNoFavicon) &&
						!errors.Is(err, ErrBlocked) && ctx.Err() == nil {
						slog.WarnContext(ctx, "fetching favicon failed", "error", err)
					}
				}
			}
		}()
	}
}

// Backfill queues one link URL for every origin without a favicons row.
func (f *Favicons) Backfill(ctx context.Context) error {
	rows, err := f.Images.DB.QueryContext(ctx, `SELECT url FROM favourite_nodes WHERE kind = 'link'`)
	if err != nil {
		return err
	}
	byOrigin := map[string]string{}
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			rows.Close()
			return err
		}
		if o, err := Origin(u); err == nil {
			if _, ok := byOrigin[o]; !ok {
				byOrigin[o] = u
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	known, err := f.Images.DB.QueryContext(ctx, `SELECT origin FROM favicons`)
	if err != nil {
		return err
	}
	for known.Next() {
		var o string
		if err := known.Scan(&o); err != nil {
			known.Close()
			return err
		}
		delete(byOrigin, o)
	}
	known.Close()
	if err := known.Err(); err != nil {
		return err
	}
	for _, u := range byOrigin {
		f.Queue(u)
	}
	return nil
}

func (f *Favicons) load(ctx context.Context, origin string) (cached, error) {
	var c cached
	err := f.Images.DB.QueryRowContext(ctx,
		`SELECT image_id, fetched_at, failed_at FROM favicons WHERE origin = ?`, origin,
	).Scan(&c.imageID, &c.fetchedAt, &c.failedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return cached{}, nil
	}
	return c, err
}

// within reports whether the stored time ts is less than d before now.
func (f *Favicons) within(ts sql.NullString, now time.Time, d time.Duration) bool {
	if !ts.Valid {
		return false
	}
	t, err := db.ParseTime(ts.String)
	return err == nil && now.Sub(t) < d
}

// refresh fetches the favicon and stores it. A refresh keeps the image ID,
// so references stay valid. A failed refresh keeps serving the old image.
func (f *Favicons) refresh(ctx context.Context, origin, pageURL string, c cached) (string, error) {
	now := db.FormatTime(f.Now())
	data, err := f.fetch(ctx, origin, pageURL)
	if errors.Is(err, ErrBlocked) {
		// Not a failure of the site: nothing is recorded, so the answer
		// stays "blocked" instead of turning into a back-off.
		if c.imageID.Valid {
			return c.imageID.String, nil
		}
		return "", ErrBlocked
	}
	if err != nil {
		if _, dbErr := f.Images.DB.ExecContext(ctx,
			`INSERT INTO favicons (origin, failed_at) VALUES (?, ?)
			 ON CONFLICT (origin) DO UPDATE SET failed_at = excluded.failed_at`, origin, now); dbErr != nil {
			return "", dbErr
		}
		if c.imageID.Valid {
			return c.imageID.String, nil
		}
		return "", fmt.Errorf("%w: %v", ErrNoFavicon, err)
	}

	id := c.imageID.String
	if c.imageID.Valid {
		err = f.Images.Replace(ctx, id, data)
	}
	if !c.imageID.Valid || errors.Is(err, ErrNotFound) {
		var img Image
		img, err = f.Images.Save(ctx, SourceFavicon, "", data)
		id = img.ID
	}
	if err != nil {
		return "", err
	}
	if _, err := f.Images.DB.ExecContext(ctx,
		`INSERT INTO favicons (origin, image_id, fetched_at) VALUES (?, ?, ?)
		 ON CONFLICT (origin) DO UPDATE SET image_id = excluded.image_id,
		   fetched_at = excluded.fetched_at, failed_at = NULL`, origin, id, now); err != nil {
		return "", err
	}
	return id, nil
}

// fetch loads the page, tries its declared icons in order of preference and
// then /favicon.ico of the origin. It returns the first valid image.
func (f *Favicons) fetch(ctx context.Context, origin, pageURL string) ([]byte, error) {
	var candidates []string
	icons, err := f.pageIcons(ctx, pageURL)
	if errors.Is(err, ErrBlocked) {
		return nil, err
	}
	candidates = append(candidates, icons...)
	candidates = append(candidates, origin+"/favicon.ico")

	var last error = ErrNoFavicon
	for _, c := range candidates {
		data, err := f.download(ctx, c)
		if err == nil {
			return data, nil
		}
		if errors.Is(err, ErrBlocked) {
			return nil, err
		}
		last = err
	}
	return nil, last
}

// pageIcons returns the absolute URLs of the page's icons, best first.
func (f *Favicons) pageIcons(ctx context.Context, pageURL string) ([]string, error) {
	resp, err := f.get(ctx, pageURL, "text/html")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Content-Type"), "html") {
		return nil, nil
	}
	icons := ParseIcons(io.LimitReader(resp.Body, MaxPageSize), resp.Request.URL)
	if len(icons) > maxCandidates {
		icons = icons[:maxCandidates]
	}
	return icons, nil
}

// download loads one icon and checks its type and size.
func (f *Favicons) download(ctx context.Context, iconURL string) ([]byte, error) {
	resp, err := f.get(ctx, iconURL, "image/*")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: status %d", iconURL, resp.StatusCode)
	}
	data, err := ReadLimited(resp.Body)
	if err != nil {
		return nil, err
	}
	if _, err := Sniff(data); err != nil {
		return nil, err
	}
	return data, nil
}

func (f *Favicons) get(ctx context.Context, rawURL, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "Famodash favicon fetcher")
	return f.Client.Do(req)
}

// icon is a <link> to an icon with its preference.
type icon struct {
	href string
	rank int // by rel: icon 3, shortcut icon 2, apple-touch-icon 1
	size int // largest declared size up to 256, 0 if unknown
}

// ParseIcons returns the absolute http(s) URLs of the icons a page declares
// with <link rel="icon">, "shortcut icon" or "apple-touch-icon", best first:
// by rel, then by size. SVG icons and data: URLs are left out.
func ParseIcons(r io.Reader, base *url.URL) []string {
	var icons []icon
	z := html.NewTokenizer(r)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}
		name, hasAttr := z.TagName()
		tag := string(name)
		if tag == "body" {
			break
		}
		if !hasAttr || (tag != "link" && tag != "base") {
			continue
		}
		attrs := map[string]string{}
		for {
			k, v, more := z.TagAttr()
			attrs[string(k)] = string(v)
			if !more {
				break
			}
		}
		if tag == "base" {
			if b, err := base.Parse(attrs["href"]); err == nil && attrs["href"] != "" {
				base = b
			}
			continue
		}
		rank := relRank(attrs["rel"])
		href := strings.TrimSpace(attrs["href"])
		if rank == 0 || href == "" || strings.Contains(attrs["type"], "svg") {
			continue
		}
		u, err := base.Parse(href)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") ||
			strings.HasSuffix(strings.ToLower(u.Path), ".svg") {
			continue
		}
		icons = append(icons, icon{href: u.String(), rank: rank, size: largestSize(attrs["sizes"])})
	}
	sort.SliceStable(icons, func(i, j int) bool {
		if icons[i].rank != icons[j].rank {
			return icons[i].rank > icons[j].rank
		}
		return icons[i].size > icons[j].size
	})
	out := make([]string, len(icons))
	for i, ic := range icons {
		out[i] = ic.href
	}
	return out
}

func relRank(rel string) int {
	tokens := strings.Fields(strings.ToLower(rel))
	has := func(t string) bool {
		for _, x := range tokens {
			if x == t {
				return true
			}
		}
		return false
	}
	switch {
	case has("icon") && has("shortcut"):
		return 2
	case has("icon"):
		return 3
	case has("apple-touch-icon") || has("apple-touch-icon-precomposed"):
		return 1
	}
	return 0
}

// largestSize returns the largest size in a sizes attribute ("16x16 32x32")
// that is at most 256, or 0.
func largestSize(sizes string) int {
	best := 0
	for _, s := range strings.Fields(strings.ToLower(sizes)) {
		w, _, ok := strings.Cut(s, "x")
		n, err := strconv.Atoi(w)
		if ok && err == nil && n <= 256 && n > best {
			best = n
		}
	}
	return best
}
