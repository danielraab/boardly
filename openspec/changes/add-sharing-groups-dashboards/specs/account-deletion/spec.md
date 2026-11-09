# Spec Delta

## MODIFIED Requirements

### Requirement: Account stays usable during the grace period
Until the deletion is carried out, the account SHALL work as before: the user can sign in, edit their data and create new app tokens. The only differences are the revoked tokens and ended sessions from scheduling, and that their public and shared dashboards are hidden from others (see the `dashboards` capability).

#### Scenario: Edit during grace period
- **WHEN** Ann's deletion is scheduled and she adds a favourite
- **THEN** the favourite is saved

#### Scenario: Sign in during grace period
- **WHEN** Ann's deletion is scheduled and she signs in on another device by email link
- **THEN** she is signed in to her account and her deletion is still scheduled

### Requirement: Confirmation dialog
"Delete account…" SHALL open a dialog that states:
- the date on which the account will be deleted, and that the deletion can be cancelled until then;
- that all other devices are signed out right away;
- that all connected apps lose access right away and stay revoked even if the deletion is cancelled;
- that the user's address (`/<user-slug>/…`, with their actual slug) becomes available to other users after the deletion, and that their public and shared dashboards are hidden from others right away;
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

### Requirement: Carry out the deletion
The server SHALL check at least once per hour for accounts whose deletion time has passed and delete each of them completely: the account and all data that belongs to it (sessions, passkeys, app tokens, dashboards and their widgets, favourites and tags, notes, note categories, attachments, media sources, feed subscriptions, the flags on feed items, uploaded images, including their files, the shares the user made and received, their group memberships, the invitations they created and the invitations sent to their address), the sign-in links and codes issued to its email address, and its pending background jobs. Shared data that doesn't belong to a user, such as favicons and feeds and their items, SHALL be kept, apart from the feed items that only this user had imported (see the `data-import` capability). Feeds and items nobody uses any more are removed by the item cleanup of the `feeds` capability. Groups the user was a member of are handled as the `groups` capability describes. Afterwards the email address can register again (if registration is open) and the user slug can be taken by any user. No email is sent on deletion. Deleting the last remaining user is allowed.

#### Scenario: Account deleted
- **WHEN** Ann's deletion time has passed and the sweep runs
- **THEN** her account, dashboards, favourites, notes, files and tokens no longer exist, and her session responds 401

#### Scenario: Slug free again
- **WHEN** Ann's account with the slug `ann` was deleted and Bob changes his slug to `ann`
- **THEN** the change succeeds

#### Scenario: Other users untouched
- **WHEN** Ann's account is deleted
- **THEN** Bob's data, his feeds and their items, and the shared favicons are unchanged

#### Scenario: Not yet due
- **WHEN** the sweep runs before Ann's deletion time
- **THEN** her account still exists

#### Scenario: Shares removed
- **WHEN** Ann shared "Work" with Bob and Bob shared "Trips" with Ann, and Ann's account is deleted
- **THEN** "Work" and its share no longer exist, and Bob's "Trips" is no longer shared with Ann
