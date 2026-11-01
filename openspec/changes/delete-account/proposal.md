# Proposal

## Why

Users can't leave Boardly: there is no way to delete an account and the data that belongs to it, short of an operator editing the database. Users should be able to delete their own account from the settings, with a short grace period that protects against mistakes and a hijacked session.

## What Changes

- New **"Delete account"** section at the end of the Profile tab of `/settings`. It lists what gets deleted, hints at exporting the data first (link to the "Import / Export" tab) and says that the user's address `/<user-slug>/…` becomes free for others once the account is gone.
- **Deletion window**: scheduling a deletion requires a session that was signed in by email link (magic link or one-time code) within the last **3 minutes**. Outside the window, the section only says that a new sign-in by email link is needed. There is no special link or redirect for this.
- **Confirmation dialog**: the user types their email address. The dialog states the deletion date, that other devices are signed out and connected apps lose access right away, and that the slug becomes free.
- New `DELETE /api/me` schedules the deletion; new `POST /api/me/restore` cancels it. `GET /api/me` returns `deletionScheduledAt`, `deletionWindowUntil` and `accountDeletionDays`.
- **Grace period** set by the new env var `BOARDLY_ACCOUNT_DELETION_DAYS` (default `7`, `0` = delete on the next sweep). When a deletion is scheduled:
  - every other session of the user is deleted, the current one is kept;
  - all active app tokens are revoked;
  - the user's public dashboards are hidden from everyone else (not found);
  - the account otherwise stays fully usable, and every signed-in page shows a banner with the deletion date and a "Cancel deletion" action.
- **Cancelling** clears the schedule. Revoked tokens and ended sessions stay that way; a hint tells the user to create new tokens in "Connected apps". Public dashboards are visible again.
- **Reminder email** sent once, 1 day before the deletion (right away if the deletion is less than a day away, never with `0` days), in the locale used when scheduling. It says when the account is deleted, how to cancel, and that no further emails will be sent.
- **Final deletion** by an hourly background sweep: the user row (cascading to sessions, passkeys, app tokens, dashboards, favourites, tags, notes, categories, attachments, media sources, uploaded images), plus the user's sign-in tokens (stored by email), their pending background jobs and their image and attachment files. The email address and slug are free again afterwards.

## Capabilities

### New Capabilities

- `account-deletion`: scheduling, cancelling and carrying out the deletion of the signed-in user's account: deletion window, API, settings UI, banner, grace period, reminder email and what gets deleted.

### Modified Capabilities

- `dashboards`: public dashboards of an account scheduled for deletion are shown only to their owner.

## Impact

- Backend:
  - Migration: `users.delete_after`, `users.deletion_locale`, `users.deletion_reminder_sent_at`.
  - `internal/config`: `BOARDLY_ACCOUNT_DELETION_DAYS`.
  - `internal/auth`: deletion window from the session's method and creation time; deleting the user's other sessions.
  - `internal/users`: schedule, cancel, due/reminder queries, final deletion (incl. `magic_link_tokens`, `jobs`, files via `images` and `attachments` stores).
  - `internal/apptokens`: revoke all active tokens of a user.
  - `internal/dashboards`: hide public dashboards of scheduled users.
  - `internal/mail`: localized reminder email (en/de).
  - `internal/httpapi`: `DELETE /api/me`, `POST /api/me/restore`, new `GET /api/me` fields.
  - `main.go`: start the deletion sweep.
- Frontend: Profile tab section and dialog, global banner, cancel hint, `en.json`/`de.json`, tests.
- README: account deletion and `BOARDLY_ACCOUNT_DELETION_DAYS`.
- No new dependencies.
