// Package offline keeps offline copies of links: a readable article of the
// page and a screenshot, captured in the background as jobs of the
// persistent queue, and only ever shown to the link's owner.
package offline

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"boardly/internal/blobs"
	"boardly/internal/db"
)

// States of an offline copy.
const (
	StatusNone    = "none"
	StatusPending = "pending"
	StatusReady   = "ready"
	StatusFailed  = "failed"
)

// Reasons a capture failed.
const (
	ReasonFetchFailed    = "fetch_failed"
	ReasonBlockedAddress = "blocked_address"
	ReasonTooLarge       = "too_large"
	ReasonNotHTML        = "not_html"
	ReasonNoContent      = "no_content"
)

// ErrNotFound is returned for a node that isn't a link of the user, and for a
// part the copy doesn't have.
var ErrNotFound = errors.New("offline copy not found")

// Copy is the state of a link's offline copy.
type Copy struct {
	Status        string     `json:"status"`
	CapturedAt    *time.Time `json:"capturedAt"`
	Title         *string    `json:"title"`
	HasArticle    bool       `json:"hasArticle"`
	HasScreenshot bool       `json:"hasScreenshot"`
	Error         *string    `json:"error"`

	articleID, screenshotID string
}

// Store reads and writes offline copies: their records in DB and their files
// in Files.
type Store struct {
	DB    *sql.DB
	Files blobs.Dir
	Now   func() time.Time
}

// NewStore returns a store for <dataDir>/offline, creating the directory.
func NewStore(database *sql.DB, dataDir string) (*Store, error) {
	dir, err := blobs.Open(filepath.Join(dataDir, "offline"))
	if err != nil {
		return nil, err
	}
	return &Store{DB: database, Files: dir, Now: time.Now}, nil
}

// Get returns the offline copy of the user's link nodeID. A link without a
// copy has the status none. A copy that is pending but whose capture is no
// longer queued, because it was dropped after too many interrupts, is
// failed. Folders, other users' nodes and unknown IDs give ErrNotFound.
func (s *Store) Get(ctx context.Context, userID, nodeID string) (Copy, error) {
	var isLink bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT kind = 'link' FROM favourite_nodes WHERE id = ? AND user_id = ?`, nodeID, userID,
	).Scan(&isLink)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !isLink) {
		return Copy{}, ErrNotFound
	}
	if err != nil {
		return Copy{}, err
	}
	var (
		c                                      Copy
		reason, title, article, shot, captured sql.NullString
		queued                                 bool
	)
	err = s.DB.QueryRowContext(ctx,
		`SELECT status, error, title, article_id, screenshot_id, captured_at,
		   EXISTS (SELECT 1 FROM jobs WHERE kind = 'offline_capture' AND dedup_key = offline_copies.node_id)
		 FROM offline_copies WHERE node_id = ?`, nodeID,
	).Scan(&c.Status, &reason, &title, &article, &shot, &captured, &queued)
	if errors.Is(err, sql.ErrNoRows) {
		return Copy{Status: StatusNone}, nil
	}
	if err != nil {
		return Copy{}, err
	}
	if c.Status == StatusPending && !queued {
		c.Status = StatusFailed
		reason = sql.NullString{String: ReasonFetchFailed, Valid: true}
	}
	if reason.Valid && c.Status == StatusFailed {
		c.Error = &reason.String
	}
	if title.Valid {
		c.Title = &title.String
	}
	if captured.Valid {
		t, err := db.ParseTime(captured.String)
		if err != nil {
			return Copy{}, err
		}
		c.CapturedAt = &t
	}
	c.articleID, c.screenshotID = article.String, shot.String
	c.HasArticle, c.HasScreenshot = article.Valid, shot.Valid
	return c, nil
}

// Part names a file of an offline copy.
type Part int

// Parts of an offline copy.
const (
	PartArticle Part = iota
	PartScreenshot
)

// Open opens a part of the user's link's copy. ErrNotFound when the node
// isn't the user's link or the copy doesn't have that part.
func (s *Store) Open(ctx context.Context, userID, nodeID string, part Part) (*os.File, error) {
	c, err := s.Get(ctx, userID, nodeID)
	if err != nil {
		return nil, err
	}
	id := c.articleID
	if part == PartScreenshot {
		id = c.screenshotID
	}
	if id == "" {
		return nil, ErrNotFound
	}
	f, err := os.Open(s.Files.Path(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

// Result is what a capture got.
type Result struct {
	Title      string
	Article    []byte
	Screenshot []byte
	// Reason is why nothing was captured, when Article and Screenshot are nil.
	Reason string
}

// Record stores the result of the capture of nodeID for rawURL. It reports
// false, and stores nothing, when the link was deleted or its copy is for
// another URL by now. A result with a part replaces the copy's parts, title
// and capture time; one without marks the copy failed and keeps them.
func (s *Store) Record(ctx context.Context, nodeID, rawURL string, r Result) (bool, error) {
	now := db.FormatTime(s.Now())
	if r.Article == nil && r.Screenshot == nil {
		res, err := s.DB.ExecContext(ctx,
			`UPDATE offline_copies SET status = 'failed', error = ?, updated_at = ? WHERE node_id = ? AND url = ?`,
			r.Reason, now, nodeID, rawURL)
		if err != nil {
			return false, err
		}
		n, err := res.RowsAffected()
		return n > 0, err
	}
	var article, shot sql.NullString
	written := []string{}
	for _, f := range []struct {
		data []byte
		id   *sql.NullString
	}{{r.Article, &article}, {r.Screenshot, &shot}} {
		if f.data == nil {
			continue
		}
		id := blobs.NewID()
		if err := s.Files.Write(id, f.data); err != nil {
			s.remove(written)
			return false, err
		}
		written = append(written, id)
		*f.id = sql.NullString{String: id, Valid: true}
	}
	var title sql.NullString
	if r.Title != "" {
		title = sql.NullString{String: r.Title, Valid: true}
	}
	res, err := s.DB.ExecContext(ctx,
		`UPDATE offline_copies SET status = 'ready', error = NULL, title = ?, article_id = ?, screenshot_id = ?,
		   captured_at = ?, updated_at = ?
		 WHERE node_id = ? AND url = ?`,
		title, article, shot, now, now, nodeID, rawURL)
	var n int64
	if err == nil {
		n, err = res.RowsAffected()
	}
	if err != nil || n == 0 {
		s.remove(written)
		return false, err
	}
	return true, nil
}

func (s *Store) remove(ids []string) {
	for _, id := range ids {
		_ = s.Files.Remove(id)
	}
}

// Sweep deletes the files that no offline copy refers to any more, once
// they are older than blobs.OrphanAge: parts that a capture replaced, and
// parts of copies whose link was deleted or got a new URL.
func (s *Store) Sweep(ctx context.Context) error {
	return s.Files.RemoveOrphans(ctx, s.Now(), func(ctx context.Context, id string) (bool, error) {
		var used bool
		err := s.DB.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM offline_copies WHERE article_id = ? OR screenshot_id = ?)`, id, id,
		).Scan(&used)
		return used, err
	})
}

// RunCleanup sweeps now and then every interval until ctx is done.
func (s *Store) RunCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "cleaning up unused offline copy files failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Queuer queues the capture of a link's offline copy in a transaction (see
// favourites.Store.QueueOfflineCapture).
type Queuer interface {
	QueueOfflineCapture(ctx context.Context, q db.Querier, userID, nodeID, rawURL string, setPreview bool) error
	NotifyOfflineCapture()
}

// Refresh queues a new capture of the user's link nodeID for its current
// URL and returns the copy, now pending. A refresh never sets the link's
// preview. ErrNotFound as for Get.
func (s *Store) Refresh(ctx context.Context, q Queuer, userID, nodeID string) (Copy, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Copy{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var rawURL string
	err = tx.QueryRowContext(ctx,
		`SELECT url FROM favourite_nodes WHERE id = ? AND user_id = ? AND kind = 'link'`, nodeID, userID,
	).Scan(&rawURL)
	if errors.Is(err, sql.ErrNoRows) {
		return Copy{}, ErrNotFound
	}
	if err != nil {
		return Copy{}, err
	}
	if err := q.QueueOfflineCapture(ctx, tx, userID, nodeID, rawURL, false); err != nil {
		return Copy{}, err
	}
	if err := tx.Commit(); err != nil {
		return Copy{}, err
	}
	q.NotifyOfflineCapture()
	return s.Get(ctx, userID, nodeID)
}
