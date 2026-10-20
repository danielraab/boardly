# Spec Delta

## ADDED Requirements

### Requirement: Profile tabs
Below the profile header, `/profile` SHALL show a tab bar. The only tab for now is "Passkeys" / "Passkeys", and it's selected by default. The selected tab is reflected in the URL (`?tab=passkeys`), so it survives a reload.

#### Scenario: Default tab
- **WHEN** a signed-in user opens `/profile`
- **THEN** the "Passkeys" tab is selected and its content is shown

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
