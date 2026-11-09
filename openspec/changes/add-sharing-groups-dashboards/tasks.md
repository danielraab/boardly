# Tasks

## 1. Data model

- [ ] 1.1 Add migration `0028_sharing.sql` with `user_groups`, `user_group_members` and `shares` as in `design.md`, including the "exactly one" checks and the partial unique indexes for dashboards; verify with a `db` test that a share with two resources or two principals is rejected and that deleting a dashboard, user or group removes its shares
- [ ] 1.2 Add `user_groups`, `user_group_members` and `shares` to `notExportedTables` in `backend/internal/export/coverage_test.go`; verify `TestEveryUserTableIsCovered` passes
- [ ] 1.3 Add `shared` and `groups` to the reserved slugs in `slugs.go`, `slugs/testdata/reserved.json` and `frontend/src/features/dashboards/slug.ts`; verify the slug tests on both sides pass and `PATCH /api/me` with `groups` responds `slug_reserved`

## 2. Groups backend

- [ ] 2.1 Add `internal/groups` with create, list, get, rename and delete, validating names; verify with store tests, including that non-members get not found
- [ ] 2.2 Add membership operations: add by slug, change role, remove and leave, with role checks and the last-admin check in one transaction; verify with store tests for `forbidden`, `already_member`, `user_not_found` and `last_admin`
- [ ] 2.3 Remove a person's shares with the group when they leave or are removed, in the same transaction; verify with a test that other members' shares stay
- [ ] 2.4 Add the `/api/groups` routes and errors from the `groups` spec; verify with `httpapi` tests for every route, including 401 and that no response contains an email address
- [ ] 2.5 In `accounts.deleteAccount`, promote the earliest-joined member of groups where the account is the only admin and delete groups where it is the only member; verify with `accounts` tests for both cases

## 3. Shares backend

- [ ] 3.1 Add `internal/shares` with list, create and remove for dashboards, owner-only, setting `owner_id` from the dashboard; verify with store tests for each error in the `sharing` spec
- [ ] 3.2 Add the `/api/shares/{type}/{id}` routes, with only `dashboard` valid and only level `read` accepted; verify with `httpapi` tests, including that a viewer of a shared dashboard gets 404
- [ ] 3.3 Add `GET /api/users/{slug}` with exact, case-insensitive matching and a per-user limit of 60 per hour; verify with tests for partial slugs, accounts scheduled for deletion, 401, 429 and no email address
- [ ] 3.4 Add the "shared with me" query and `GET /api/shared-with-me`, listing each dashboard once and leaving out own dashboards and accounts scheduled for deletion; verify with tests for direct, group and combined shares

## 4. Dashboard visibility

- [ ] 4.1 Extend `dashboards.Store.Lookup` and the switcher query so a dashboard is visible through a share with the viewer or one of their groups; verify with tests that signed-out viewers, non-members and viewers of an owner scheduled for deletion get `ErrNotFound`, and that a shared viewer gets no `id`, `isDefault` or `isPublic`
- [ ] 4.2 Verify with an `httpapi` test that a widget with `showIfPublic` off is left out for a shared viewer, and that a shared viewer can't change the dashboard or its layout

## 5. API documentation

- [ ] 5.1 Document the new routes with their schemas and errors in `backend/internal/httpapi/openapi/boardly.yaml`; verify `TestOpenAPIDocumentsMatchRoutes` passes

## 6. Frontend: groups

- [ ] 6.1 Add the groups API client and the `/groups` route with list, create and empty state; verify with component tests, including the redirect to `/login`
- [ ] 6.2 Add the group details with members, and the admin actions, the add-by-address field, role choice, leave and delete with confirmation; verify with tests of the admin and member views and the `last_admin` message

## 7. Frontend: sharing

- [ ] 7.1 Add the share dialog in `features/sharing` with the "Public" switch, add by address, group choice, the access list and the group admin hint; verify with component tests for the scenarios in the `sharing` spec
- [ ] 7.2 Add the share action to `/dashboards` and "Share" to the dashboard actions menu; verify with tests that it is shown only to the owner
- [ ] 7.3 Remove "Public" from the dashboard dialog, and update its tests
- [ ] 7.4 Add the `/shared` route with list and empty state; verify with component tests, including the redirect to `/login`
- [ ] 7.5 Add "Shared with me" and "Groups" to the user menu and set the page widths; verify with the app shell tests

## 8. Texts

- [ ] 8.1 Change the label of all nine widget kinds to "Show to other viewers" / "Anderen Betrachtern zeigen", and reword the hints so they also fit people a dashboard is shared with, keeping the stored key `showIfPublic`; verify the affected widget tests pass
- [ ] 8.2 Add all new texts in en and de, and change the `deleteAccount.publicDashboards` text in en and de to name public and shared dashboards; verify the i18n checks pass
- [ ] 8.3 Update `README.md` (including "All data belongs to a single user") and add an entry to `CHANGELOG.md`; verify the new section describes shares and groups

## 9. Final checks

- [ ] 9.1 Run the backend checks and tests once (`cd backend && go vet ./... && test -z "$(gofmt -l .)" && go test ./...`) and verify they pass
- [ ] 9.2 Run the frontend checks and tests once, not alongside anything else (`cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test`), and verify they pass
- [ ] 9.3 Confirm in the running app that Ann can share a dashboard with Bob directly and through a group, that Bob sees it under "Shared with me", and that removing the share gives Bob the not-found page
