package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"boardly/internal/blobs"
	"boardly/internal/media"
)

// Station searches a user may start: 10 at once, then one every 2 seconds.
const (
	searchesPerHour = 30 * 60
	searchBurst     = 10
	// searchLogoCacheControl lets the browser keep a result's logo while the
	// dialog is used, but no shared cache.
	searchLogoCacheControl = "private, max-age=3600"
)

// searchResult is a directory station with a link to its logo through Boardly.
type searchResult struct {
	media.Station
	// Logo is the logo's link through Boardly, or nil.
	Logo *string `json:"logo"`
}

// logoSignature signs a logo URL for one user, so that the logo endpoint only
// fetches URLs that a search returned to that user.
func (a *api) logoSignature(userID, logoURL string) string {
	mac := hmac.New(sha256.New, a.logoKey)
	mac.Write([]byte(userID + "\n" + logoURL))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// searchStations searches radio-browser.info for the user, so that the
// browser doesn't send the user's IP address to it.
func (a *api) searchStations(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if n := utf8.RuneCountInString(query); n < 2 || n > 100 {
		return invalid("invalid_query", "The search must have 2 to 100 characters")
	}
	if ok, retry := a.searchLimiter.Reserve(user.ID, time.Now()); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many searches, try again in a moment")
	}
	stations, err := a.directory.Search(r.Context(), query)
	if errors.Is(err, media.ErrDirectoryUnreachable) {
		return NewError(http.StatusBadGateway, "directory_unreachable", "The station directory isn't reachable")
	} else if err != nil {
		return err
	}
	out := make([]searchResult, len(stations))
	for i, s := range stations {
		out[i] = searchResult{Station: s}
		if s.LogoURL != nil {
			link := "/api/media/search/logo?" + url.Values{
				"u": {*s.LogoURL},
				"s": {a.logoSignature(user.ID, *s.LogoURL)},
			}.Encode()
			out[i].Logo = &link
		}
	}
	WriteJSON(w, http.StatusOK, out)
	return nil
}

// searchLogo serves the logo of a search result through Boardly, so that the
// browser doesn't contact the logo's host.
func (a *api) searchLogo(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	notFound := NewError(http.StatusNotFound, "not_found", "Not found")
	q := r.URL.Query()
	logoURL, sig := q.Get("u"), q.Get("s")
	if logoURL == "" || !hmac.Equal([]byte(sig), []byte(a.logoSignature(user.ID, logoURL))) {
		return notFound
	}
	data, contentType, err := a.favicons.FetchImage(r.Context(), logoURL)
	if err != nil {
		return notFound
	}
	h := w.Header()
	blobs.SetHeaders(h, contentType)
	h.Set("Content-Length", strconv.Itoa(len(data)))
	h.Set("Cache-Control", searchLogoCacheControl)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}
