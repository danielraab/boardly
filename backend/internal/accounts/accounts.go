// Package accounts schedules, cancels and carries out the deletion of user
// accounts (openspec change delete-account).
package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"boardly/internal/apptokens"
	"boardly/internal/auth"
	"boardly/internal/blobs"
	"boardly/internal/db"
	"boardly/internal/mail"
	"boardly/internal/users"
)

var (
	// ErrWindowClosed is returned when the session's deletion window isn't open.
	ErrWindowClosed = errors.New("deletion window closed")
	// ErrEmailMismatch is returned when the confirmation isn't the user's email address.
	ErrEmailMismatch = errors.New("email doesn't match")
	// ErrScheduled is returned when a deletion is already scheduled.
	ErrScheduled = errors.New("deletion already scheduled")
	// ErrNotScheduled is returned when cancelling without a scheduled deletion.
	ErrNotScheduled = errors.New("no deletion scheduled")
)

// ReminderBefore is how long before the deletion the reminder email is sent.
const ReminderBefore = 24 * time.Hour

// locales are the languages of the reminder email; others become "en".
var locales = map[string]bool{"en": true, "de": true}

// Service runs the account deletion lifecycle.
type Service struct {
	DB    *sql.DB
	Users *users.Store
	Now   func() time.Time
	// Days is the grace period between scheduling and deleting.
	Days   int
	Mailer mail.Sender
	// AppTitle names the product in the reminder email.
	AppTitle string
	// Link is the address of the web app the reminder email links to.
	Link string
	// ImageFiles and AttachmentFiles hold the files removed with an account.
	ImageFiles      blobs.Dir
	AttachmentFiles blobs.Dir
}

// Schedule schedules the deletion of the session's user. email must be the
// user's address (case and surrounding space are ignored). In the same
// transaction, the user's other sessions are deleted and their active app
// tokens revoked. Errors are ErrWindowClosed, ErrEmailMismatch and ErrScheduled.
func (s *Service) Schedule(ctx context.Context, sess auth.Session, email, locale string) (users.User, error) {
	now := s.Now()
	if !sess.DeletionOpen(now) {
		return users.User{}, ErrWindowClosed
	}
	if !locales[locale] {
		locale = "en"
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return users.User{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var address string
	var deleteAfter sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT email, delete_after FROM users WHERE id = ?`, sess.UserID).Scan(&address, &deleteAfter)
	if errors.Is(err, sql.ErrNoRows) {
		return users.User{}, users.ErrNotFound
	}
	if err != nil {
		return users.User{}, err
	}
	if !strings.EqualFold(strings.TrimSpace(email), address) {
		return users.User{}, ErrEmailMismatch
	}
	if deleteAfter.Valid {
		return users.User{}, ErrScheduled
	}
	at := now.Add(time.Duration(s.Days) * 24 * time.Hour)
	if _, err := tx.ExecContext(ctx,
		`UPDATE users SET delete_after = ?, deletion_locale = ?, deletion_reminder_sent_at = NULL WHERE id = ?`,
		db.FormatTime(at), locale, sess.UserID,
	); err != nil {
		return users.User{}, err
	}
	if err := auth.DeleteOtherSessions(ctx, tx, sess.UserID, sess.Secret); err != nil {
		return users.User{}, err
	}
	if err := apptokens.RevokeAll(ctx, tx, sess.UserID, now); err != nil {
		return users.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return users.User{}, err
	}
	return s.Users.Get(ctx, sess.UserID)
}

// Cancel cancels the user's scheduled deletion, or returns ErrNotScheduled.
// Revoked tokens and deleted sessions stay that way.
func (s *Service) Cancel(ctx context.Context, userID string) (users.User, error) {
	res, err := s.DB.ExecContext(ctx,
		`UPDATE users SET delete_after = NULL, deletion_locale = NULL, deletion_reminder_sent_at = NULL
		 WHERE id = ? AND delete_after IS NOT NULL`, userID)
	if err != nil {
		return users.User{}, err
	}
	if n, err := res.RowsAffected(); err != nil {
		return users.User{}, err
	} else if n == 0 {
		return users.User{}, ErrNotScheduled
	}
	return s.Users.Get(ctx, userID)
}

// Sweep sends the reminders that are due and deletes the accounts whose
// deletion time has passed. A failure for one account doesn't stop the
// others; the first error is returned.
func (s *Service) Sweep(ctx context.Context) (reminded, deleted int, err error) {
	reminded, err = s.remind(ctx)
	n, delErr := s.deleteDue(ctx)
	if err == nil {
		err = delErr
	}
	return reminded, n, err
}

type due struct {
	id, email, locale string
	at                time.Time
	atText            string
}

// remind sends the reminder to every account whose deletion is less than
// ReminderBefore away and that hasn't got one yet. A reminder that fails is
// tried again at the next sweep.
func (s *Service) remind(ctx context.Context) (int, error) {
	if s.Days == 0 {
		return 0, nil
	}
	now := s.Now()
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, email, COALESCE(deletion_locale, 'en'), delete_after FROM users
		 WHERE delete_after IS NOT NULL AND deletion_reminder_sent_at IS NULL
		   AND delete_after > ? AND delete_after <= ?`,
		db.FormatTime(now), db.FormatTime(now.Add(ReminderBefore)))
	if err != nil {
		return 0, err
	}
	var list []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.email, &d.locale, &d.atText); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if d.at, err = db.ParseTime(d.atText); err != nil {
			_ = rows.Close()
			return 0, err
		}
		list = append(list, d)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}

	var firstErr error
	sent := 0
	for _, d := range list {
		if err := s.sendReminder(ctx, d); err != nil {
			slog.ErrorContext(ctx, "sending the account deletion reminder failed", "user", d.id, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		// Only mark the reminder for the deletion it was sent for.
		if _, err := s.DB.ExecContext(ctx,
			`UPDATE users SET deletion_reminder_sent_at = ? WHERE id = ? AND delete_after = ?`,
			db.FormatTime(s.Now()), d.id, d.atText,
		); err != nil && firstErr == nil {
			firstErr = err
		}
		sent++
	}
	return sent, firstErr
}

func (s *Service) sendReminder(ctx context.Context, d due) error {
	m, err := mail.RenderAccountDeletion(d.locale, d.email, mail.AccountDeletionData{
		AppName: s.AppTitle,
		Date:    FormatDate(d.locale, d.at),
		Link:    s.Link,
	})
	if err != nil {
		return err
	}
	return s.Mailer.Send(ctx, m)
}

// deleteDue deletes every account whose deletion time has passed.
func (s *Service) deleteDue(ctx context.Context) (int, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id FROM users WHERE delete_after IS NOT NULL AND delete_after <= ?`, db.FormatTime(s.Now()))
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
	if err := rows.Close(); err != nil {
		return 0, err
	}

	var firstErr error
	deleted := 0
	for _, id := range ids {
		ok, err := s.deleteAccount(ctx, id)
		if err != nil {
			slog.ErrorContext(ctx, "deleting an account failed", "user", id, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if ok {
			deleted++
		}
	}
	return deleted, firstErr
}

// deleteAccount deletes the account if its deletion is still due, with
// everything that belongs to it, and then removes its files. It reports
// whether the account was deleted.
func (s *Service) deleteAccount(ctx context.Context, id string) (bool, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	// Check again: the deletion may have been cancelled since the list was read.
	var email string
	err = tx.QueryRowContext(ctx,
		`SELECT email FROM users WHERE id = ? AND delete_after IS NOT NULL AND delete_after <= ?`,
		id, db.FormatTime(s.Now()),
	).Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	images, err := ids(ctx, tx, `SELECT id FROM images WHERE user_id = ?`, id)
	if err != nil {
		return false, err
	}
	files, err := ids(ctx, tx, `SELECT id FROM attachments WHERE user_id = ?`, id)
	if err != nil {
		return false, err
	}
	for _, q := range []struct {
		query string
		arg   string
	}{
		{`DELETE FROM magic_link_tokens WHERE email = ?`, email},
		{`DELETE FROM jobs WHERE kind IN ('preview_fetch', 'preview_convert') AND json_extract(payload, '$.userId') = ?`, id},
		{`DELETE FROM users WHERE id = ?`, id},
	} {
		if _, err := tx.ExecContext(ctx, q.query, q.arg); err != nil {
			return false, fmt.Errorf("%s: %w", q.query, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}

	// Files left behind by a failure are removed by the orphan cleanups.
	for _, f := range []struct {
		dir blobs.Dir
		ids []string
	}{{s.ImageFiles, images}, {s.AttachmentFiles, files}} {
		if f.dir == "" {
			continue
		}
		for _, fileID := range f.ids {
			if err := f.dir.Remove(fileID); err != nil {
				slog.ErrorContext(ctx, "removing a file of a deleted account failed", "file", fileID, "error", err)
			}
		}
	}
	slog.InfoContext(ctx, "deleted account", "user", id)
	return true, nil
}

func ids(ctx context.Context, q db.Querier, query string, args ...any) ([]string, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		list = append(list, id)
	}
	return list, rows.Err()
}

// RunSweep sweeps now and then every interval until ctx is done.
func (s *Service) RunSweep(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if reminded, deleted, err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "account deletion sweep failed", "error", err)
		} else if reminded > 0 || deleted > 0 {
			slog.InfoContext(ctx, "account deletion sweep", "reminded", reminded, "deleted", deleted)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

var germanMonths = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli",
	"August", "September", "Oktober", "November", "Dezember"}

// FormatDate formats the deletion time in UTC for the email's locale, such as
// "8 November 2026, 10:00 UTC" or "8. November 2026, 10:00 UTC".
func FormatDate(locale string, t time.Time) string {
	t = t.UTC()
	if locale == "de" {
		return fmt.Sprintf("%d. %s %d, %s UTC", t.Day(), germanMonths[t.Month()-1], t.Year(), t.Format("15:04"))
	}
	return t.Format("2 January 2006, 15:04") + " UTC"
}
