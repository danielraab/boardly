# user-profile Specification

## Purpose

The signed-in user's settings page at `/settings`. Its tabs show their account details, let them edit their personal data (starting with an optional display name) and manage their passkeys.
## Requirements
### Requirement: Settings page
`/settings` SHALL be available only to signed-in users. It shows the heading "Settings" / "Einstellungen" and a tab bar. The "Profile" / "Profil" tab shows the account's email (read-only), display name and user slug.

#### Scenario: Open settings
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and shows their email, their display name (or an empty name field if none is set) and their user slug

### Requirement: Edit display name
The user SHALL be able to set, change and clear their display name in the "Profile" tab of the settings page. The web app saves it with `PATCH /api/me`, and the new name shows up in the user menu right away.

#### Scenario: Set name
- **WHEN** the user enters "Ann Smith" and saves
- **THEN** the "Profile" tab shows "Ann Smith" and the user menu shows "Ann Smith" without a reload

#### Scenario: Clear name
- **WHEN** the user clears the name field and saves
- **THEN** the name is removed and the user menu shows only the email

### Requirement: Display name validation
`PATCH /api/me` with `{"name": ...}` SHALL trim the value. An empty string or `null` clears the name. A value longer than 100 characters after trimming is rejected with 400 and `error.code` `invalid_name`. On success it responds 200 with the updated user.

#### Scenario: Too long
- **WHEN** a client sends a name of 101 characters
- **THEN** the server responds 400 with `invalid_name` and the name is unchanged

#### Scenario: Whitespace only
- **WHEN** a client sends `{"name": "   "}`
- **THEN** the server responds 200 and the user's `name` is `null`

### Requirement: Settings tabs
The settings page SHALL show the tabs "Profile" / "Profil", "Passkeys" / "Passkeys", "Connected apps" / "Verbundene Apps" and "Import / Export" / "Import / Export", in this order. "Profile" is selected by default. The selected tab is reflected in the URL (`?tab=profile`, `?tab=passkeys`, `?tab=apps` or `?tab=data`), so it survives a reload; an unknown value selects "Profile".

#### Scenario: Default tab
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and its content is shown

#### Scenario: Passkeys tab after reload
- **WHEN** a signed-in user selects "Passkeys" and reloads the page
- **THEN** the URL contains `?tab=passkeys` and the "Passkeys" tab is still selected

#### Scenario: Connected apps tab after reload
- **WHEN** a signed-in user selects "Connected apps" and reloads the page
- **THEN** the URL contains `?tab=apps` and the "Connected apps" tab is still selected

#### Scenario: Import / Export tab after reload
- **WHEN** a signed-in user selects "Import / Export" and reloads the page
- **THEN** the URL contains `?tab=data` and the "Import / Export" tab is still selected

### Requirement: Passkeys tab content
The Passkeys tab SHALL list the user's passkeys with provider icon, display name (label, else provider name, else "Passkey"), creation date and last-used date, each with a delete action. With no passkeys, it shows a short explanation of passkeys instead.

#### Scenario: Empty list
- **WHEN** a user without passkeys opens the Passkeys tab
- **THEN** it explains what passkeys are and how to add one

### Requirement: Add passkey within the window
While the enrollment window is open, the Passkeys tab SHALL show an "Add passkey" action and a live countdown of the remaining time. Adding opens a dialog with the optional label field, followed by the browser prompt.

#### Scenario: Countdown
- **WHEN** a user opens the Passkeys tab 1 minute after a magic-link sign-in
- **THEN** it shows "Add passkey" and a countdown at about 4:00 that decreases every second

#### Scenario: Window ends while open
- **WHEN** the countdown reaches zero
- **THEN** the add action is disabled and the closed-window hint appears without a reload

### Requirement: Reopen the window by email
When no enrollment window is open, the Passkeys tab SHALL explain that adding passkeys requires a recent sign-in by email link, and offer a "Send sign-in link" action that sends a magic link to the user's own address.

#### Scenario: Passkey session
- **WHEN** a user signed in with a passkey opens the Passkeys tab
- **THEN** the add action is disabled, the hint is shown, and "Send sign-in link" sends a link to their email

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

