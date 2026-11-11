// Package invitations invites email addresses to dashboards and groups. The
// inviter gets the same result for every address: the request never reads the
// users table, and a background job decides what to send.
package invitations

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/google/uuid"

	"boardly/internal/auth"
	"boardly/internal/db"
	"boardly/internal/groups"
	"boardly/internal/jobs"
	"boardly/internal/mail"
	"boardly/internal/shares"
)

// ValidDays is how long an invitation can be accepted.
const ValidDays = 14

// TTL is how long an invitation can be accepted.
const TTL = ValidDays * 24 * time.Hour

// NoticeInterval is the shortest time between two notices to an address.
const NoticeInterval = 24 * time.Hour

// Types of an invitation.
const (
	TypeDashboard = "dashboard"
	TypeGroup     = "group"
)

// Errors returned by the store.
var (
	// ErrNotFound means the dashboard isn't the user's, or the group doesn't
	// exist for them, or the invitation to revoke doesn't exist.
	ErrNotFound       = errors.New("not found")
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidLevel   = errors.New("invalid share level")
	ErrAlreadyInvited = errors.New("already invited")
	ErrCannotSelf     = errors.New("cannot invite own address")
	// ErrInvitationNotFound means the token or ID is unknown, expired or used.
	ErrInvitationNotFound = errors.New("invitation not found")
	// ErrWrongAccount means a valid token belongs to another address.
	ErrWrongAccount = errors.New("invitation is for another account")
)

// Actor is the signed-in user acting on an invitation.
type Actor struct {
	ID    string
	Email string
}

// Invitation is an open invitation as the inviter's side sees it. Whether the
// address has an account is deliberately not part of it.
type Invitation struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Level     string `json:"level,omitempty"`
	Role      string `json:"role,omitempty"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt"`
}

// Preview says who invited someone to what.
type Preview struct {
	Type string `json:"type"`
	// Title is the dashboard's title or the group's name.
	Title string `json:"title"`
	// InviterName is the inviter's name, or their slug when they have none.
	InviterName string `json:"inviterName"`
	Level       string `json:"level,omitempty"`
	Role        string `json:"role,omitempty"`
}

// Open is an open invitation as the invited person sees it.
type Open struct {
	ID string `json:"id"`
	Preview
	ExpiresAt string `json:"expiresAt"`
}

// Accepted says what an accepted invitation led to.
type Accepted struct {
	Type      string `json:"type"`
	OwnerSlug string `json:"ownerSlug,omitempty"`
	Slug      string `json:"slug,omitempty"`
	GroupID   string `json:"groupId,omitempty"`
}

// Store creates, delivers and accepts invitations.
type Store struct {
	DB   *sql.DB
	Now  func() time.Time
	Jobs *jobs.Queue
	// Mailer sends the invitation and the notice.
	Mailer mail.Sender
	// AppTitle names the product in the emails.
	AppTitle string
	// BaseURL is the address of the web app the invitation links to.
	BaseURL *url.URL
	// RegistrationClosed keeps the notice from suggesting to sign up.
	RegistrationClosed bool
}

var locales = map[string]bool{"en": true, "de": true}

// InviteToDashboard invites an address to one of the inviter's dashboards.
func (s *Store) InviteToDashboard(ctx context.Context, inviter Actor, dashboardID, rawEmail, level, locale string) (Invitation, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var one int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM dashboards WHERE id = ? AND user_id = ?`, dashboardID, inviter.ID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return Invitation{}, ErrNotFound
	}
	if err != nil {
		return Invitation{}, err
	}
	if level != shares.LevelRead {
		return Invitation{}, ErrInvalidLevel
	}
	return s.create(ctx, tx, inviter, "dashboard_id", dashboardID, rawEmail, level, "", locale)
}

// InviteToGroup invites an address to a group. Group admins only; errors for
// others come from groups.RequireAdmin. An empty role means group member.
func (s *Store) InviteToGroup(ctx context.Context, inviter Actor, groupID, rawEmail, role, locale string) (Invitation, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Invitation{}, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := groups.RequireAdmin(ctx, tx, groupID, inviter.ID); err != nil {
		return Invitation{}, err
	}
	if role == "" {
		role = groups.RoleMember
	}
	if !groups.ValidRole(role) {
		return Invitation{}, groups.ErrInvalidRole
	}
	return s.create(ctx, tx, inviter, "group_id", groupID, rawEmail, "", role, locale)
}

// create stores the invitation and queues its mail, and commits tx. It does
// the same for every address and never reads the users table. column is one of
// the two constant column names of the callers.
func (s *Store) create(ctx context.Context, tx *sql.Tx, inviter Actor, column, targetID, rawEmail, level, role, locale string) (Invitation, error) {
	email, err := auth.NormalizeEmail(rawEmail)
	if err != nil {
		return Invitation{}, ErrInvalidEmail
	}
	if email == inviter.Email {
		return Invitation{}, ErrCannotSelf
	}
	if !locales[locale] {
		locale = mail.DefaultLocale
	}
	now := s.Now()
	// An expired invitation that the sweep hasn't removed yet isn't open.
	if _, err := tx.ExecContext(ctx,
		`DELETE FROM invitations WHERE `+column+` = ? AND email = ? AND expires_at <= ?`,
		targetID, email, db.FormatTime(now)); err != nil {
		return Invitation{}, err
	}

	inv := Invitation{
		ID: uuid.NewString(), Email: email, Level: level, Role: role,
		CreatedAt: db.FormatTime(now), ExpiresAt: db.FormatTime(now.Add(TTL)),
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO invitations (id, inviter_id, email, locale, `+column+`, level, role, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?)`,
		inv.ID, inviter.ID, email, locale, targetID, level, role, inv.CreatedAt, inv.ExpiresAt)
	if db.IsUniqueViolation(err) {
		return Invitation{}, ErrAlreadyInvited
	}
	if err != nil {
		return Invitation{}, err
	}
	if err := s.Jobs.Enqueue(ctx, tx, jobs.KindInvitationMail, "", jobs.InvitationMailPayload{InvitationID: inv.ID}); err != nil {
		return Invitation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Invitation{}, err
	}
	s.Jobs.Notify(jobs.KindInvitationMail)
	return inv, nil
}

// RevokeDashboard deletes an open invitation to one of the owner's dashboards.
func (s *Store) RevokeDashboard(ctx context.Context, ownerID, dashboardID, invitationID string) error {
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM invitations WHERE id = ? AND dashboard_id = ?
		   AND EXISTS (SELECT 1 FROM dashboards WHERE id = ? AND user_id = ?)`,
		invitationID, dashboardID, dashboardID, ownerID)
	if err != nil {
		return err
	}
	return oneRow(res)
}

// RevokeGroup deletes an open invitation to a group. Group admins only.
func (s *Store) RevokeGroup(ctx context.Context, adminID, groupID, invitationID string) error {
	if err := groups.RequireAdmin(ctx, s.DB, groupID, adminID); err != nil {
		return err
	}
	res, err := s.DB.ExecContext(ctx, `DELETE FROM invitations WHERE id = ? AND group_id = ?`, invitationID, groupID)
	if err != nil {
		return err
	}
	return oneRow(res)
}

func oneRow(res sql.Result) error {
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListForDashboard returns the open invitations to a dashboard. The caller has
// checked that the user owns it.
func (s *Store) ListForDashboard(ctx context.Context, dashboardID string) ([]Invitation, error) {
	return s.listFor(ctx, "dashboard_id", dashboardID)
}

// ListForGroup returns the open invitations to a group. The caller has checked
// that the user is one of its group admins.
func (s *Store) ListForGroup(ctx context.Context, groupID string) ([]Invitation, error) {
	return s.listFor(ctx, "group_id", groupID)
}

func (s *Store) listFor(ctx context.Context, column, id string) ([]Invitation, error) {
	rows, err := s.DB.QueryContext(ctx,
		`SELECT id, email, COALESCE(level, ''), COALESCE(role, ''), created_at, expires_at FROM invitations
		 WHERE `+column+` = ? AND expires_at > ? ORDER BY email`, id, db.FormatTime(s.Now()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invitation{}
	for rows.Next() {
		var inv Invitation
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Level, &inv.Role, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// details is an open invitation with everything shown or needed about it.
type details struct {
	id, email, locale, expiresAt string
	dashboardID, groupID         sql.NullString
	ownerID, ownerSlug, slug     sql.NullString
	preview                      Preview
}

// openQuery selects open invitations, leaving out dashboards whose owner's
// account is scheduled for deletion. Its first argument is the current time.
const openQuery = `SELECT i.id, i.email, i.locale, i.expires_at, i.dashboard_id, i.group_id, d.user_id, o.slug, d.slug,
	  COALESCE(d.title, g.name), COALESCE(inviter.name, inviter.slug), COALESCE(i.level, ''), COALESCE(i.role, '')
	FROM invitations i
	JOIN users inviter ON inviter.id = i.inviter_id
	LEFT JOIN dashboards d ON d.id = i.dashboard_id
	LEFT JOIN users o ON o.id = d.user_id
	LEFT JOIN user_groups g ON g.id = i.group_id
	WHERE i.expires_at > ? AND (i.dashboard_id IS NULL OR o.delete_after IS NULL)
	  AND (i.dashboard_id IS NOT NULL OR i.group_id IS NOT NULL) `

func scanDetails(row interface{ Scan(...any) error }) (details, error) {
	var d details
	err := row.Scan(&d.id, &d.email, &d.locale, &d.expiresAt, &d.dashboardID, &d.groupID, &d.ownerID, &d.ownerSlug, &d.slug,
		&d.preview.Title, &d.preview.InviterName, &d.preview.Level, &d.preview.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return details{}, ErrInvitationNotFound
	}
	if err != nil {
		return details{}, err
	}
	d.preview.Type = TypeGroup
	if d.dashboardID.Valid {
		d.preview.Type = TypeDashboard
	}
	return d, nil
}

// ListOpen returns the open invitations sent to the address, newest first,
// also those from before the account existed.
func (s *Store) ListOpen(ctx context.Context, email string) ([]Open, error) {
	rows, err := s.DB.QueryContext(ctx, openQuery+`AND i.email = ? ORDER BY i.created_at DESC, i.id`,
		db.FormatTime(s.Now()), email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Open{}
	for rows.Next() {
		d, err := scanDetails(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, Open{ID: d.id, Preview: d.preview, ExpiresAt: d.expiresAt})
	}
	return out, rows.Err()
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// byToken returns the open invitation of a token for the account with email.
func byToken(ctx context.Context, q db.Querier, now, token, email string) (details, error) {
	d, err := scanDetails(q.QueryRowContext(ctx, openQuery+`AND i.token_hash = ?`, now, hashToken(token)))
	if err != nil {
		return details{}, err
	}
	if d.email != email {
		return details{}, ErrWrongAccount
	}
	return d, nil
}

// Preview says who invited the account to what. Errors are
// ErrInvitationNotFound and ErrWrongAccount, which keeps the details from
// anyone else holding the link.
func (s *Store) Preview(ctx context.Context, account Actor, token string) (Preview, error) {
	d, err := byToken(ctx, s.DB, db.FormatTime(s.Now()), token, account.Email)
	if err != nil {
		return Preview{}, err
	}
	return d.preview, nil
}

// Accept redeems an invitation by its token or, from the list of open
// invitations, by its ID. It creates the share or membership and uses up the
// invitation in one transaction; an existing share or membership stays as it
// is. An ID sent to another address is ErrInvitationNotFound.
func (s *Store) Accept(ctx context.Context, account Actor, token, invitationID string) (Accepted, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Accepted{}, err
	}
	defer func() { _ = tx.Rollback() }()

	now := s.Now()
	var d details
	if token != "" {
		d, err = byToken(ctx, tx, db.FormatTime(now), token, account.Email)
	} else {
		d, err = scanDetails(tx.QueryRowContext(ctx, openQuery+`AND i.id = ? AND i.email = ?`,
			db.FormatTime(now), invitationID, account.Email))
	}
	if err != nil {
		return Accepted{}, err
	}

	var result Accepted
	if d.preview.Type == TypeDashboard {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO shares (id, owner_id, dashboard_id, user_id, level, created_at)
			 VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			uuid.NewString(), d.ownerID.String, d.dashboardID.String, account.ID, d.preview.Level, db.FormatTime(now)); err != nil {
			return Accepted{}, err
		}
		result = Accepted{Type: TypeDashboard, OwnerSlug: d.ownerSlug.String, Slug: d.slug.String}
	} else {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO user_group_members (group_id, user_id, role, joined_at)
			 VALUES (?, ?, ?, ?) ON CONFLICT DO NOTHING`,
			d.groupID.String, account.ID, d.preview.Role, db.FormatTime(now)); err != nil {
			return Accepted{}, err
		}
		result = Accepted{Type: TypeGroup, GroupID: d.groupID.String}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM invitations WHERE id = ?`, d.id); err != nil {
		return Accepted{}, err
	}
	return result, tx.Commit()
}

// RunMailJob is the handler of the invitation_mail job.
func (s *Store) RunMailJob(ctx context.Context, job jobs.Job) error {
	var p jobs.InvitationMailPayload
	if err := json.Unmarshal(job.Payload, &p); err != nil {
		return err
	}
	return s.Deliver(ctx, p.InvitationID)
}

// Deliver sends the mail of an invitation: the link when the address has an
// account, otherwise a notice, at most one per address per NoticeInterval. An
// invitation that is gone by now sends nothing. The token is only created
// here, so it is never in the job's payload, and a job that runs again creates
// a new one.
func (s *Store) Deliver(ctx context.Context, invitationID string) error {
	now := s.Now()
	d, err := scanDetails(s.DB.QueryRowContext(ctx, openQuery+`AND i.id = ?`, db.FormatTime(now), invitationID))
	if errors.Is(err, ErrInvitationNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	var one int
	err = s.DB.QueryRowContext(ctx, `SELECT 1 FROM users WHERE email = ?`, d.email).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return s.sendNotice(ctx, d, now)
	}
	if err != nil {
		return err
	}
	return s.sendInvitation(ctx, d)
}

func (s *Store) sendInvitation(ctx context.Context, d details) error {
	token, err := newToken()
	if err != nil {
		return err
	}
	if _, err := s.DB.ExecContext(ctx, `UPDATE invitations SET token_hash = ? WHERE id = ?`, hashToken(token), d.id); err != nil {
		return err
	}
	link := s.BaseURL.JoinPath("invitation")
	link.RawQuery = url.Values{"token": {token}}.Encode()
	m, err := mail.RenderInvitation(d.locale, d.email, mail.InvitationData{
		AppName: s.AppTitle, InviterName: d.preview.InviterName, Title: d.preview.Title,
		IsGroup: d.preview.Type == TypeGroup, Link: link.String(), ValidDays: ValidDays,
	})
	if err != nil {
		return err
	}
	return s.Mailer.Send(ctx, m)
}

// sendNotice records the notice before sending it, in one statement, so two
// invitations can't both send one.
func (s *Store) sendNotice(ctx context.Context, d details, now time.Time) error {
	sentAt := db.FormatTime(now)
	res, err := s.DB.ExecContext(ctx,
		`INSERT INTO invitation_notices (email, sent_at) VALUES (?, ?)
		 ON CONFLICT (email) DO UPDATE SET sent_at = excluded.sent_at WHERE sent_at <= ?`,
		d.email, sentAt, db.FormatTime(now.Add(-NoticeInterval)))
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return nil
	}
	m, err := mail.RenderInvitationNotice(d.locale, d.email, mail.InvitationNoticeData{
		AppName: s.AppTitle, RegistrationOpen: !s.RegistrationClosed, ValidDays: ValidDays,
	})
	if err == nil {
		err = s.Mailer.Send(ctx, m)
	}
	if err != nil {
		// Nothing was sent, so a later invitation may try again.
		if _, delErr := s.DB.ExecContext(ctx,
			`DELETE FROM invitation_notices WHERE email = ? AND sent_at = ?`, d.email, sentAt); delErr != nil {
			slog.ErrorContext(ctx, "releasing an invitation notice failed", "error", delErr)
		}
	}
	return err
}

// Sweep deletes expired invitations and the records of notices that no longer
// limit anything.
func (s *Store) Sweep(ctx context.Context) error {
	now := s.Now()
	if _, err := s.DB.ExecContext(ctx, `DELETE FROM invitations WHERE expires_at <= ?`, db.FormatTime(now)); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `DELETE FROM invitation_notices WHERE sent_at <= ?`, db.FormatTime(now.Add(-NoticeInterval)))
	return err
}

// RunSweep sweeps now and then every interval until ctx is done.
func (s *Store) RunSweep(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Sweep(ctx); err != nil && ctx.Err() == nil {
			slog.ErrorContext(ctx, "invitation sweep failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// DeleteForAddress removes the invitations and the notice record of an
// address, in the caller's transaction, when its account is deleted.
func DeleteForAddress(ctx context.Context, tx db.Querier, email string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM invitations WHERE email = ?`, email); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM invitation_notices WHERE email = ?`, email)
	return err
}
