# Tasks

## 1. Data model

- [x] 1.1 Add migration `0028_sharing.sql` with `user_groups`, `user_group_members`, `shares`, `invitations` and `invitation_notices` as in `design.md`, adding `invitation_mail` to the job kinds, including the "exactly one" checks and the partial unique indexes for dashboards and open invitations; verify with a `db` test that a share with two resources or two principals is rejected and that deleting a dashboard, user or group removes its shares and invitations
- [x] 1.2 Add `user_group_members`, `shares` and `invitations` to `notExportedTables` in `backend/internal/export/coverage_test.go` (`user_groups` and `invitation_notices` have no reference to a user, so the test doesn't count them as user data); verify `TestEveryUserTableIsCovered` passes
- [x] 1.3 Add `shared`, `groups` and `invitation` to the reserved slugs in `slugs.go`, `slugs/testdata/reserved.json` and `frontend/src/features/dashboards/slug.ts`; verify the slug tests on both sides pass and `PATCH /api/me` with `groups` responds `slug_reserved`

## 2. Groups backend

- [x] 2.1 Add `internal/groups` with create, list, get, rename and delete, validating names; verify with store tests, including that non-members get not found
- [x] 2.2 Add changing roles, removing and leaving, with role checks and the last-admin check in one transaction; verify with store tests for `forbidden` and `last_admin`
- [x] 2.3 Remove a person's shares with the group when they leave or are removed, in the same transaction; verify with a test that other members' shares stay
- [x] 2.4 Add the `/api/groups` routes and errors from the `groups` spec; verify with `httpapi` tests for every route, including 401 and that no member entry contains an email address
- [x] 2.5 In `accounts.deleteAccount`, promote the earliest-joined member of groups where the account is the only admin, delete groups where it is the only member, and delete invitations sent to its address; verify with `accounts` tests for each case

## 3. Shares backend

- [x] 3.1 Add `internal/shares` with list, group share and remove for dashboards, owner-only, setting `owner_id` from the dashboard; verify with store tests for each error in the `sharing` spec
- [x] 3.2 Add the `/api/shares/{type}/{id}` routes, with only `dashboard` valid and only level `read` accepted; verify with `httpapi` tests, including that a viewer of a shared dashboard gets 404
- [x] 3.3 Add the "shared with me" query and `GET /api/shared-with-me`, listing each dashboard once and leaving out own dashboards and accounts scheduled for deletion; verify with tests for direct, group and combined shares

## 4. Invitations backend

- [x] 4.1 Add `internal/invitations` with create and revoke for dashboards and groups, checking owner or admin, without reading `users`; verify with store tests for `invalid_email`, `already_invited` and `cannot_invite_self`
- [x] 4.2 Add the `invitation` and `invitation_notice` mail templates in en and de; verify with golden files, including that the notice has no inviter, title or link, names the 14 days when registration is open and doesn't suggest signing up when it is closed
- [x] 4.3 Add the `invitation_mail` job: generate the token and store its hash for an existing account, send the notice otherwise, at most one notice per address per 24 hours, also after the invitation was revoked; verify with tests for both paths, the limit, revoking and a retry
- [x] 4.4 Add accepting by token or by invitation ID in one transaction, checking the session's address and an owner scheduled for deletion, and leaving an existing share or membership as it is; verify with tests for dashboards, groups, `invitation_wrong_account`, another address's invitation ID and a used or expired token
- [x] 4.5 Add the invitation routes with the limit of 30 per user per hour; verify with `httpapi` tests that a known and an unknown address get identical responses and listings, and for 401, 403, 404 and 429
- [x] 4.6 Add the sweep for expired invitations; verify with a test that an invitation expired for more than a day is deleted and that a notice record older than 24 hours is removed
- [x] 4.7 Add `GET /api/invitations`, matching the session's address; verify with tests that an invitation made before the account existed is listed after signing up, that it gives no access until accepted, and that others' and expired invitations are left out

## 5. Dashboard visibility

- [x] 5.1 Extend `dashboards.Store.Lookup` and the switcher query so a dashboard is visible through a share with the viewer or one of their groups; verify with tests that signed-out viewers, non-members and viewers of an owner scheduled for deletion get `ErrNotFound`, and that a shared viewer gets no `id`, `isDefault` or `isPublic`
- [x] 5.2 Verify with an `httpapi` test that a widget with `showIfPublic` off is left out for a shared viewer, and that a shared viewer can't change the dashboard or its layout

## 6. API documentation

- [x] 6.1 Document the new routes with their schemas and errors in `backend/internal/httpapi/openapi/boardly.yaml`; verify `TestOpenAPIDocumentsMatchRoutes` passes

## 7. Frontend: groups

- [x] 7.1 Add the groups API client and the `/groups` route with list, create and empty state; verify with component tests, including the redirect to `/login`
- [x] 7.2 Add the group details with members, and the admin actions: invite by email with a role, open invitations with revoke, role choice, remove, leave and delete with confirmation; verify with tests of the admin and member views and the `last_admin` message

## 8. Frontend: sharing and invitations

- [x] 8.1 Add the share dialog in `features/sharing` with the "Public" switch, invite by email, group choice, the access list with invited addresses and the group admin hint; verify with component tests for the scenarios in the `sharing` and `invitations` specs
- [x] 8.2 Add the share action to `/dashboards` and "Share" to the dashboard actions menu; verify with tests that it is shown only to the owner
- [x] 8.3 Remove "Public" from the dashboard dialog, and update its tests
- [x] 8.4 Add the `/shared` route with open invitations and their accept action, the list and the empty state; verify with component tests, including the redirect to `/login`
- [x] 8.5 Add the `/invitation` route in `features/invitations` with preview, accept and the invalid and wrong-account messages; verify with component tests, including the redirect to `/login` and back
- [x] 8.6 Add "Shared with me" and "Groups" to the user menu and set the page widths; verify with the app shell tests

## 9. Texts

- [x] 9.1 Change the label of all nine widget kinds to "Show to other viewers" / "Anderen Betrachtern zeigen", and reword the hints so they also fit people a dashboard is shared with, keeping the stored key `showIfPublic`; verify the affected widget tests pass
- [x] 9.2 Add all new texts in en and de, and change the `deleteAccount.publicDashboards` text in en and de to name public and shared dashboards; verify the i18n checks pass
- [x] 9.3 Update `README.md` (including "All data belongs to a single user"), add an entry to `CHANGELOG.md`, and tell operators in the README's privacy policy section that invited addresses are stored for up to 15 days; verify the new sections describe shares, groups and invitations

## 10. Final checks

- [x] 10.1 Run the backend checks and tests once (`cd backend && go vet ./... && test -z "$(gofmt -l .)" && go test ./...`) and verify they pass
- [x] 10.2 Run the frontend checks and tests once, not alongside anything else (`cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test`), and verify they pass
- [x] 10.3 Confirm in the running app that Ann can invite Bob by email and through a group, that Bob sees the dashboard under "Shared with me" after accepting, that an unknown address looks the same to Ann and gets only the notice, that signing up with it shows the invitation under "Shared with me" to accept, and that removing the share gives Bob the not-found page
