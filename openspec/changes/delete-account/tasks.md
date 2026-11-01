# Tasks

## 1. Data and configuration

- [x] 1.1 Add migration `backend/internal/db/migrations/0024_account_deletion.sql` with `users.delete_after`, `users.deletion_locale`, `users.deletion_reminder_sent_at` and an index on `delete_after`; verify the migration test in `internal/db` passes on a fresh and an existing database
- [x] 1.2 Add `AccountDeletionDays` to `internal/config` from `BOARDLY_ACCOUNT_DELETION_DAYS` (default 7, integer ≥ 0, error naming the variable otherwise); verify with `config_test.go` cases for unset, `0`, `14`, `-1` and `week`
- [x] 1.3 Read `created_at` in `auth.LookupSession`, add `DeletionWindow` (3 min), `Session.DeletionUntil` and `DeletionOpen` (only for `magic_link` sessions); verify with auth tests for a fresh email session, one older than 3 minutes and a passkey session

## 2. Scheduling and cancelling

- [x] 2.1 Add `auth.DeleteOtherSessions(ctx, q, userID, keepIDHash)` and `apptokens.RevokeAll(ctx, q, userID, now)` (active tokens only); verify with unit tests that the current session and already revoked/expired tokens are untouched
- [x] 2.2 Create `internal/accounts` with `Service.Schedule` (window 403 `deletion_window_closed`, email 400 `email_mismatch`, already scheduled 409 `deletion_scheduled`, locale fallback `en`; one transaction setting the columns, deleting other sessions, revoking tokens) and `Service.Cancel` (409 `deletion_not_scheduled`); verify with service tests for each error, the success path and that cancel leaves tokens revoked
- [x] 2.3 Expose `deletionScheduledAt` in the users model and add `deletionWindowUntil` to the `GET /api/me` response; verify in `httpapi/auth_test.go`
- [x] 2.4 Add `DELETE /api/me` and `POST /api/me/restore` in `internal/httpapi` (cookie session only); verify with httptest cases for success, the three error codes, other sessions returning 401 and app token requests returning 401 afterwards

## 3. Dashboards

- [x] 3.1 Return 404 from `GET /api/u/{userSlug}/{dashboardSlug}` for non-owners when the owner's deletion is scheduled; verify in `httpapi/dashboards_test.go` for a visitor, another user, the owner, and after cancelling

## 4. Reminder and final deletion

- [x] 4.1 Add `mail.RenderAccountDeletion` with `en`/`de` text and HTML templates (app title, deletion date, how to cancel, link to the web app, "no further emails"); verify with `mail_test.go` golden files for both locales and a custom app title
- [x] 4.2 Implement the reminder step of `Service.Sweep` (due within 24h, not sent yet, grace > 0; mark sent only on success); verify with tests using a fake clock and mailer for: 7-day grace (not before day 6, once on day 6), 1-day grace (on the next sweep), 0-day grace (none), send failure (retried), cancel before reminder (none), cancel and reschedule (new reminder)
- [x] 4.3 Implement the deletion step of `Service.Sweep`: per user transaction deleting `magic_link_tokens` by email, the user's `preview_fetch`/`preview_convert` jobs and the user row, then removing the collected image and attachment files; verify with a store test for a user with nested note categories, notes with attachments, dashboards with widgets, favourites with tags, uploaded images, media sources, passkeys and tokens — all gone, files removed, another user's data and favicons untouched, slug and email reusable, not-yet-due users kept
- [x] 4.4 Add `Service.RunSweep(ctx, time.Hour)` and start it from `main.go`; verify with `main_test.go` / a run of the server that the sweep starts and logs

## 5. Frontend

- [x] 5.1 Extend the `User` type with `deletionScheduledAt`, `deletionWindowUntil` and `accountDeletionDays`, and add `features/settings/accountDeletion.ts` (schedule and cancel mutations, shared cancelled-notice state, date formatting); move the countdown hook to `lib/countdown.ts`; verify with the TypeScript build (`make check`)
- [x] 5.2 Add `DeleteAccountSection.tsx` at the end of the Profile tab with the window-open (countdown), window-closed and scheduled states and the export link to `?tab=data`; verify in `DeleteAccount.test.tsx` for each state, the switch at zero without reload, and the export link
- [x] 5.3 Add the confirmation dialog in `DeleteAccountSection.tsx` (deletion date, other devices signed out, apps revoked permanently, `/<slug>/…` becomes free, public dashboards hidden, export link, email confirmation, server errors incl. `deletion_window_closed`); verify in `DeleteAccount.test.tsx` that the button is disabled until the email matches (case-insensitive) and confirming calls the API with the current locale
- [x] 5.4 Add `features/settings/DeletionBanner.tsx` under the global header (in `AppShell`) using the shared cancel hook that updates the cached user and shows the closable notice (tokens stay revoked, link to `/settings?tab=apps`, other devices sign in again); verify in `DeleteAccount.test.tsx` that the banner shows the date, cancel removes it and shows the notice, and no banner without a schedule
- [x] 5.5 Add all `deleteAccount.*` strings in `en.json` and `de.json`; verify the i18n key test passes

## 6. Docs and checks

- [x] 6.1 Document account deletion and `BOARDLY_ACCOUNT_DELETION_DAYS` in `README.md` (feature section and configuration table) and `.env.example`; verify by reading the rendered sections
- [x] 6.2 Run `make check` and `make test`; verify both pass
