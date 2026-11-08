# Spec Delta

## MODIFIED Requirements

### Requirement: Carry out the deletion
The server SHALL check at least once per hour for accounts whose deletion time has passed and delete each of them completely: the account and all data that belongs to it (sessions, passkeys, app tokens, dashboards and their widgets, favourites and tags, notes, note categories, attachments, media sources, feed subscriptions, the flags on feed items and uploaded images, including their files), the sign-in links and codes issued to its email address, and its pending background jobs. Shared data that doesn't belong to a user, such as favicons and feeds and their items, SHALL be kept, apart from the feed items that only this user had imported (see the `data-import` capability). Feeds and items nobody uses any more are removed by the item cleanup of the `feeds` capability. Afterwards the email address can register again (if registration is open) and the user slug can be taken by any user. No email is sent on deletion. Deleting the last remaining user is allowed.

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
