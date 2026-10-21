# Spec Delta

## MODIFIED Requirements

### Requirement: Settings page
`/settings` SHALL be available only to signed-in users. It shows the heading "Settings" / "Einstellungen" and a tab bar. The "Profile" / "Profil" tab shows the account's email (read-only), display name and user slug.

#### Scenario: Open settings
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and shows their email, their display name (or an empty name field if none is set) and their user slug

## ADDED Requirements

### Requirement: User slug
Every user SHALL have a user slug, which is the first part of their dashboard addresses (`/<user-slug>/<dashboard-slug>`). It follows the slug format of the `dashboards` capability and is unique across all users. When an account is created, it gets a generated slug of 10 random characters from `a-z0-9`. If that slug is taken, `-2`, `-3`, … is appended until it is free. Generated slugs never form a reserved word. Accounts that existed before this change get a generated slug when the database is migrated. `GET /api/me` returns the slug as `slug`.

#### Scenario: New account
- **WHEN** a new user signs in for the first time
- **THEN** `GET /api/me` returns a `slug` of 10 characters from `a-z0-9`

#### Scenario: Existing account after upgrade
- **WHEN** the server with this change starts on a database with existing users
- **THEN** every user has a unique slug

### Requirement: Reserved user slugs
A user slug SHALL NOT be one of the reserved words: paths that the app uses or may use as its own first path segment (at least `api`, `assets`, `karakeep`, `login`, `logout`, `settings`, `favourites`, `dashboards`, `admin`, `help`, `about`). The server rejects them with 400 and `slug_reserved`. The web app checks the same list, shows a hint at the field and blocks saving. If a later version adds a fixed route that equals an existing user slug, the route takes precedence and that user has to choose a new slug.

#### Scenario: Reserved word in the form
- **WHEN** the user enters `login` as their slug
- **THEN** the field shows that this address is reserved and the save action is disabled

#### Scenario: Reserved word over the API
- **WHEN** a client sends `PATCH /api/me` with `{"slug": "API"}`
- **THEN** the server responds 400 with `slug_reserved` and the slug is unchanged

### Requirement: Edit user slug
The user SHALL be able to change their slug in the Profile tab. The web app saves it with `PATCH /api/me` and `{"slug": ...}`. The server lowercases it and checks the format (`invalid_slug`), the reserved words (`slug_reserved`) and uniqueness across users (409 `slug_taken`). `PATCH /api/me` accepts `name` and `slug` alone or together. A request with neither is rejected with 400 `invalid_request`. While the field differs from the saved slug, the tab shows a hint that links using the old address stop working. Saving then asks for confirmation, showing an example of the old and the new address. Old addresses are not redirected.

#### Scenario: Change slug
- **WHEN** Ann changes her slug from `ann` to `annsmith`, saves and confirms
- **THEN** her dashboards are at `/annsmith/…` and `/ann/home` shows not found

#### Scenario: Slug taken
- **WHEN** Ann tries to save the slug `bob`, which Bob uses
- **THEN** the field shows that the slug is taken and her slug is unchanged

#### Scenario: Cancel confirmation
- **WHEN** Ann changes her slug, saves and cancels the confirmation
- **THEN** nothing is saved and the field still shows her edit
