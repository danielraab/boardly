package feeds

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"boardly/internal/db"
	"boardly/internal/notes"
)

// Limits of the background refresh.
const (
	refreshWorkers = 4
	refreshBatch   = 200
	maxBackoff     = 24 * time.Hour
)

type dueFeed struct {
	id, url  string
	valid    Conditional
	failures int
}

// RefreshDue fetches the feeds that are due and have subscribers, and
// returns how many it tried.
func (s *Store) RefreshDue(ctx context.Context) (int, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, url, COALESCE(etag, ''), COALESCE(last_modified, ''), failures FROM feeds
		 WHERE next_fetch_at <= ? AND EXISTS (SELECT 1 FROM feed_subscriptions WHERE feed_id = feeds.id)
		 ORDER BY next_fetch_at LIMIT ?`, db.FormatTime(s.Now()), refreshBatch)
	if err != nil {
		return 0, err
	}
	var due []dueFeed
	for rows.Next() {
		var f dueFeed
		if err := rows.Scan(&f.id, &f.url, &f.valid.ETag, &f.valid.LastModified, &f.failures); err != nil {
			_ = rows.Close()
			return 0, err
		}
		due = append(due, f)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var group errgroup.Group
	group.SetLimit(refreshWorkers)
	for _, f := range due {
		group.Go(func() error {
			if err := s.refresh(ctx, f); err != nil && ctx.Err() == nil {
				slog.ErrorContext(ctx, "storing a feed refresh failed", "feed", f.id, "error", err)
			}
			return nil
		})
	}
	_ = group.Wait()
	return len(due), nil
}

func (s *Store) refresh(ctx context.Context, f dueFeed) error {
	resp, err := s.Fetcher.Get(ctx, f.url, f.valid)
	if err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return s.recordFailure(ctx, f, err)
	}
	now := s.Now()
	if resp.NotModified {
		stamp := db.FormatTime(now)
		_, err := s.DB.ExecContext(ctx,
			`UPDATE feeds SET last_success_at = ?, last_fetched_at = ?, next_fetch_at = ?, failures = 0 WHERE id = ?`,
			stamp, stamp, s.nextFetch(now), f.id)
		return err
	}
	parsed, err := Parse(resp.Body, s.Keep, now)
	if err != nil {
		return s.recordFailure(ctx, f, err)
	}

	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	target := f.id
	if resp.Moved && resp.URL != f.url {
		if target, err = s.move(ctx, tx, f.id, resp.URL); err != nil {
			return err
		}
	}
	if err := s.apply(ctx, tx, target, parsed, resp.Validators, now); err != nil {
		return err
	}
	if err := s.trim(ctx, tx, target); err != nil {
		return err
	}
	return tx.Commit()
}

// recordFailure doubles the wait for each failure in a row, up to a limit,
// and honours a longer wait that the host asked for.
func (s *Store) recordFailure(ctx context.Context, f dueFeed, cause error) error {
	failures := f.failures + 1
	limit := max(maxBackoff, s.Interval)
	wait := s.Interval
	for i := 0; i < failures && wait < limit; i++ {
		wait *= 2
	}
	wait = min(wait, limit)
	var fetchErr *FetchError
	if errors.As(cause, &fetchErr) && fetchErr.RetryAfter > wait {
		wait = fetchErr.RetryAfter
	}
	slog.DebugContext(ctx, "refreshing a feed failed", "feed", f.id, "failures", failures, "error", cause)
	now := s.Now()
	_, err := s.DB.ExecContext(ctx,
		`UPDATE feeds SET last_fetched_at = ?, next_fetch_at = ?, failures = ? WHERE id = ?`,
		db.FormatTime(now), db.FormatTime(now.Add(wait)), failures, f.id)
	return err
}

// move gives a feed that permanently redirected its new URL. When another
// feed already has that URL, the two become one, and that one's ID is
// returned.
func (s *Store) move(ctx context.Context, tx *sql.Tx, feedID, newURL string) (string, error) {
	var other string
	err := tx.QueryRowContext(ctx, `SELECT id FROM feeds WHERE url = ?`, newURL).Scan(&other)
	if errors.Is(err, sql.ErrNoRows) {
		_, err := tx.ExecContext(ctx, `UPDATE feeds SET url = ? WHERE id = ?`, newURL, feedID)
		return feedID, err
	}
	if err != nil {
		return "", err
	}
	if err := s.merge(ctx, tx, feedID, other); err != nil {
		return "", err
	}
	return other, nil
}

type mergeItem struct {
	from, into             string
	fromPublic, intoPublic bool
}

// merge moves everything of feed from to feed into and deletes from. An item
// in both is kept once with both users' flags, and a user subscribed to both
// keeps the subscription they made first.
func (s *Store) merge(ctx context.Context, tx *sql.Tx, from, into string) error {
	rows, err := tx.QueryContext(ctx,
		`SELECT a.id, b.id, a.imported_by IS NULL, b.imported_by IS NULL
		 FROM feed_items a JOIN feed_items b ON b.feed_id = ? AND b.identity = a.identity
		 WHERE a.feed_id = ?`, into, from)
	if err != nil {
		return err
	}
	var both []mergeItem
	for rows.Next() {
		var m mergeItem
		if err := rows.Scan(&m.from, &m.into, &m.fromPublic, &m.intoPublic); err != nil {
			_ = rows.Close()
			return err
		}
		both = append(both, m)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, m := range both {
		// Fetched content wins over an item only an import vouches for.
		keep, drop := m.into, m.from
		if m.fromPublic && !m.intoPublic {
			keep, drop = m.from, m.into
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO feed_item_flags (user_id, item_id, read_later_at, liked_at)
			 SELECT user_id, ?, read_later_at, liked_at FROM feed_item_flags WHERE item_id = ?
			 ON CONFLICT (user_id, item_id) DO UPDATE SET
			   read_later_at = CASE WHEN read_later_at IS NULL THEN excluded.read_later_at
			                        WHEN excluded.read_later_at IS NULL THEN read_later_at
			                        ELSE MIN(read_later_at, excluded.read_later_at) END,
			   liked_at = CASE WHEN liked_at IS NULL THEN excluded.liked_at
			                   WHEN excluded.liked_at IS NULL THEN liked_at
			                   ELSE MIN(liked_at, excluded.liked_at) END`,
			keep, drop); err != nil {
			return err
		}
		// Of two listen records of one user, the one changed last stays.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO feed_item_listens (user_id, item_id, position_seconds, duration_seconds, started_at, updated_at, finished_at)
			 SELECT user_id, ?, position_seconds, duration_seconds, started_at, updated_at, finished_at
			 FROM feed_item_listens WHERE item_id = ?
			 ON CONFLICT (user_id, item_id) DO UPDATE SET
			   position_seconds = excluded.position_seconds,
			   duration_seconds = excluded.duration_seconds,
			   started_at = MIN(started_at, excluded.started_at),
			   updated_at = excluded.updated_at,
			   finished_at = excluded.finished_at
			 WHERE excluded.updated_at > updated_at`,
			keep, drop); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM feed_items WHERE id = ?`, drop); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE feed_items SET feed_id = ? WHERE feed_id = ?`, into, from); err != nil {
		return err
	}
	// Of two subscriptions of one user, the later one goes.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM feed_subscriptions WHERE feed_id IN (?, ?) AND id IN (
		   SELECT a.id FROM feed_subscriptions a JOIN feed_subscriptions b
		     ON b.user_id = a.user_id AND b.feed_id IN (?, ?) AND b.id <> a.id
		   WHERE a.feed_id IN (?, ?) AND (a.created_at > b.created_at OR (a.created_at = b.created_at AND a.id > b.id)))`,
		from, into, from, into, from, into); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE feed_subscriptions SET feed_id = ? WHERE feed_id = ?`, into, from); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `DELETE FROM feeds WHERE id = ?`, from)
	return err
}

// trim applies the cleanup rule to one feed. Only items everyone can see
// count as the newest ones, so that items of an import can't push them out.
func (s *Store) trim(ctx context.Context, q db.Querier, feedID string) error {
	_, err := q.ExecContext(ctx,
		`DELETE FROM feed_items
		 WHERE feed_id = ?
		   AND NOT EXISTS (SELECT 1 FROM feed_item_flags WHERE item_id = feed_items.id)
		   AND NOT EXISTS (SELECT 1 FROM feed_item_listens WHERE item_id = feed_items.id)
		   AND id NOT IN (
		     SELECT id FROM feed_items
		     WHERE feed_id = ? AND imported_by IS NULL
		       AND EXISTS (SELECT 1 FROM feed_subscriptions WHERE feed_id = ?)
		     ORDER BY item_time DESC, id DESC LIMIT ?)`,
		feedID, feedID, feedID, s.Keep)
	return err
}

// Cleanup applies the cleanup rule to every feed and removes the feeds
// nobody uses any more. It returns the number of removed feeds.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM feeds`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, id := range ids {
		if err := s.trim(ctx, s.DB, id); err != nil {
			return 0, err
		}
	}
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM feeds
		 WHERE NOT EXISTS (SELECT 1 FROM feed_subscriptions WHERE feed_id = feeds.id)
		   AND NOT EXISTS (SELECT 1 FROM feed_items i JOIN feed_item_flags fl ON fl.item_id = i.id WHERE i.feed_id = feeds.id)
		   AND NOT EXISTS (SELECT 1 FROM feed_items i JOIN feed_item_listens ls ON ls.item_id = i.id WHERE i.feed_id = feeds.id)`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Run refreshes due feeds every tick and cleans up every cleanupEvery, until
// ctx is done.
func (s *Store) Run(ctx context.Context, tick, cleanupEvery time.Duration) {
	refresh := time.NewTicker(tick)
	defer refresh.Stop()
	cleanup := time.NewTicker(cleanupEvery)
	defer cleanup.Stop()
	s.runCleanup(ctx)
	for {
		if _, err := s.RefreshDue(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "refreshing feeds failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-refresh.C:
		case <-cleanup.C:
			s.runCleanup(ctx)
		}
	}
}

func (s *Store) runCleanup(ctx context.Context) {
	if removed, err := s.Cleanup(ctx); err != nil && ctx.Err() == nil {
		slog.ErrorContext(ctx, "cleaning up feeds failed", "error", err)
	} else if removed > 0 {
		slog.InfoContext(ctx, "cleaned up feeds", "removed", removed)
	}
}

// Imported is an item of an import with flags or a listen record.
type Imported struct {
	Identity    string
	Title       string
	Link        *string
	Summary     *string
	PublishedAt string
	Audio       *Audio
	ReadLaterAt *string
	LikedAt     *string
	Listen      *ImportedListen
}

// ImportedListen is a listen record of an import.
type ImportedListen struct {
	PositionSeconds int
	DurationSeconds *int
	StartedAt       string
	UpdatedAt       string
	FinishedAt      *string
}

// valid returns the record with its times in the stored form, or nil when
// the listen API wouldn't accept its values.
func (l *ImportedListen) valid() *ImportedListen {
	if l == nil || l.PositionSeconds < 0 || l.PositionSeconds > MaxListenSeconds ||
		(l.DurationSeconds != nil && (*l.DurationSeconds <= 0 || *l.DurationSeconds > MaxListenSeconds)) {
		return nil
	}
	stamp := func(v string) (string, bool) {
		t, err := time.Parse(time.RFC3339, v)
		return db.FormatTime(t), err == nil
	}
	out := *l
	var ok bool
	if out.StartedAt, ok = stamp(l.StartedAt); !ok {
		return nil
	}
	if out.UpdatedAt, ok = stamp(l.UpdatedAt); !ok {
		return nil
	}
	if l.FinishedAt != nil {
		f, ok := stamp(*l.FinishedAt)
		if !ok {
			return nil
		}
		out.FinishedAt = &f
	}
	return &out
}

// EnsureFeedTx returns the feed with this URL, creating it when the server
// doesn't have it: from the URL alone, because the first fetch checks and
// fills it. It fetches nothing.
func (s *Store) EnsureFeedTx(ctx context.Context, q db.Querier, rawURL string) (string, error) {
	u, err := normalizeURL(rawURL)
	if err != nil {
		return "", err
	}
	return s.ensureFeed(ctx, q, u, s.Now())
}

// SubscribeTx subscribes the user to a stored feed for an import. It returns
// the subscription's ID, also of one the user already had, and whether it
// was added.
func (s *Store) SubscribeTx(ctx context.Context, q db.Querier, userID, feedID string, customTitle *string, icon *notes.Icon) (string, bool, error) {
	symbol, imageID, err := checkIcon(ctx, q, userID, icon)
	if err != nil {
		return "", false, err
	}
	if customTitle != nil {
		t := strings.TrimSpace(*customTitle)
		if t == "" || utf8.RuneCountInString(t) > MaxCustomTitle {
			customTitle = nil
		} else {
			customTitle = &t
		}
	}
	return s.insertSubscription(ctx, q, userID, feedID, customTitle, symbol, imageID, s.Now())
}

// RestoreItemTx applies an item of an import with flags or a listen record.
// It never changes a stored item's content: a known item only gets the
// user's flags and record, and an unknown one is created visible to this user
// alone until a fetch confirms it. Flags and a record the user already has
// stay. An invalid record, or one on an item without audio, is left out. An
// item only another user can see is left alone, and false is returned.
func (s *Store) RestoreItemTx(ctx context.Context, q db.Querier, userID, feedID string, item Imported) (bool, error) {
	flagged := item.ReadLaterAt != nil || item.LikedAt != nil
	listen := item.Listen.valid()
	if strings.TrimSpace(item.Identity) == "" || (!flagged && listen == nil) {
		return false, nil
	}
	var itemID string
	var importedBy *string
	var hasAudio bool
	err := q.QueryRowContext(ctx, `SELECT id, imported_by, audio_url IS NOT NULL FROM feed_items WHERE feed_id = ? AND identity = ?`,
		feedID, item.Identity).Scan(&itemID, &importedBy, &hasAudio)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if item.Audio == nil || httpLink(item.Audio.URL) == nil {
			listen = nil
		}
		if !flagged && listen == nil {
			return false, nil
		}
		if itemID, err = s.insertImported(ctx, q, userID, feedID, item, s.Now()); err != nil || itemID == "" {
			return false, err
		}
	case err != nil:
		return false, err
	case importedBy != nil && *importedBy != userID:
		return false, nil
	case !hasAudio:
		listen = nil
		if !flagged {
			return false, nil
		}
	}
	if flagged {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO feed_item_flags (user_id, item_id, read_later_at, liked_at) VALUES (?, ?, ?, ?)
			 ON CONFLICT (user_id, item_id) DO UPDATE SET
			   read_later_at = COALESCE(read_later_at, excluded.read_later_at),
			   liked_at = COALESCE(liked_at, excluded.liked_at)`,
			userID, itemID, item.ReadLaterAt, item.LikedAt); err != nil {
			return false, err
		}
	}
	if listen != nil {
		if _, err := q.ExecContext(ctx,
			`INSERT INTO feed_item_listens (user_id, item_id, position_seconds, duration_seconds, started_at, updated_at, finished_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?) ON CONFLICT (user_id, item_id) DO NOTHING`,
			userID, itemID, listen.PositionSeconds, listen.DurationSeconds, listen.StartedAt, listen.UpdatedAt, listen.FinishedAt); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (s *Store) insertImported(ctx context.Context, q db.Querier, userID, feedID string, item Imported, now time.Time) (string, error) {
	title := cut(plainText(item.Title), maxTitleLength)
	if title == "" {
		return "", nil
	}
	var link, summary *string
	if item.Link != nil {
		link = httpLink(*item.Link)
	}
	if item.Summary != nil {
		if text := cut(plainText(*item.Summary), maxSummaryLength); text != "" {
			summary = &text
		}
	}
	when := now
	if parsed, err := time.Parse(time.RFC3339, item.PublishedAt); err == nil && parsed.Before(now) {
		when = parsed
	}
	var audioURL, audioType *string
	var seconds *int
	if item.Audio != nil {
		if a := httpLink(item.Audio.URL); a != nil {
			audioURL, audioType, seconds = a, item.Audio.Type, item.Audio.DurationSeconds
		}
	}
	id := uuid.NewString()
	_, err := q.ExecContext(ctx,
		`INSERT INTO feed_items (id, feed_id, identity, title, link, summary, item_time, first_seen_at, audio_url, audio_type, audio_seconds, imported_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, feedID, item.Identity, title, link, summary, db.FormatTime(when), db.FormatTime(now), audioURL, audioType, seconds, userID)
	return id, err
}
