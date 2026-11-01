# Design

## Context

- Almost every user-owned table references `users (id)` with `ON DELETE CASCADE`: sessions, passkeys, WebAuthn challenges, app tokens, dashboards (→ widgets), favourite nodes (→ node tags), tags, notes (→ note attachments), note categories, attachments, media sources and images with a `user_id`. `note_attachments → attachments` is `DEFERRABLE INITIALLY DEFERRED` precisely so a user delete is checked only at commit. `note_categories` references its parent with no `ON DELETE` action; SQLite checks that at the end of the statement, so deleting the whole tree in one cascade is fine — this needs a test with nested categories.
- Not covered by the cascade: `magic_link_tokens` (keyed by `email`, with `request_ip`), `jobs` (`preview_fetch`/`preview_convert` payloads carry `userId` in JSON), and the files under `<data dir>/images` and `<data dir>/attachments`. Favicons (`images.user_id IS NULL`) are shared.
- Image and attachment files without a record are removed by the hourly cleanups (`blobs.RemoveOrphans`, `OrphanAge` 1h), and raw preview files that no job names by `images.removeUnusedRaw`.
- Sessions store `auth_method` and `created_at`; the passkey enrollment window is stored as `passkey_enrollment_until`. One-time-code sign-ins create `magic_link` sessions too.
- Background loops follow the `RunCleanup(ctx, interval)` pattern started from `main.go`.
- Mails are rendered from `internal/mail/templates/{en,de}` with a `Render…` function per mail (text + HTML) and the app title.

## Goals / Non-Goals

**Goals:**
- One place that owns the account deletion lifecycle (schedule, cancel, remind, delete).
- Final deletion is atomic in the database; files are removed right after, with the orphan cleanup as a safety net.
- The deletion window can't be bypassed by a stolen cookie of a passkey or older session.

**Non-Goals:**
- Data export (separate, upcoming change; only linked here).
- Admin-initiated deletion, retired slugs, or a notice when a deleted account's addresses are taken by someone else.
- A mail at scheduling time or after the deletion.

## Decisions

### Store the schedule on `users`
Migration `0024_account_deletion.sql` adds three nullable columns: `delete_after` (RFC 3339 text), `deletion_locale` (text) and `deletion_reminder_sent_at` (text), plus an index on `delete_after`. Scheduling sets `delete_after` and `deletion_locale` and clears `deletion_reminder_sent_at`; cancelling clears all three. *Alternative:* a separate `account_deletions` table — more joins for no gain since there's at most one schedule per user.

### Deletion window computed from the session
`auth.DeletionWindow = 3 * time.Minute`. `LookupSession` also reads `created_at`; `Session.DeletionUntil()` returns `created_at + 3m` for `magic_link` sessions and nil otherwise, and `DeletionOpen(now)` mirrors `EnrollmentOpen`. No new column: the window is fixed relative to sign-in, unlike the enrollment window that is persisted. *Alternative:* store a `deletion_window_until` column like the enrollment window — unnecessary since nothing ever reopens it.

### New package `internal/accounts`
A `Service` with `Schedule(ctx, sess, email, locale)`, `Cancel(ctx, userID)`, `Sweep(ctx)` and `RunSweep(ctx, interval)`. It depends on `*sql.DB`, the grace period, a clock, the mailer, the base URL/app title, and the image and attachment file dirs. Keeping this out of `users` avoids making the users store depend on mail, auth and file storage.

- **Schedule** runs in one transaction: check the window (403), compare the email (`strings.EqualFold` after `TrimSpace`; 400), check not already scheduled (409), set the columns, `DELETE FROM sessions WHERE user_id = ? AND id_hash <> ?`, and `UPDATE app_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)` (only active tokens, keeping the "revoking is final" semantics). The token update is a helper in `apptokens` (`RevokeAll(ctx, q, userID, now)`), and the session delete a helper in `auth` (`DeleteOtherSessions`), both taking a `db.Querier` so they join the transaction.
- **Cancel** clears the columns if `delete_after IS NOT NULL`, otherwise 409.

### Sweep: reminders then deletions
`RunSweep` runs at start and every hour (like the cleanups).
1. **Reminders:** `SELECT … WHERE delete_after IS NOT NULL AND deletion_reminder_sent_at IS NULL AND delete_after > now AND delete_after <= now + 24h AND <grace period > 0>`. For each, render and send the mail; on success set `deletion_reminder_sent_at`. On failure, log and leave it for the next sweep. With grace `0` the reminder is skipped by checking the configured days, so the condition is `days > 0`.
2. **Deletions:** for each user with `delete_after <= now`, in one transaction: collect the IDs of their images and attachments, `DELETE FROM magic_link_tokens WHERE email = ?` (emails are stored normalised; compare in lower case), `DELETE FROM jobs WHERE kind IN ('preview_fetch','preview_convert') AND json_extract(payload, '$.userId') = ?`, then `DELETE FROM users WHERE id = ?`. After commit, remove the collected files via `blobs.Dir.Remove`, logging errors (orphan cleanup catches leftovers). Each user is handled separately so one failure doesn't block others.

The sweep re-checks `delete_after` inside each deletion transaction, so a cancel that races with the sweep either wins or loses cleanly. *Alternative:* using the jobs queue for deletions — the queue is kind-restricted by a CHECK constraint and built for fetches; a dedicated sweep is simpler.

A running preview job of a deleted user finds its node gone and stops, as it already does for deleted links.

### Hide public dashboards
In the dashboards query behind `GET /api/u/{userSlug}/{dashboardSlug}`, a viewer who isn't the owner gets 404 when the owner's `delete_after IS NOT NULL`. Single condition in the existing owner lookup.

### Config
`config.AccountDeletionDays int` from `BOARDLY_ACCOUNT_DELETION_DAYS`, default 7, `strconv.Atoi` and `>= 0`, else an error naming the variable (like the other variables). The grace period is `days * 24h`.

### API
- `GET /api/me` adds `deletionScheduledAt` (from `users.delete_after`), `deletionWindowUntil` (from the session) and `accountDeletionDays` (the configured grace period, so the dialog can show the date).
- `DELETE /api/me` body `{"email", "locale"}` → 200 with the same body as `GET /api/me`.
- `POST /api/me/restore` → 200 with the same body as `GET /api/me`.
Both are cookie-session endpoints behind `requireUser`; app tokens can't reach them.

### Reminder mail
`mail.RenderAccountDeletion(locale, to, data)` with templates `account_deletion.{txt,html}` in `en` and `de`. Data: app title, deletion date formatted for the locale (UTC, date and time), and the base URL. The text says no further emails will be sent. Same plain-text + HTML structure as the sign-in mail.

### Frontend
- `features/settings/DeleteAccountSection.tsx` at the end of the Profile tab: three states from `GET /api/me` (window open with countdown, window closed, scheduled). The countdown re-uses the passkey tab's approach (ticks every second, switches state at zero without a reload).
- The confirmation dialog (in `DeleteAccountSection.tsx`): Base UI dialog with the listed explanations, the user's slug, the export link, and an email field that enables the destructive button.
- `features/settings/DeletionBanner.tsx` rendered by the app shell under the header when `deletionScheduledAt` is set; "Cancel deletion" calls restore, updates the cached `me` and shows a closable notice with a link to `/settings?tab=apps`. The settings section uses the same cancel hook and notice.
- Strings under `deleteAccount.*` in `en.json` / `de.json`.

## Risks / Trade-offs

- [The 3-minute window is short; the user has to sign out and sign in by email again] → The closed state says exactly what to do; the window starts at sign-in, so they can go straight to Settings.
- [A freed slug lets another user publish content under old public links] → Accepted by the user; the dialog says so. Public dashboards are hidden during the grace period, so old links 404 first.
- [Mail delivery fails] → Retried every sweep until the deletion; deletion still happens on schedule since the user was informed in the dialog.
- [Server is down at the deletion time] → The first sweep after start deletes overdue accounts (`delete_after <= now`).
- [File removal fails after commit] → Logged; the hourly orphan cleanup removes files without a record after `OrphanAge`.
- [Nested note categories and deferred FKs fail the cascade] → Store test that deletes a user with nested categories, notes with attachments, dashboards with widgets, favourites with tags and images.

## Migration Plan

Additive migration with nullable columns; no backfill. Rollback = revert the binary; the extra columns are ignored by older code (scheduled deletions would then simply not run).
