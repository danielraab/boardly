# Spec Delta

## RENAMED Requirements

- FROM: `### Requirement: Profile page`
- TO: `### Requirement: Settings page`

## MODIFIED Requirements

### Requirement: Settings page
`/settings` SHALL be available only to signed-in users. It shows the heading "Settings" / "Einstellungen" and a tab bar. The "Profile" / "Profil" tab shows the account's email (read-only) and display name.

#### Scenario: Open settings
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and shows their email and their display name, or an empty name field if none is set

### Requirement: Edit display name
The user SHALL be able to set, change and clear their display name in the "Profile" tab of the settings page. The web app saves it with `PATCH /api/me`, and the new name shows up in the user menu right away.

#### Scenario: Set name
- **WHEN** the user enters "Ann Smith" and saves
- **THEN** the "Profile" tab shows "Ann Smith" and the user menu shows "Ann Smith" without a reload

#### Scenario: Clear name
- **WHEN** the user clears the name field and saves
- **THEN** the name is removed and the user menu shows only the email

## ADDED Requirements

### Requirement: Settings tabs
The settings page SHALL show the tabs "Profile" / "Profil" and "Passkeys" / "Passkeys", in this order. "Profile" is selected by default. The selected tab is reflected in the URL (`?tab=profile` or `?tab=passkeys`), so it survives a reload; an unknown value selects "Profile".

#### Scenario: Default tab
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and its content is shown

#### Scenario: Passkeys tab after reload
- **WHEN** a signed-in user selects "Passkeys" and reloads the page
- **THEN** the URL contains `?tab=passkeys` and the "Passkeys" tab is still selected

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
