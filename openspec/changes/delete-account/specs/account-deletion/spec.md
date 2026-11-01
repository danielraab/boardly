# Spec Delta

## Purpose

Lets signed-in users delete their own account and all data that belongs to it: scheduling the deletion from the settings, a configurable grace period in which it can be cancelled, a reminder email, and the final removal of the data.

## ADDED Requirements

### Requirement: Deletion window
A session SHALL have a deletion window that ends 3 minutes after the session was created, if it was created by a sign-in by email (magic link or one-time code). A session created by passkey SHALL never have a deletion window. `GET /api/me` SHALL include `deletionWindowUntil`: the end of the window as an RFC 3339 timestamp while it is open, otherwise `null`.

#### Scenario: Fresh email sign-in
- **WHEN** a user signed in by email link 1 minute ago requests `GET /api/me`
- **THEN** `deletionWindowUntil` is about 2 minutes in the future

#### Scenario: Window over
- **WHEN** a user signed in by email link 4 minutes ago requests `GET /api/me`
- **THEN** `deletionWindowUntil` is `null`

#### Scenario: Passkey session
- **WHEN** a user signed in with a passkey requests `GET /api/me`
- **THEN** `deletionWindowUntil` is `null`

### Requirement: Grace period setting
The grace period SHALL be set in whole days by `BOARDLY_ACCOUNT_DELETION_DAYS`, default `7`. `0` means the account is deleted at the next run of the deletion sweep. A value that isn't a non-negative integer SHALL stop the server from starting with an error that names the variable. `GET /api/me` SHALL include the setting as `accountDeletionDays`, so the web app can show the deletion date before scheduling.

#### Scenario: Default
- **WHEN** the server runs without `BOARDLY_ACCOUNT_DELETION_DAYS` and a user schedules their deletion on 2026-11-01 at 10:00 UTC
- **THEN** the deletion is scheduled for 2026-11-08 at 10:00 UTC

#### Scenario: Setting in the current user
- **WHEN** the server runs with `BOARDLY_ACCOUNT_DELETION_DAYS=14` and a signed-in user requests `GET /api/me`
- **THEN** the response contains `accountDeletionDays` 14

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_ACCOUNT_DELETION_DAYS=-1` or `=week`
- **THEN** it exits with an error naming `BOARDLY_ACCOUNT_DELETION_DAYS`

### Requirement: Schedule the deletion
`DELETE /api/me` with `{"email": "<address>", "locale": "en" | "de"}` SHALL schedule the deletion of the signed-in user's account for now plus the grace period and respond 200 with the updated user. `GET /api/me` SHALL include `deletionScheduledAt`, the time the account will be deleted (RFC 3339) or `null`. The request SHALL be rejected, with nothing changed:
- with 403 and `deletion_window_closed` when the session's deletion window isn't open;
- with 400 and `email_mismatch` when `email` doesn't equal the user's email address, ignoring case and surrounding whitespace;
- with 409 and `deletion_scheduled` when a deletion is already scheduled.

An unsupported or missing `locale` SHALL be stored as `en`.

When the deletion is scheduled, the server SHALL, in the same step, delete all other sessions of the user, keep the current session, and revoke all active app tokens of the user.

#### Scenario: Schedule
- **WHEN** Ann, signed in by email link 1 minute ago, sends `DELETE /api/me` with her email address
- **THEN** the server responds 200 and `deletionScheduledAt` is 7 days from now

#### Scenario: Window closed
- **WHEN** Ann, signed in with a passkey, sends `DELETE /api/me` with her email address
- **THEN** the server responds 403 with `deletion_window_closed` and no deletion is scheduled

#### Scenario: Wrong email
- **WHEN** Ann sends `DELETE /api/me` within the window with `bob@example.org`
- **THEN** the server responds 400 with `email_mismatch` and no deletion is scheduled

#### Scenario: Other devices signed out
- **WHEN** Ann is signed in on her laptop and her phone and schedules the deletion on the laptop
- **THEN** the laptop session still works and the next request from the phone responds 401

#### Scenario: Tokens revoked
- **WHEN** Ann has the active app tokens "Phone" and "Browser" and schedules the deletion
- **THEN** both tokens have the status `revoked`, and requests with their secrets are rejected with 401

### Requirement: Account stays usable during the grace period
Until the deletion is carried out, the account SHALL work as before: the user can sign in, edit their data and create new app tokens. The only differences are the revoked tokens and ended sessions from scheduling, and that their public dashboards are hidden from others (see the `dashboards` capability).

#### Scenario: Edit during grace period
- **WHEN** Ann's deletion is scheduled and she adds a favourite
- **THEN** the favourite is saved

#### Scenario: Sign in during grace period
- **WHEN** Ann's deletion is scheduled and she signs in on another device by email link
- **THEN** she is signed in to her account and her deletion is still scheduled

### Requirement: Cancel the deletion
`POST /api/me/restore` SHALL cancel the scheduled deletion of the signed-in user and respond 200 with the updated user, with `deletionScheduledAt` `null`. It does not require the deletion window. Without a scheduled deletion it SHALL respond 409 with `deletion_not_scheduled`. Cancelling SHALL NOT restore revoked app tokens or ended sessions. A later deletion can be scheduled again as usual.

#### Scenario: Cancel
- **WHEN** Ann, signed in with a passkey, cancels her scheduled deletion
- **THEN** the server responds 200, `deletionScheduledAt` is `null` and her account isn't deleted

#### Scenario: Tokens stay revoked
- **WHEN** Ann cancels her deletion
- **THEN** the tokens revoked when she scheduled it are still `revoked`

#### Scenario: Nothing scheduled
- **WHEN** a user without a scheduled deletion sends `POST /api/me/restore`
- **THEN** the server responds 409 with `deletion_not_scheduled`

### Requirement: Delete account section
The Profile tab of `/settings` SHALL end with a section "Delete account" / "Konto löschen". It SHALL say that deleting the account removes all of the user's dashboards, favourites, notes, uploaded files, passkeys and connected apps, and that this can't be undone after the grace period. It SHALL show a hint to export the data first, with a link to the "Import / Export" tab (`/settings?tab=data`).

- While the deletion window is open, it SHALL show a "Delete account…" action and the time left. When the window ends while the page is open, the action is replaced by the hint below without a reload.
- While the window is closed, it SHALL only say that deleting the account requires signing in again by email link, and that the deletion must then be confirmed within 3 minutes. It offers no action.
- While a deletion is scheduled, it SHALL show the deletion date and a "Cancel deletion" action instead.

All text SHALL be translated (English and German).

#### Scenario: Window open
- **WHEN** a user signed in by email link 1 minute ago opens the Profile tab
- **THEN** the section shows "Delete account…", the time left and the export hint

#### Scenario: Window closed
- **WHEN** a user signed in with a passkey opens the Profile tab
- **THEN** the section says a new sign-in by email link is needed and shows no delete action

#### Scenario: Export hint
- **WHEN** the user follows the export link in the section
- **THEN** the "Import / Export" tab is selected

### Requirement: Confirmation dialog
"Delete account…" SHALL open a dialog that states:
- the date on which the account will be deleted, and that the deletion can be cancelled until then;
- that all other devices are signed out right away;
- that all connected apps lose access right away and stay revoked even if the deletion is cancelled;
- that the user's address (`/<user-slug>/…`, with their actual slug) becomes available to other users after the deletion, and that their public dashboards are hidden from others right away;
- the hint to export the data first, with the link to the "Import / Export" tab.

The user SHALL type their email address to enable the delete action; the comparison ignores case and surrounding whitespace. Confirming sends `DELETE /api/me` with the web app's current language as `locale`. Server errors are shown in the dialog; on `deletion_window_closed` the dialog says that a new sign-in by email link is needed.

#### Scenario: Confirm
- **WHEN** Ann types her email address and confirms
- **THEN** the deletion is scheduled, the dialog closes and the banner appears

#### Scenario: Email not typed
- **WHEN** the email field is empty or doesn't match
- **THEN** the delete action is disabled

#### Scenario: Slug shown
- **WHEN** Ann with the slug `ann` opens the dialog
- **THEN** it says that the address `/ann/…` becomes available to others after the deletion

### Requirement: Deletion banner
While the signed-in user's deletion is scheduled, every page of the web app that shows the global header SHALL show a banner below it saying that the account will be deleted on the deletion date, with a "Cancel deletion" / "Löschung abbrechen" action. All text SHALL be translated (English and German).

#### Scenario: Banner
- **WHEN** Ann's deletion is scheduled for 8 November 2026 and she opens `/favourites`
- **THEN** a banner says that her account will be deleted on 8 November 2026 and offers "Cancel deletion"

#### Scenario: No deletion scheduled
- **WHEN** a user without a scheduled deletion opens any page
- **THEN** no deletion banner is shown

### Requirement: Hint after cancelling
After cancelling a deletion from the banner or the settings, the web app SHALL remove the banner and show a notice that the deletion was cancelled, that connected apps stay revoked and new tokens must be created, with a link to the "Connected apps" tab (`/settings?tab=apps`), and that other devices need to sign in again. The notice can be closed. All text SHALL be translated (English and German).

#### Scenario: Cancel from the banner
- **WHEN** Ann selects "Cancel deletion" in the banner
- **THEN** the banner disappears and a notice says that her connected apps stay revoked, with a link to "Connected apps"

### Requirement: Reminder email
The server SHALL send one reminder email to the user 1 day before the scheduled deletion. If the deletion is scheduled less than 1 day ahead (but later than now), the reminder SHALL be sent at the next run of the deletion sweep. With a grace period of `0` no reminder is sent. The email SHALL be in the locale given when scheduling, use the configured app title, state the deletion date, say that signing in and choosing "Cancel deletion" stops it, include a link to the web app, and say that no further emails will be sent. It SHALL be sent as plain text with an HTML alternative with the same content. Cancelling and scheduling again SHALL allow a new reminder for the new date. A reminder that fails to send SHALL be retried at the next sweep until the deletion is carried out.

#### Scenario: Reminder
- **WHEN** Ann's deletion is scheduled for 8 November 2026 at 10:00 UTC with locale `de`
- **THEN** on 7 November 2026 around 10:00 UTC she receives one German email naming 8 November 2026, explaining how to cancel and saying that no further emails will be sent

#### Scenario: Cancelled before the reminder
- **WHEN** Ann cancels her deletion 3 days after scheduling it
- **THEN** no reminder email is sent

#### Scenario: Short grace period
- **WHEN** the grace period is `1` day and Ann schedules her deletion
- **THEN** the reminder is sent at the next run of the deletion sweep

### Requirement: Carry out the deletion
The server SHALL check at least once per hour for accounts whose deletion time has passed and delete each of them completely: the account and all data that belongs to it (sessions, passkeys, app tokens, dashboards and their widgets, favourites and tags, notes, note categories, attachments, media sources and uploaded images, including their files), the sign-in links and codes issued to its email address, and its pending background jobs. Shared data that doesn't belong to a user, such as favicons, SHALL be kept. Afterwards the email address can register again (if registration is open) and the user slug can be taken by any user. No email is sent on deletion. Deleting the last remaining user is allowed.

#### Scenario: Account deleted
- **WHEN** Ann's deletion time has passed and the sweep runs
- **THEN** her account, dashboards, favourites, notes, files and tokens no longer exist, and her session responds 401

#### Scenario: Slug free again
- **WHEN** Ann's account with the slug `ann` was deleted and Bob changes his slug to `ann`
- **THEN** the change succeeds

#### Scenario: Other users untouched
- **WHEN** Ann's account is deleted
- **THEN** Bob's data and the shared favicons are unchanged

#### Scenario: Not yet due
- **WHEN** the sweep runs before Ann's deletion time
- **THEN** her account still exists
