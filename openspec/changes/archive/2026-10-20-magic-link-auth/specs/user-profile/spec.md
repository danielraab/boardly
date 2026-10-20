# Spec Delta

## Purpose

The signed-in user's own profile page at `/profile`. It shows their account details and lets them edit their personal data, starting with an optional display name.

## ADDED Requirements

### Requirement: Profile page
`/profile` SHALL be available only to signed-in users. It shows a profile header with the account's email (read-only) and display name.

#### Scenario: Open profile
- **WHEN** a signed-in user opens `/profile`
- **THEN** the page shows their email and their display name, or an empty name field if none is set

### Requirement: Edit display name
The user SHALL be able to set, change and clear their display name from the profile header. The web app saves it with `PATCH /api/me`, and the new name shows up in the user menu right away.

#### Scenario: Set name
- **WHEN** the user enters "Ann Smith" and saves
- **THEN** the profile shows "Ann Smith" and the user menu shows "Ann Smith" without a reload

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
