// Package feeds keeps RSS, Atom and JSON feeds. A feed and its items exist
// once per server, however many users have them; subscriptions and the flags
// on items are what belongs to one user. Items are fetched by the server in
// the background, and showing them only reads the database.
package feeds

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"boardly/internal/db"
	"boardly/internal/images"
	"boardly/internal/notes"
)

// Errors of the store.
var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidURL        = errors.New("not an absolute http or https URL")
	ErrUnreachable       = errors.New("feed unreachable")
	ErrNotAFeed          = errors.New("not a feed")
	ErrAlreadySubscribed = errors.New("already subscribed")
	ErrInvalidTitle      = errors.New("invalid title")
	ErrInvalidIcon       = errors.New("invalid icon")
	ErrInvalidOrder      = errors.New("invalid order")
	ErrInvalidQuery      = errors.New("invalid query")
)

// Views of a user's items.
const (
	ViewLatest    = "latest"
	ViewReadLater = "read_later"
	ViewLiked     = "liked"
)

// MaxCustomTitle bounds a subscription's own title, in characters.
const MaxCustomTitle = 200

// FailingAfter is how many failures in a row make a feed count as failing.
const FailingAfter = 3

// Icons finds the shared favicons of feeds. It may be nil.
type Icons interface {
	Lookup(ctx context.Context, origins []string) (map[string]string, error)
	Queue(rawURL string)
}

// Subscription is a user's subscription as the API returns it.
type Subscription struct {
	ID            string      `json:"id"`
	FeedID        string      `json:"feedId"`
	URL           string      `json:"url"`
	Title         string      `json:"title"`
	CustomTitle   *string     `json:"customTitle"`
	SiteURL       *string     `json:"siteUrl"`
	Icon          *notes.Icon `json:"icon"`
	FaviconID     *string     `json:"faviconId"`
	Position      int         `json:"position"`
	LastFetchedAt *string     `json:"lastFetchedAt"`
	Failing       bool        `json:"failing"`
}

// ItemFeed is the feed an item is shown with.
type ItemFeed struct {
	Title          string      `json:"title"`
	SubscriptionID *string     `json:"subscriptionId"`
	Icon           *notes.Icon `json:"icon"`
	FaviconID      *string     `json:"faviconId"`

	origin string
}

// Item is a feed item as a user sees it.
type Item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Link        *string  `json:"link"`
	Summary     *string  `json:"summary"`
	PublishedAt string   `json:"publishedAt"`
	Audio       *Audio   `json:"audio"`
	Feed        ItemFeed `json:"feed"`
	ReadLaterAt *string  `json:"readLaterAt"`
	LikedAt     *string  `json:"likedAt"`
}

// Patch lists a subscription's fields to change; nil fields stay as they
// are, and a nil inner value removes the field.
type Patch struct {
	CustomTitle **string
	Icon        **notes.Icon
}

// Store keeps feeds, items, subscriptions and flags.
type Store struct {
	DB      *sql.DB
	Fetcher *Fetcher
	Icons   Icons
	Now     func() time.Time
	// Interval is the refresh cycle.
	Interval time.Duration
	// Keep is how many of the newest items are kept per feed.
	Keep int
	// Jitter spreads refreshes over part of a cycle.
	Jitter func(interval time.Duration) time.Duration
}

// NewStore returns a store that uses the real clock.
func NewStore(database *sql.DB, fetcher *Fetcher, icons Icons, interval time.Duration, keep int) *Store {
	return &Store{
		DB: database, Fetcher: fetcher, Icons: icons, Now: time.Now, Interval: interval, Keep: keep,
		Jitter: func(d time.Duration) time.Duration { return time.Duration(rand.Int64N(int64(d/10) + 1)) },
	}
}

func (s *Store) nextFetch(now time.Time) string {
	return db.FormatTime(now.Add(s.Interval + s.Jitter(s.Interval)))
}

// resolved is a fetched feed, ready to be stored.
type resolved struct {
	url    string
	parsed *Parsed
	valid  Conditional
}

// resolve fetches a feed URL, or a page that advertises one.
func (s *Store) resolve(ctx context.Context, u *url.URL) (resolved, error) {
	ctx, cancel := context.WithTimeout(ctx, s.Fetcher.Timeout)
	defer cancel()
	resp, err := s.Fetcher.getUpgraded(ctx, u)
	if err != nil {
		return resolved{}, err
	}
	if isHTML(resp.ContentType, resp.Body) {
		base, err := url.Parse(resp.URL)
		if err != nil {
			return resolved{}, ErrNotAFeed
		}
		target, ok := discover(resp.Body, base)
		if !ok {
			return resolved{}, ErrNotAFeed
		}
		feedURL, err := parseURL(target)
		if err != nil {
			return resolved{}, ErrNotAFeed
		}
		if resp, err = s.Fetcher.getUpgraded(ctx, feedURL); err != nil {
			return resolved{}, err
		}
		if isHTML(resp.ContentType, resp.Body) {
			return resolved{}, ErrNotAFeed
		}
	}
	parsed, err := Parse(resp.Body, s.Keep, s.Now())
	if err != nil {
		return resolved{}, err
	}
	return resolved{url: resp.URL, parsed: parsed, valid: resp.Validators}, nil
}

// Subscribe subscribes the user to the feed at rawURL, or the one a page at
// rawURL advertises, at the end of their list. A feed that is already stored
// and has been fetched successfully isn't fetched again.
func (s *Store) Subscribe(ctx context.Context, userID, rawURL string) (Subscription, error) {
	u, err := parseURL(rawURL)
	if err != nil {
		return Subscription{}, err
	}
	feedID, err := s.knownFeed(ctx, u)
	if err != nil {
		return Subscription{}, err
	}
	var fetched *resolved
	if feedID == "" {
		r, err := s.resolve(ctx, u)
		if err != nil {
			return Subscription{}, err
		}
		fetched = &r
	}

	now := s.Now()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Subscription{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if fetched != nil {
		if feedID, err = s.ensureFeed(ctx, tx, fetched.url, now); err != nil {
			return Subscription{}, err
		}
		if err := s.apply(ctx, tx, feedID, fetched.parsed, fetched.valid, now); err != nil {
			return Subscription{}, err
		}
	} else if _, err := tx.ExecContext(ctx,
		// A feed nobody subscribed to has stopped refreshing.
		`UPDATE feeds SET next_fetch_at = ? WHERE id = ?
		   AND NOT EXISTS (SELECT 1 FROM feed_subscriptions WHERE feed_id = feeds.id)`,
		db.FormatTime(now), feedID); err != nil {
		return Subscription{}, err
	}
	id, added, err := s.insertSubscription(ctx, tx, userID, feedID, nil, nil, nil, now)
	if err != nil {
		return Subscription{}, err
	}
	if !added {
		return Subscription{}, ErrAlreadySubscribed
	}
	if err := tx.Commit(); err != nil {
		return Subscription{}, err
	}
	if fetched != nil {
		s.queueIcon(fetched)
	}
	return s.Get(ctx, userID, id)
}

func (s *Store) queueIcon(r *resolved) {
	if s.Icons == nil {
		return
	}
	target := r.url
	if r.parsed.SiteURL != nil {
		target = *r.parsed.SiteURL
	}
	s.Icons.Queue(target)
}

// knownFeed finds a successfully fetched feed by u or, for an http URL, by
// its https twin.
func (s *Store) knownFeed(ctx context.Context, u *url.URL) (string, error) {
	secure := *u
	secure.Scheme = "https"
	var id string
	err := s.DB.QueryRowContext(ctx,
		`SELECT id FROM feeds WHERE url IN (?, ?) AND last_success_at IS NOT NULL ORDER BY url = ? DESC LIMIT 1`,
		u.String(), secure.String(), u.String()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// ensureFeed returns the ID of the feed with this URL, creating it if needed.
// The UNIQUE constraint on url decides races between users.
func (s *Store) ensureFeed(ctx context.Context, q db.Querier, feedURL string, now time.Time) (string, error) {
	u, err := url.Parse(feedURL)
	if err != nil {
		return "", ErrInvalidURL
	}
	if _, err := q.ExecContext(ctx,
		`INSERT INTO feeds (id, url, title, next_fetch_at, created_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (url) DO NOTHING`,
		uuid.NewString(), feedURL, u.Hostname(), db.FormatTime(now), db.FormatTime(now)); err != nil {
		return "", err
	}
	var id string
	err = q.QueryRowContext(ctx, `SELECT id FROM feeds WHERE url = ?`, feedURL).Scan(&id)
	return id, err
}

// apply stores a successful fetch. Flags are in their own table, so an
// updated item keeps them; a fetch makes an imported item an ordinary one.
func (s *Store) apply(ctx context.Context, q db.Querier, feedID string, p *Parsed, valid Conditional, now time.Time) error {
	stamp := db.FormatTime(now)
	title := p.Title
	titleSet := title != ""
	if _, err := q.ExecContext(ctx,
		`UPDATE feeds SET title = CASE WHEN ? THEN ? ELSE title END, site_url = ?, etag = ?, last_modified = ?,
		   last_success_at = ?, last_fetched_at = ?, next_fetch_at = ?, failures = 0
		 WHERE id = ?`,
		titleSet, title, p.SiteURL, emptyNil(valid.ETag), emptyNil(valid.LastModified), stamp, stamp, s.nextFetch(now), feedID); err != nil {
		return err
	}
	for _, item := range p.Items {
		var audioURL, audioType *string
		var seconds *int
		if item.Audio != nil {
			audioURL, audioType, seconds = &item.Audio.URL, item.Audio.Type, item.Audio.DurationSeconds
		}
		if _, err := q.ExecContext(ctx,
			`INSERT INTO feed_items (id, feed_id, identity, title, link, summary, item_time, first_seen_at, audio_url, audio_type, audio_seconds)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (feed_id, identity) DO UPDATE SET
			   title = excluded.title, link = excluded.link, summary = excluded.summary,
			   item_time = MIN(excluded.item_time, feed_items.first_seen_at),
			   audio_url = excluded.audio_url, audio_type = excluded.audio_type, audio_seconds = excluded.audio_seconds,
			   imported_by = NULL`,
			uuid.NewString(), feedID, item.Identity, item.Title, item.Link, item.Summary, db.FormatTime(item.Time), stamp, audioURL, audioType, seconds,
		); err != nil {
			return err
		}
	}
	return nil
}

func emptyNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// insertSubscription adds a subscription at the end of the user's list. When
// the user already has one, it returns its ID and false.
func (s *Store) insertSubscription(ctx context.Context, q db.Querier, userID, feedID string, customTitle, symbol, imageID *string, now time.Time) (string, bool, error) {
	id := uuid.NewString()
	stamp := db.FormatTime(now)
	res, err := q.ExecContext(ctx,
		`INSERT INTO feed_subscriptions (id, user_id, feed_id, custom_title, icon_symbol, icon_image_id, position, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM feed_subscriptions WHERE user_id = ?), ?, ?)
		 ON CONFLICT (user_id, feed_id) DO NOTHING`,
		id, userID, feedID, customTitle, symbol, imageID, userID, stamp, stamp)
	if err != nil {
		return "", false, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return "", false, err
	} else if n == 0 {
		err := q.QueryRowContext(ctx, `SELECT id FROM feed_subscriptions WHERE user_id = ? AND feed_id = ?`, userID, feedID).Scan(&id)
		return id, false, err
	}
	return id, true, nil
}

const subscriptionColumns = `s.id, s.feed_id, f.url, COALESCE(s.custom_title, f.title), s.custom_title, f.site_url,
	s.icon_symbol, s.icon_image_id, s.position, f.last_fetched_at, f.failures`

func scanSubscription(row interface{ Scan(...any) error }) (Subscription, error) {
	var sub Subscription
	var symbol, imageID *string
	var failures int
	if err := row.Scan(&sub.ID, &sub.FeedID, &sub.URL, &sub.Title, &sub.CustomTitle, &sub.SiteURL, &symbol, &imageID, &sub.Position, &sub.LastFetchedAt, &failures); err != nil {
		return Subscription{}, err
	}
	sub.Icon = notes.IconOf(symbol, imageID)
	sub.Failing = failures >= FailingAfter
	return sub, nil
}

// List returns the user's subscriptions in their order.
func (s *Store) List(ctx context.Context, userID string) ([]Subscription, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+subscriptionColumns+` FROM feed_subscriptions s JOIN feeds f ON f.id = s.feed_id
		 WHERE s.user_id = ? ORDER BY s.position, s.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Subscription{}
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	origins := make([]string, len(list))
	for i := range list {
		origins[i] = feedOrigin(list[i].SiteURL, list[i].URL)
	}
	icons := s.lookupIcons(ctx, origins)
	for i := range list {
		if id, ok := icons[origins[i]]; ok {
			list[i].FaviconID = &id
		}
	}
	return list, nil
}

func feedOrigin(siteURL *string, feedURL string) string {
	target := feedURL
	if siteURL != nil {
		target = *siteURL
	}
	origin, err := images.Origin(target)
	if err != nil {
		return ""
	}
	return origin
}

// lookupIcons never fails: a feed without its favicon still works.
func (s *Store) lookupIcons(ctx context.Context, origins []string) map[string]string {
	if s.Icons == nil {
		return nil
	}
	seen := map[string]bool{}
	unique := []string{}
	for _, o := range origins {
		if o != "" && !seen[o] {
			seen[o] = true
			unique = append(unique, o)
		}
	}
	icons, err := s.Icons.Lookup(ctx, unique)
	if err != nil {
		slog.ErrorContext(ctx, "looking up feed favicons failed", "error", err)
		return nil
	}
	return icons
}

// Get returns one of the user's subscriptions.
func (s *Store) Get(ctx context.Context, userID, id string) (Subscription, error) {
	sub, err := scanSubscription(s.DB.QueryRowContext(ctx,
		`SELECT `+subscriptionColumns+` FROM feed_subscriptions s JOIN feeds f ON f.id = s.feed_id
		 WHERE s.id = ? AND s.user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Subscription{}, ErrNotFound
	}
	if err != nil {
		return Subscription{}, err
	}
	origin := feedOrigin(sub.SiteURL, sub.URL)
	if icon, ok := s.lookupIcons(ctx, []string{origin})[origin]; ok {
		sub.FaviconID = &icon
	}
	return sub, nil
}

// Update changes the fields in p.
func (s *Store) Update(ctx context.Context, userID, id string, p Patch) (Subscription, error) {
	if _, err := s.Get(ctx, userID, id); err != nil {
		return Subscription{}, err
	}
	sets, args := []string{}, []any{}
	if p.CustomTitle != nil {
		var title *string
		if *p.CustomTitle != nil {
			t := strings.TrimSpace(**p.CustomTitle)
			if utf8.RuneCountInString(t) > MaxCustomTitle {
				return Subscription{}, ErrInvalidTitle
			}
			if t != "" {
				title = &t
			}
		}
		sets, args = append(sets, "custom_title = ?"), append(args, title)
	}
	if p.Icon != nil {
		symbol, imageID, err := checkIcon(ctx, s.DB, userID, *p.Icon)
		if err != nil {
			return Subscription{}, err
		}
		sets, args = append(sets, "icon_symbol = ?", "icon_image_id = ?"), append(args, symbol, imageID)
	}
	if len(sets) == 0 {
		return s.Get(ctx, userID, id)
	}
	sets, args = append(sets, "updated_at = ?"), append(args, db.FormatTime(s.Now()), id, userID)
	if _, err := s.DB.ExecContext(ctx,
		`UPDATE feed_subscriptions SET `+strings.Join(sets, ", ")+` WHERE id = ? AND user_id = ?`, args...); err != nil {
		return Subscription{}, err
	}
	return s.Get(ctx, userID, id)
}

func checkIcon(ctx context.Context, q db.Querier, userID string, icon *notes.Icon) (symbol, imageID *string, err error) {
	symbol, imageID, err = notes.CheckIcon(ctx, q, userID, icon)
	if errors.Is(err, notes.ErrInvalidIcon) {
		err = ErrInvalidIcon
	}
	return symbol, imageID, err
}

// Reorder sets the order of the user's subscriptions. ids must be exactly
// the IDs of all of them.
func (s *Store) Reorder(ctx context.Context, userID string, ids []string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM feed_subscriptions WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return err
	}
	if count != len(ids) {
		return ErrInvalidOrder
	}
	seen := make(map[string]bool, len(ids))
	now := db.FormatTime(s.Now())
	for i, id := range ids {
		if seen[id] {
			return ErrInvalidOrder
		}
		seen[id] = true
		res, err := tx.ExecContext(ctx,
			`UPDATE feed_subscriptions SET position = ?, updated_at = ? WHERE id = ? AND user_id = ?`, i, now, id, userID)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil {
			return err
		} else if n == 0 {
			return ErrInvalidOrder
		}
	}
	return tx.Commit()
}

// Unsubscribe removes one of the user's subscriptions. The feed stays until
// the cleanup finds it unused.
func (s *Store) Unsubscribe(ctx context.Context, userID, id string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM feed_subscriptions WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// HasSubscription tells whether any of ids is a subscription of the user.
func (s *Store) HasSubscription(ctx context.Context, userID string, ids []string) (bool, error) {
	args := []any{userID}
	for _, id := range ids {
		args = append(args, id)
	}
	var exists bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM feed_subscriptions WHERE user_id = ? AND id IN (?`+strings.Repeat(", ?", len(ids)-1)+`))`, args...).Scan(&exists)
	return exists, err
}

// ItemQuery selects a user's items.
type ItemQuery struct {
	View string
	// SubscriptionIDs limits the items to these feeds; empty means all.
	SubscriptionIDs []string
	// AudioOnly limits the items to those with audio.
	AudioOnly bool
	Limit     int
	Cursor    string
}

// itemFrom is the FROM clause of a view. Each binds the user three times.
func itemFrom(view string) (from, sortKey, flagged string, err error) {
	switch view {
	case ViewLatest:
		return `feed_items i
			JOIN feed_subscriptions s ON s.feed_id = i.feed_id AND s.user_id = ?
			JOIN feeds f ON f.id = i.feed_id
			LEFT JOIN feed_item_flags fl ON fl.item_id = i.id AND fl.user_id = ?`, "i.item_time", "", nil
	case ViewReadLater, ViewLiked:
		column := "fl.read_later_at"
		if view == ViewLiked {
			column = "fl.liked_at"
		}
		return `feed_item_flags fl
			JOIN feed_items i ON i.id = fl.item_id
			JOIN feeds f ON f.id = i.feed_id
			LEFT JOIN feed_subscriptions s ON s.feed_id = i.feed_id AND s.user_id = ?`, column, ` AND fl.user_id = ? AND ` + column + ` IS NOT NULL`, nil
	}
	return "", "", "", ErrInvalidQuery
}

const itemColumns = `i.id, i.title, i.link, i.summary, i.item_time, i.audio_url, i.audio_type, i.audio_seconds,
	COALESCE(s.custom_title, f.title), s.id, s.icon_symbol, s.icon_image_id, f.site_url, f.url,
	fl.read_later_at, fl.liked_at`

func scanItem(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	var audioURL, audioType, symbol, imageID, siteURL *string
	var seconds *int
	var feedURL string
	if err := row.Scan(&it.ID, &it.Title, &it.Link, &it.Summary, &it.PublishedAt, &audioURL, &audioType, &seconds,
		&it.Feed.Title, &it.Feed.SubscriptionID, &symbol, &imageID, &siteURL, &feedURL, &it.ReadLaterAt, &it.LikedAt); err != nil {
		return Item{}, err
	}
	if audioURL != nil {
		it.Audio = &Audio{URL: *audioURL, Type: audioType, DurationSeconds: seconds}
	}
	it.Feed.Icon = notes.IconOf(symbol, imageID)
	it.Feed.origin = feedOrigin(siteURL, feedURL)
	return it, nil
}

func encodeCursor(key, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(key + "\x00" + id))
}

func decodeCursor(cursor string) (key, id string, err error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", "", ErrInvalidQuery
	}
	key, id, ok := strings.Cut(string(raw), "\x00")
	if !ok {
		return "", "", ErrInvalidQuery
	}
	return key, id, nil
}

// Items returns a page of the user's items and the cursor of the next page,
// or "" for the last one.
func (s *Store) Items(ctx context.Context, userID string, q ItemQuery) ([]Item, string, error) {
	from, sortKey, flagged, err := itemFrom(q.View)
	if err != nil {
		return nil, "", err
	}
	if q.Limit < 1 || q.Limit > 100 {
		return nil, "", ErrInvalidQuery
	}
	args := []any{userID}
	if q.View == ViewLatest {
		args = append(args, userID)
	}
	where := `(i.imported_by IS NULL OR i.imported_by = ?)`
	args = append(args, userID)
	if flagged != "" {
		where += flagged
		args = append(args, userID)
	}
	if len(q.SubscriptionIDs) > 0 {
		where += ` AND s.id IN (?` + strings.Repeat(", ?", len(q.SubscriptionIDs)-1) + `)`
		for _, id := range q.SubscriptionIDs {
			args = append(args, id)
		}
	}
	if q.AudioOnly {
		where += ` AND i.audio_url IS NOT NULL`
	}
	if q.Cursor != "" {
		key, id, err := decodeCursor(q.Cursor)
		if err != nil {
			return nil, "", err
		}
		where += fmt.Sprintf(` AND (%[1]s < ? OR (%[1]s = ? AND i.id < ?))`, sortKey)
		args = append(args, key, key, id)
	}
	args = append(args, q.Limit+1)
	selectKey := `, ` + sortKey
	rows, err := s.DB.QueryContext(ctx,
		`SELECT `+itemColumns+selectKey+` FROM `+from+` WHERE `+where+
			` ORDER BY `+sortKey+` DESC, i.id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := []Item{}
	keys := []string{}
	for rows.Next() {
		var key string
		it, err := scanItem(keyedRow{rows, &key})
		if err != nil {
			return nil, "", err
		}
		items = append(items, it)
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > q.Limit {
		items = items[:q.Limit]
		next = encodeCursor(keys[q.Limit-1], items[q.Limit-1].ID)
	}
	s.setFavicons(ctx, items)
	return items, next, nil
}

// keyedRow scans the sort key that follows an item's columns.
type keyedRow struct {
	rows *sql.Rows
	key  *string
}

func (k keyedRow) Scan(dest ...any) error {
	return k.rows.Scan(append(dest, k.key)...)
}

func (s *Store) setFavicons(ctx context.Context, items []Item) {
	origins := make([]string, len(items))
	for i := range items {
		origins[i] = items[i].Feed.origin
	}
	icons := s.lookupIcons(ctx, origins)
	for i := range items {
		if id, ok := icons[items[i].Feed.origin]; ok {
			items[i].Feed.FaviconID = &id
		}
	}
}

// visibleItem loads an item that the user subscribes to or has flagged.
func (s *Store) visibleItem(ctx context.Context, userID, itemID string) (Item, error) {
	it, err := scanItem(s.DB.QueryRowContext(ctx,
		`SELECT `+itemColumns+` FROM feed_items i
		 JOIN feeds f ON f.id = i.feed_id
		 LEFT JOIN feed_subscriptions s ON s.feed_id = i.feed_id AND s.user_id = ?
		 LEFT JOIN feed_item_flags fl ON fl.item_id = i.id AND fl.user_id = ?
		 WHERE i.id = ? AND (i.imported_by IS NULL OR i.imported_by = ?)
		   AND (s.id IS NOT NULL OR fl.item_id IS NOT NULL)`,
		userID, userID, itemID, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	items := []Item{it}
	s.setFavicons(ctx, items)
	return items[0], nil
}

// SetFlags changes the user's flags on an item; a nil flag stays as it is.
// A row without flags is removed.
func (s *Store) SetFlags(ctx context.Context, userID, itemID string, readLater, liked *bool) (Item, error) {
	it, err := s.visibleItem(ctx, userID, itemID)
	if err != nil {
		return Item{}, err
	}
	now := db.FormatTime(s.Now())
	it.ReadLaterAt = nextFlag(it.ReadLaterAt, readLater, now)
	it.LikedAt = nextFlag(it.LikedAt, liked, now)
	if it.ReadLaterAt == nil && it.LikedAt == nil {
		_, err = s.DB.ExecContext(ctx, `DELETE FROM feed_item_flags WHERE user_id = ? AND item_id = ?`, userID, itemID)
	} else {
		_, err = s.DB.ExecContext(ctx,
			`INSERT INTO feed_item_flags (user_id, item_id, read_later_at, liked_at) VALUES (?, ?, ?, ?)
			 ON CONFLICT (user_id, item_id) DO UPDATE SET read_later_at = excluded.read_later_at, liked_at = excluded.liked_at`,
			userID, itemID, it.ReadLaterAt, it.LikedAt)
	}
	if err != nil {
		return Item{}, err
	}
	return it, nil
}

func nextFlag(current *string, set *bool, now string) *string {
	switch {
	case set == nil:
		return current
	case !*set:
		return nil
	case current != nil:
		return current
	}
	return &now
}
