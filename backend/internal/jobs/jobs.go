// Package jobs is a persistent queue of background jobs in the jobs table.
// Jobs are queued with a plain INSERT, also inside a caller's transaction,
// and run by worker pools per kind in the order they were queued. A job that
// was running when the server stopped runs again after the next start.
package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"boardly/internal/db"
)

// Kind is the type of a job.
type Kind string

// Job kinds.
const (
	KindFavicon        Kind = "favicon"
	KindPreviewFetch   Kind = "preview_fetch"
	KindPreviewConvert Kind = "preview_convert"
)

// FaviconPayload asks for the favicon of URL's origin.
type FaviconPayload struct {
	URL string `json:"url"`
}

// PreviewFetchPayload asks for the description and preview image of a link's
// page. URL is the link's URL when the job was queued.
type PreviewFetchPayload struct {
	UserID string `json:"userId"`
	NodeID string `json:"nodeId"`
	URL    string `json:"url"`
}

// PreviewConvertPayload asks to convert the raw image File (a name in the
// images' raw directory) into the preview of a link whose URL is still URL.
// With Replace, it replaces the link's preview; otherwise it is only stored
// when the link has none.
type PreviewConvertPayload struct {
	UserID  string `json:"userId"`
	NodeID  string `json:"nodeId"`
	URL     string `json:"url"`
	File    string `json:"file"`
	Replace bool   `json:"replace"`
}

// MaxAttempts is how often a job may be started. A job that was interrupted
// this often is dropped by Recover, so a job that crashes the server can't
// keep it from starting.
const MaxAttempts = 3

// DefaultPollInterval is how often idle workers look for jobs that were
// queued without waking them, such as jobs inserted in a transaction that
// committed after the wake-up.
const DefaultPollInterval = 30 * time.Second

// replaceOnConflict lists the kinds whose pending job with the same key takes
// the newer payload. Other kinds keep the pending job.
var replaceOnConflict = map[Kind]bool{KindPreviewFetch: true}

// Job is a claimed job.
type Job struct {
	ID       int64
	Kind     Kind
	Payload  json.RawMessage
	Attempts int
}

// Handler runs a job. Its error is logged; the job is done either way.
type Handler func(ctx context.Context, job Job) error

// Queue queues and runs jobs.
type Queue struct {
	DB           *sql.DB
	Now          func() time.Time
	PollInterval time.Duration

	mu   sync.Mutex
	wake map[Kind]chan struct{}
}

// New returns a queue on database that uses the real clock.
func New(database *sql.DB) *Queue {
	return &Queue{DB: database, Now: time.Now, PollInterval: DefaultPollInterval}
}

func (qu *Queue) signal(kind Kind) chan struct{} {
	qu.mu.Lock()
	defer qu.mu.Unlock()
	if qu.wake == nil {
		qu.wake = map[Kind]chan struct{}{}
	}
	ch, ok := qu.wake[kind]
	if !ok {
		ch = make(chan struct{}, 1)
		qu.wake[kind] = ch
	}
	return ch
}

// Notify wakes an idle worker of each kind. Call it after committing a
// transaction that queued jobs.
func (qu *Queue) Notify(kinds ...Kind) {
	for _, k := range kinds {
		select {
		case qu.signal(k) <- struct{}{}:
		default:
		}
	}
}

// Enqueue queues a job; q may be a transaction. An empty key queues the job
// as it is. With a key, a pending job of the same kind and key is combined
// with it: it keeps its payload, or takes the new one for kinds in
// replaceOnConflict.
func (qu *Queue) Enqueue(ctx context.Context, q db.Querier, kind Kind, key string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	var dedup sql.NullString
	if key != "" {
		dedup = sql.NullString{String: key, Valid: true}
	}
	conflict := `ON CONFLICT (kind, dedup_key) WHERE started_at IS NULL AND dedup_key IS NOT NULL DO NOTHING`
	if replaceOnConflict[kind] {
		conflict = `ON CONFLICT (kind, dedup_key) WHERE started_at IS NULL AND dedup_key IS NOT NULL DO UPDATE SET payload = excluded.payload`
	}
	if _, err := q.ExecContext(ctx,
		`INSERT INTO jobs (kind, dedup_key, payload, created_at) VALUES (?, ?, ?, ?) `+conflict,
		kind, dedup, string(data), db.FormatTime(qu.Now()),
	); err != nil {
		return fmt.Errorf("queueing %s job: %w", kind, err)
	}
	qu.Notify(kind)
	return nil
}

// Recover prepares the queue on start, before any worker runs: jobs that
// were running when the server stopped are queued again, except those that
// were started MaxAttempts times, which are dropped and logged. A running job
// whose key already has a pending job is dropped in favour of that one.
func (qu *Queue) Recover(ctx context.Context) error {
	tx, err := qu.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx,
		`DELETE FROM jobs WHERE started_at IS NOT NULL AND attempts >= ? RETURNING id, kind, payload`, MaxAttempts)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var kind, payload string
		if err := rows.Scan(&id, &kind, &payload); err != nil {
			rows.Close()
			return err
		}
		slog.WarnContext(ctx, "dropped a background job that was interrupted too often",
			"id", id, "kind", kind, "payload", payload, "attempts", MaxAttempts)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM jobs WHERE started_at IS NOT NULL AND dedup_key IS NOT NULL AND EXISTS (
		   SELECT 1 FROM jobs AS p WHERE p.kind = jobs.kind AND p.dedup_key = jobs.dedup_key AND p.started_at IS NULL)`,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jobs SET started_at = NULL WHERE started_at IS NOT NULL`); err != nil {
		return err
	}
	return tx.Commit()
}

// claim starts the oldest pending job of kind. ok is false when there is none.
func (qu *Queue) claim(ctx context.Context, kind Kind) (job Job, ok bool, err error) {
	var payload string
	err = qu.DB.QueryRowContext(ctx,
		`UPDATE jobs SET started_at = ?, attempts = attempts + 1
		 WHERE id = (SELECT id FROM jobs WHERE kind = ? AND started_at IS NULL ORDER BY id LIMIT 1)
		 RETURNING id, payload, attempts`,
		db.FormatTime(qu.Now()), kind,
	).Scan(&job.ID, &payload, &job.Attempts)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	job.Kind = kind
	job.Payload = json.RawMessage(payload)
	return job, true, nil
}

// Run starts workers goroutines that run the jobs of kind with handler until
// ctx is done. It returns at once.
func (qu *Queue) Run(ctx context.Context, kind Kind, workers int, handler Handler) {
	wake := qu.signal(kind)
	poll := qu.PollInterval
	if poll <= 0 {
		poll = DefaultPollInterval
	}
	for range workers {
		go func() {
			ticker := time.NewTicker(poll)
			defer ticker.Stop()
			for {
				job, ok, err := qu.claim(ctx, kind)
				if err != nil && ctx.Err() == nil {
					slog.ErrorContext(ctx, "claiming a background job failed", "kind", kind, "error", err)
				}
				if ok {
					// More jobs may be waiting: let another worker look too.
					qu.Notify(kind)
					qu.run(ctx, job, handler)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-wake:
				case <-ticker.C:
				}
			}
		}()
	}
}

// run runs one job and deletes it, unless the server is stopping: then the
// job stays started, and Recover queues it again on the next start.
func (qu *Queue) run(ctx context.Context, job Job, handler Handler) {
	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic: %v", r)
			}
		}()
		return handler(ctx, job)
	}()
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		slog.WarnContext(ctx, "background job failed", "id", job.ID, "kind", job.Kind, "error", err)
	}
	if _, err := qu.DB.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, job.ID); err != nil {
		slog.ErrorContext(ctx, "deleting a finished background job failed", "id", job.ID, "error", err)
	}
}
