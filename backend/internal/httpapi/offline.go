package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"boardly/internal/offline"
)

// Offline copy refreshes a user may start: 30 at once, then one every 2 minutes.
const (
	offlineRefreshesPerHour = 30
	offlineRefreshBurst     = 30
)

// articleCSP lets an offline article show its images from the original
// site and open links in a new tab, and nothing else: no scripts, no access
// to Boardly's origin, and no framing by other sites.
const articleCSP = "sandbox allow-popups allow-popups-to-escape-sandbox; default-src 'none'; " +
	"img-src http: https: data:; style-src 'unsafe-inline'; frame-ancestors 'self'"

func offlineError(err error) error {
	if errors.Is(err, offline.ErrNotFound) {
		return NewError(http.StatusNotFound, "not_found", "Offline copy not found")
	}
	return err
}

// getOfflineCopy returns the state of a link's offline copy.
func (a *api) getOfflineCopy(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	c, err := a.offline.Get(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		return offlineError(err)
	}
	WriteJSON(w, http.StatusOK, c)
	return nil
}

// refreshOfflineCopy queues a new capture of a link's offline copy.
func (a *api) refreshOfflineCopy(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	// Check the link first, so a wrong ID doesn't use up the limit.
	if _, err := a.offline.Get(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return offlineError(err)
	}
	if ok, retry := a.offlineLimiter.Reserve(user.ID, time.Now()); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many offline copy refreshes, try again later")
	}
	c, err := a.offline.Refresh(r.Context(), a.favourites, user.ID, r.PathValue("id"))
	if err != nil {
		return offlineError(err)
	}
	WriteJSON(w, http.StatusAccepted, c)
	return nil
}

// getOfflineArticle serves the readable article of a link's offline copy.
func (a *api) getOfflineArticle(w http.ResponseWriter, r *http.Request) error {
	return a.serveOfflinePart(w, r, offline.PartArticle)
}

// getOfflineScreenshot serves the screenshot of a link's offline copy.
func (a *api) getOfflineScreenshot(w http.ResponseWriter, r *http.Request) error {
	return a.serveOfflinePart(w, r, offline.PartScreenshot)
}

func (a *api) serveOfflinePart(w http.ResponseWriter, r *http.Request, part offline.Part) error {
	user, _ := CurrentUser(r.Context())
	f, err := a.offline.Open(r.Context(), user.ID, r.PathValue("id"), part)
	if err != nil {
		return offlineError(err)
	}
	defer f.Close()
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "private, no-cache")
	if part == offline.PartArticle {
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("Content-Security-Policy", articleCSP)
		h.Set("Referrer-Policy", "no-referrer")
	} else {
		h.Set("Content-Type", "image/jpeg")
		h.Set("Content-Security-Policy", "sandbox")
	}
	if info, err := f.Stat(); err == nil {
		h.Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, f)
	}
	return nil
}
