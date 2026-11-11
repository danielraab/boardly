package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"boardly/internal/feeds"
	"boardly/internal/notes"
)

// Limits of adding feeds, which fetches URLs on the user's behalf.
const (
	feedAddsPerHour = 120
	feedAddBurst    = 20
	maxFeedBytes    = 16 << 10
)

// feedError maps store errors to API errors. Other errors pass through.
func feedError(err error) error {
	switch {
	case errors.Is(err, feeds.ErrNotFound):
		return NewError(http.StatusNotFound, "not_found", "Not found")
	case errors.Is(err, feeds.ErrInvalidURL):
		return invalid("invalid_url", "The URL must be an absolute http or https URL")
	case errors.Is(err, feeds.ErrNotAFeed):
		return invalid("feed_not_found", "No feed was found at this address")
	case errors.Is(err, feeds.ErrUnreachable):
		return invalid("feed_unreachable", "The address can't be reached")
	case errors.Is(err, feeds.ErrAlreadySubscribed):
		return NewError(http.StatusConflict, "already_subscribed", "You already have this feed")
	case errors.Is(err, feeds.ErrInvalidTitle):
		return invalid("invalid_title", "The title can have at most 200 characters")
	case errors.Is(err, feeds.ErrInvalidIcon):
		return invalid("invalid_icon", "The icon must be a known symbol or an image you uploaded")
	case errors.Is(err, feeds.ErrInvalidOrder):
		return invalid("invalid_order", "The order must list each of your feeds once")
	case errors.Is(err, feeds.ErrInvalidQuery):
		return invalid("invalid_request", "The view, limit or cursor is invalid")
	}
	return err
}

func (a *api) listFeeds(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	list, err := a.feeds.List(r.Context(), user.ID)
	if err != nil {
		return err
	}
	WriteJSON(w, http.StatusOK, list)
	return nil
}

func (a *api) createFeed(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	body, err := decodeFields(r, maxFeedBytes, invalid("invalid_request", "The request is too large"))
	if err != nil {
		return err
	}
	bad := invalid("invalid_request", "A field has the wrong type")
	var feedURL *string
	if _, err := field(body, "url", &feedURL, bad); err != nil {
		return err
	}
	if err := unknownFields(body, "invalid_request"); err != nil {
		return err
	}
	if feedURL == nil {
		return invalid("invalid_url", "The URL must be an absolute http or https URL")
	}
	if ok, retry := a.feedLimiter.Reserve(user.ID, time.Now()); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(retry)))
		return NewError(http.StatusTooManyRequests, "rate_limited", "Too many feeds added, try again in a moment")
	}
	sub, err := a.feeds.Subscribe(r.Context(), user.ID, *feedURL)
	if err != nil {
		return feedError(err)
	}
	WriteJSON(w, http.StatusCreated, sub)
	return nil
}

func (a *api) updateFeed(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	body, err := decodeFields(r, maxFeedBytes, invalid("invalid_request", "The request is too large"))
	if err != nil {
		return err
	}
	bad := invalid("invalid_request", "A field has the wrong type")
	var p feeds.Patch
	var title *string
	if ok, err := field(body, "customTitle", &title, bad); err != nil {
		return err
	} else if ok {
		p.CustomTitle = &title
	}
	var icon *notes.Icon
	if ok, err := field(body, "icon", &icon, bad); err != nil {
		return err
	} else if ok {
		p.Icon = &icon
	}
	if err := unknownFields(body, "invalid_request"); err != nil {
		return err
	}
	sub, err := a.feeds.Update(r.Context(), user.ID, r.PathValue("id"), p)
	if err != nil {
		return feedError(err)
	}
	WriteJSON(w, http.StatusOK, sub)
	return nil
}

func (a *api) reorderFeeds(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	var ids []string
	if err := decodeJSON(r, &ids); err != nil {
		return err
	}
	if err := a.feeds.Reorder(r.Context(), user.ID, ids); err != nil {
		return feedError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (a *api) deleteFeed(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	if err := a.feeds.Unsubscribe(r.Context(), user.ID, r.PathValue("id")); err != nil {
		return feedError(err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// maxFeedSearchLength is the longest search for feed items, in characters.
const maxFeedSearchLength = 200

func (a *api) listFeedItems(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	query := r.URL.Query()
	q := feeds.ItemQuery{View: query.Get("view"), Limit: 50, Cursor: query.Get("cursor")}
	if q.View == "" {
		q.View = feeds.ViewLatest
	}
	if v := query.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return feedError(feeds.ErrInvalidQuery)
		}
		q.Limit = n
	}
	if id := query.Get("feedId"); id != "" {
		q.SubscriptionIDs = []string{id}
	}
	q.Search = query.Get("q")
	if utf8.RuneCountInString(q.Search) > maxFeedSearchLength {
		return feedError(feeds.ErrInvalidQuery)
	}
	items, next, err := a.feeds.Items(r.Context(), user.ID, q)
	if err != nil {
		return feedError(err)
	}
	var cursor *string
	if next != "" {
		cursor = &next
	}
	WriteJSON(w, http.StatusOK, struct {
		Items      []feeds.Item `json:"items"`
		NextCursor *string      `json:"nextCursor"`
	}{items, cursor})
	return nil
}

func (a *api) updateFeedItem(w http.ResponseWriter, r *http.Request) error {
	user, _ := CurrentUser(r.Context())
	body, err := decodeFields(r, maxFeedBytes, invalid("invalid_request", "The request is too large"))
	if err != nil {
		return err
	}
	bad := invalid("invalid_request", "A field has the wrong type")
	var readLater, liked *bool
	for name, dst := range map[string]**bool{"readLater": &readLater, "liked": &liked} {
		if ok, err := field(body, name, dst, bad); err != nil {
			return err
		} else if ok && *dst == nil {
			return bad
		}
	}
	if err := unknownFields(body, "invalid_request"); err != nil {
		return err
	}
	item, err := a.feeds.SetFlags(r.Context(), user.ID, r.PathValue("id"), readLater, liked)
	if err != nil {
		return feedError(err)
	}
	WriteJSON(w, http.StatusOK, item)
	return nil
}
