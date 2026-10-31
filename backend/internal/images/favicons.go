package images

import (
	"context"
	"database/sql"
	"encoding/json"
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

	"boardly/internal/db"
	"boardly/internal/favourites"
	"boardly/internal/jobs"
)

// Limits of the favicon fetcher.
const (
	FetchTimeout  = 10 * time.Second
	MaxPageSize   = 512 << 10
	MaxRedirects  = 5
	RetryAfter    = 24 * time.Hour
	workers       = 4
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

	// Jobs is the queue that favicon fetches run on.
	Jobs *jobs.Queue

	group singleflight.Group
}

// NewFavicons returns a fetcher whose connections are guarded unless
// allowPrivate is set.
func NewFavicons(store *Store, allowPrivate bool) *Favicons {
	client := &http.Client{
		Transport: GuardedTransport(allowPrivate),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// via holds the requests so far, so len(via) redirects were followed.
			if len(via) > MaxRedirects {
				return fmt.Errorf("more than %d redirects", MaxRedirects)
			}
			return nil
		},
	}
	return &Favicons{Images: store, Client: client, Now: time.Now}
}

// Origin returns scheme://host[:port] of an http or https URL, lower-cased
// and without the default port (see favourites.Origin), or ErrInvalidURL.
func Origin(rawURL string) (string, error) {
	o, err := favourites.Origin(rawURL)
	if err != nil {
		return "", ErrInvalidURL
	}
	return o, nil
}

// cached is an origin's row in favicons.
type cached struct {
	imageID   sql.NullString
	fetchedAt sql.NullString
	failedAt  sql.NullString
}

// Get returns the image ID of the favicon of rawURL's origin, from the cache
// or by fetching it. A stored favicon is kept until Refresh replaces it.
// After a failed fetch, the origin isn't fetched again for RetryAfter. It
// fails with ErrInvalidURL, ErrBlocked or ErrNoFavicon.
func (f *Favicons) Get(ctx context.Context, rawURL string) (string, error) {
	origin, err := Origin(rawURL)
	if err != nil {
		return "", err
	}
	c, err := f.load(ctx, origin)
	if err != nil {
		return "", err
	}
	if c.imageID.Valid {
		return c.imageID.String, nil
	}
	if f.within(c.failedAt, f.Now(), RetryAfter) {
		return "", ErrNoFavicon
	}
	return f.fetchShared(ctx, origin, rawURL, c)
}

// Refresh fetches the favicon of rawURL's origin again right away, also
// during the back-off after a failure, and returns its image ID. A refresh
// keeps the ID of a stored favicon. When it fails, the stored favicon stays
// and the error is ErrNoFavicon or ErrBlocked.
func (f *Favicons) Refresh(ctx context.Context, rawURL string) (string, error) {
	origin, err := Origin(rawURL)
	if err != nil {
		return "", err
	}
	c, err := f.load(ctx, origin)
	if err != nil {
		return "", err
	}
	return f.fetchShared(ctx, origin, rawURL, c)
}

// fetchShared runs one refresh per origin at a time; concurrent callers wait
// for it and share its result.
func (f *Favicons) fetchShared(ctx context.Context, origin, rawURL string, c cached) (string, error) {
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
	images, err := f.LookupImages(ctx, origins)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(images))
	for origin, img := range images {
		out[origin] = img.ID
	}
	return out, nil
}

// FaviconImage is a known favicon: its image ID and content type.
type FaviconImage struct {
	ID          string
	ContentType string
}

// LookupImages is Lookup with each favicon's content type, for clients that
// can only show some image types.
func (f *Favicons) LookupImages(ctx context.Context, origins []string) (map[string]FaviconImage, error) {
	out := map[string]FaviconImage{}
	if len(origins) == 0 {
		return out, nil
	}
	args := make([]any, len(origins))
	for i, o := range origins {
		args[i] = o
	}
	rows, err := f.Images.DB.QueryContext(ctx,
		`SELECT f.origin, i.id, i.content_type FROM favicons f JOIN images i ON i.id = f.image_id
		 WHERE f.origin IN (?`+strings.Repeat(",?", len(origins)-1)+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var origin string
		var img FaviconImage
		if err := rows.Scan(&origin, &img.ID, &img.ContentType); err != nil {
			return nil, err
		}
		out[origin] = img
	}
	return out, rows.Err()
}

// Queue asks for rawURL's favicon in the background, as a favicon job keyed
// by its origin. Links queue theirs with the change that creates them; this
// is for other URLs, such as plain link widgets.
func (f *Favicons) Queue(rawURL string) {
	origin, err := Origin(rawURL)
	if err != nil || f.Jobs == nil {
		return
	}
	ctx := context.Background()
	if err := f.Jobs.Enqueue(ctx, f.Images.DB, jobs.KindFavicon, origin, jobs.FaviconPayload{URL: rawURL}); err != nil {
		slog.ErrorContext(ctx, "queueing a favicon failed", "error", err)
	}
}

// Run starts the favicon workers on the job queue until ctx is done.
func (f *Favicons) Run(ctx context.Context) {
	f.Jobs.Run(ctx, jobs.KindFavicon, workers, f.handle)
}

// handle is the favicon job. A site without a favicon or on a blocked
// address is not an error.
func (f *Favicons) handle(ctx context.Context, job jobs.Job) error {
	var p jobs.FaviconPayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		return err
	}
	if _, err := f.Get(ctx, p.URL); err != nil && !errors.Is(err, ErrNoFavicon) && !errors.Is(err, ErrBlocked) {
		return err
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
// so references stay valid. A failed refresh leaves the stored image as it is
// and returns ErrBlocked or ErrNoFavicon.
func (f *Favicons) refresh(ctx context.Context, origin, pageURL string, c cached) (string, error) {
	now := db.FormatTime(f.Now())
	data, err := f.fetch(ctx, origin, pageURL)
	if errors.Is(err, ErrBlocked) {
		// Not a failure of the site: nothing is recorded, so the answer
		// stays "blocked" instead of turning into a back-off.
		return "", ErrBlocked
	}
	if err != nil {
		if _, dbErr := f.Images.DB.ExecContext(ctx,
			`INSERT INTO favicons (origin, failed_at) VALUES (?, ?)
			 ON CONFLICT (origin) DO UPDATE SET failed_at = excluded.failed_at`, origin, now); dbErr != nil {
			return "", dbErr
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

// FetchLogo loads the image at rawURL with the favicon rules (types, size,
// time limit, guarded connections) and stores it as an upload of the user,
// such as a radio station's logo. It returns the new image's ID.
func (f *Favicons) FetchLogo(ctx context.Context, userID, rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrInvalidURL
	}
	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	data, err := f.download(ctx, u.String())
	if err != nil {
		return "", err
	}
	img, err := f.Images.Save(ctx, SourceUpload, userID, data)
	if err != nil {
		return "", err
	}
	return img.ID, nil
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
	req.Header.Set("User-Agent", "Boardly favicon fetcher")
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
