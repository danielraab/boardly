package offline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"boardly/internal/images"
	"boardly/internal/jobs"
)

// ScreenshotWidth and ScreenshotHeight are the browser's viewport, and so
// the size of a screenshot.
const (
	ScreenshotWidth  = 1280
	ScreenshotHeight = 800
)

// ScreenshotTimeout is the longest a screenshot may take.
const ScreenshotTimeout = 30 * time.Second

// Screenshotter takes a screenshot of the viewport of a page.
type Screenshotter interface {
	Screenshot(ctx context.Context, pageURL string) ([]byte, error)
}

// PreviewFiller makes an image the preview of a link that has none.
type PreviewFiller interface {
	Fill(ctx context.Context, userID, nodeID, rawURL string, data []byte) error
}

// Capturer runs the offline_capture jobs.
type Capturer struct {
	Store *Store
	Jobs  *jobs.Queue
	// Client is the guarded client that loads pages.
	Client *http.Client
	// MaxBytes is the largest page read for an article.
	MaxBytes int64
	// Browser takes screenshots; nil takes none.
	Browser Screenshotter
	// Converter turns a screenshot into the stored JPEG.
	Converter images.Converter
	// Previews makes screenshots previews of pages without a preview image;
	// nil never does.
	Previews PreviewFiller
}

// Run starts the capture workers until ctx is done. Two workers capture, so
// articles don't wait for slow screenshots; the browser takes one at a time.
func (c *Capturer) Run(ctx context.Context) {
	c.Jobs.Run(ctx, jobs.KindOfflineCapture, 2, c.capture)
}

// capture is the offline_capture job.
func (c *Capturer) capture(ctx context.Context, job jobs.Job) error {
	var pl jobs.OfflineCapturePayload
	if err := json.Unmarshal(job.Payload, &pl); err != nil {
		return err
	}
	// Skip the work when the link changed or is gone.
	var current bool
	if err := c.Store.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM offline_copies WHERE node_id = ? AND url = ?)`, pl.NodeID, pl.URL,
	).Scan(&current); err != nil || !current {
		return err
	}

	pg := c.loadPage(ctx, pl.URL)
	var r Result
	if pg.reason == "" {
		if art, ok := ExtractArticle(pg.data, pg.url); ok {
			r.Article, r.Title = art.HTML, art.Title
		} else {
			pg.reason = ReasonNoContent
		}
	}
	if r.Title == "" {
		r.Title = pg.title
	}
	var shot []byte
	if c.Browser != nil && pg.reason != ReasonBlockedAddress {
		shot = c.screenshot(ctx, pl.URL)
		r.Screenshot = shot
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	r.Reason = pg.reason
	stored, err := c.Store.Record(ctx, pl.NodeID, pl.URL, r)
	if err != nil || !stored {
		return err
	}
	if pl.SetPreview && shot != nil && !pg.hasImage && c.Previews != nil {
		if err := c.Previews.Fill(ctx, pl.UserID, pl.NodeID, pl.URL, shot); err != nil &&
			!errors.Is(err, images.ErrTooLarge) && !errors.Is(err, images.ErrUnsupported) {
			return err
		}
	}
	return nil
}

// loaded is a page loaded for an article.
type loaded struct {
	data []byte
	// url is the page's URL after redirects.
	url *url.URL
	// title is the page's <title>, for a copy without an article.
	title string
	// hasImage says whether the page declares og:image or twitter:image.
	hasImage bool
	// reason is set when the page can't give an article.
	reason string
}

// loadPage loads pageURL with the guarded client and the page limits.
func (c *Capturer) loadPage(ctx context.Context, pageURL string) loaded {
	ctx, cancel := context.WithTimeout(ctx, images.FetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return loaded{reason: ReasonFetchFailed}
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Boardly offline copy)")
	resp, err := c.Client.Do(req)
	if err != nil {
		if errors.Is(err, images.ErrBlocked) {
			return loaded{reason: ReasonBlockedAddress}
		}
		return loaded{reason: ReasonFetchFailed}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return loaded{reason: ReasonFetchFailed}
	}
	if !isHTML(resp.Header.Get("Content-Type")) {
		return loaded{reason: ReasonNotHTML}
	}
	data, tooLarge, err := readAll(resp.Body, c.MaxBytes)
	if tooLarge {
		return loaded{reason: ReasonTooLarge}
	}
	if err != nil {
		if errors.Is(err, images.ErrBlocked) {
			return loaded{reason: ReasonBlockedAddress}
		}
		return loaded{reason: ReasonFetchFailed}
	}
	meta := images.ParseMeta(bytes.NewReader(data), resp.Request.URL)
	return loaded{data: data, url: resp.Request.URL, hasImage: meta.ImageURL != "", title: pageTitle(data)}
}

// screenshot takes and converts the screenshot of pageURL, or returns nil
// and logs why it couldn't.
func (c *Capturer) screenshot(ctx context.Context, pageURL string) []byte {
	ctx, cancel := context.WithTimeout(ctx, ScreenshotTimeout)
	defer cancel()
	raw, err := c.Browser.Screenshot(ctx, pageURL)
	if err == nil {
		var out []byte
		if out, err = c.Converter.Convert(raw); err == nil {
			return out
		}
	}
	if ctx.Err() == nil || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		slog.WarnContext(ctx, "taking a screenshot for an offline copy failed", "url", pageURL, "error", err)
	}
	return nil
}

func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml+xml")
}

// pageTitle is the page's <title>, collapsed and cut.
func pageTitle(data []byte) string {
	lower := bytes.ToLower(data)
	start := bytes.Index(lower, []byte("<title"))
	if start < 0 {
		return ""
	}
	open := bytes.IndexByte(lower[start:], '>')
	if open < 0 {
		return ""
	}
	from := start + open + 1
	end := bytes.Index(lower[from:], []byte("</title"))
	if end < 0 {
		return ""
	}
	return truncate(collapse(htmlUnescape(string(data[from:from+end]))), maxTitleLength)
}
