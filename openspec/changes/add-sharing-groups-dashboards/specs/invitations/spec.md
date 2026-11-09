# Spec Delta

## Purpose

Lets a user invite someone by email address to a shared item or a group, without ever revealing to the inviter whether that address has an account.

## ADDED Requirements

### Requirement: Invitations
The owner of a dashboard SHALL be able to invite an email address to it at the level `read`, and a group admin SHALL be able to invite an email address to the group with the role `admin` or `member`. Addresses are trimmed and lowercased as for sign-in; an invalid one responds 400 with `invalid_email`. An invitation expires 14 days after it was created and can be accepted once. There is at most one open invitation per address and dashboard or group; a second one responds 409 with `already_invited`. Inviting one's own address responds 400 with `cannot_invite_self`.

#### Scenario: Invite to a dashboard
- **WHEN** Ann invites `bob@example.org` to "Work" at the level `read`
- **THEN** the response is 201 and `bob@example.org` is listed as invited

#### Scenario: Invite to a group
- **WHEN** Ann, a group admin, invites `bob@example.org` to "Family" with the role `member`
- **THEN** the response is 201 and `bob@example.org` is listed as invited

#### Scenario: Already invited
- **WHEN** `bob@example.org` has an open invitation to "Work" and Ann invites that address again
- **THEN** the server responds 409 with `already_invited`

### Requirement: Same result for every address
Creating an invitation SHALL give the same result whether or not an account exists for the address, and whether or not that account already has access. The response, its status, its timing and the way the invitation is listed SHALL NOT differ. The email is sent in the background after the response. Nothing else in the app SHALL show the inviter whether an address has an account until the invitation is accepted.

#### Scenario: Known and unknown address
- **WHEN** Ann invites an address that has an account and one that hasn't
- **THEN** both responses have the status 201 and the same fields, and both addresses are listed as invited

#### Scenario: Address that already has access
- **WHEN** Ann invites the address of someone who already has access to "Work"
- **THEN** the response is 201 and the address is listed as invited, as for any other address

#### Scenario: Notice not sent because of the limit
- **WHEN** an address without an account already received a notice in the last 24 hours and Ann invites it
- **THEN** the response is 201 and the address is listed as invited, as for any other address

### Requirement: Invitation email to registered users
When an account exists for the invited address, the server SHALL send it an email in the language the inviter was using. It names the inviter by their name, or their slug when they have no name, says what is being shared (the dashboard's title or the group's name) and contains a link to `/invitation` with a single-use token. Only a hash of the token is stored. The email doesn't contain the inviter's email address.

#### Scenario: Email to a registered user
- **WHEN** Ann invites the address of Bob's account to "Work"
- **THEN** Bob gets an email that names Ann and "Work" and has a link to accept

### Requirement: Notice to unregistered addresses
When no account exists for the invited address, the server SHALL send it a notice that somebody tried to share something with this address on the app. The notice SHALL NOT contain the inviter's name or address, the title of what is being shared, the group's name, or a link to accept. When registration is open, it says that the recipient can sign up with this address within 14 days to see the invitation. When registration is closed, it doesn't suggest signing up. An address gets at most one notice per 24 hours; invitations whose notice was held back by this limit can be accepted all the same. The invitation stays open, and the recipient finds it in the app after signing up with that address (see "Open invitations in the app").

#### Scenario: Notice
- **WHEN** Ann invites `new@example.org`, which has no account
- **THEN** that address gets a notice that doesn't name Ann or the dashboard and has no accept link

#### Scenario: Closed registration
- **WHEN** registration is closed and Ann invites an address without an account
- **THEN** the notice doesn't suggest creating an account

#### Scenario: Only one notice a day
- **WHEN** Ann and Carol each invite `new@example.org` on the same day
- **THEN** that address gets one notice

### Requirement: Open invitations in the app
`GET /api/invitations` SHALL list, for the signed-in user, the open invitations sent to their account's address, including those from before the account existed. Each has its `id`, `type` (`dashboard` or `group`), `title`, `inviterName`, `level` or `role`, and `expiresAt`, newest first. Expired invitations and dashboards whose owner's account is scheduled for deletion are left out. Requests without a session get 401. Having an account doesn't change how an invitation is shown to the inviter: it stays "Invited" until it is accepted.

#### Scenario: Registering after a notice
- **WHEN** `new@example.org` got a notice about Ann's invitation to "Work" and signs up with that address within 14 days
- **THEN** the new account's list contains the invitation with Ann's name and "Work", and it has no access to "Work" until it accepts

#### Scenario: Registered user without the email
- **WHEN** Bob has an open invitation and lost the invitation email
- **THEN** Bob's list contains the invitation and Bob can accept it

#### Scenario: Other people's invitations
- **WHEN** Carol requests the list and has no invitations
- **THEN** the list is empty, even though others have open invitations

#### Scenario: Signing up too late
- **WHEN** `new@example.org` signs up 15 days after the invitation was created
- **THEN** the list doesn't contain it

### Requirement: Limit on invitations
Each user SHALL be able to create at most 30 invitations per hour. Further ones respond 429 with `rate_limited`, whatever the address is.

#### Scenario: Too many invitations
- **WHEN** Ann has created 30 invitations in the last hour and creates another one
- **THEN** the server responds 429 with `rate_limited` and no email is sent

### Requirement: Accepting an invitation
An invitation SHALL be accepted only by a signed-in user whose account has the invited address, either with the token from the email or from the list of open invitations. Nobody gets access or joins a group without accepting, also not by signing up. Accepting a dashboard invitation creates a share with that user at the invitation's level. Accepting a group invitation makes the user a member with the invitation's role. When the user already has that share or membership, it stays as it is and accepting still succeeds. An accepted invitation is used up and no longer listed. A dashboard whose owner's account is scheduled for deletion can't be accepted and answers like an unknown invitation.

#### Scenario: Accept a dashboard invitation
- **WHEN** Bob, signed in with the invited address, accepts the invitation to "Work"
- **THEN** Ann's dashboard is shared with Bob and appears on Bob's "Shared with me" page

#### Scenario: Accept a group invitation
- **WHEN** Bob, signed in with the invited address, accepts the invitation to "Family"
- **THEN** Bob is a member of "Family" with the role from the invitation

#### Scenario: Another account
- **WHEN** Carol opens Bob's invitation link while signed in with another address
- **THEN** the server responds 403 with `invitation_wrong_account` and nothing changes

#### Scenario: Accept from the list
- **WHEN** Bob accepts an invitation by its ID from the list of open invitations
- **THEN** Ann's dashboard is shared with Bob and the link in the email answers `invitation_not_found`

#### Scenario: Someone else's invitation ID
- **WHEN** Carol sends the ID of an invitation sent to Bob's address
- **THEN** the server responds 404 with `invitation_not_found` and nothing changes

#### Scenario: Link used twice
- **WHEN** Bob accepted an invitation and opens the same link again
- **THEN** the server responds 404 with `invitation_not_found`

### Requirement: Invitations API
The API SHALL provide, for signed-in users only:
- `POST /api/shares/dashboard/{id}/invitations`: for the dashboard's owner, from `email`, `level` and `locale`. Responds 201 with `id`, `email`, `level`, `createdAt` and `expiresAt`.
- `DELETE /api/shares/dashboard/{id}/invitations/{invitationId}`: for the dashboard's owner. Responds 204.
- `POST /api/groups/{id}/invitations`: for group admins, from `email`, optional `role` (default `member`) and `locale`. Responds 201 with `id`, `email`, `role`, `createdAt` and `expiresAt`.
- `DELETE /api/groups/{id}/invitations/{invitationId}`: for group admins. Responds 204.
- `POST /api/invitations/preview`: from `token`. Responds with the `type` (`dashboard` or `group`), the `title`, the `inviterName` and the `level` or `role`.
- `GET /api/invitations`: the open invitations of the signed-in user's address, as described above.
- `POST /api/invitations/accept`: from exactly one of `token` and `invitationId`. Responds with the `type` and, for a dashboard, its `ownerSlug` and `slug`, or for a group, its `groupId`.

An unknown, expired or used token responds 404 with `invitation_not_found`, and a token for another account's address responds 403 with `invitation_wrong_account`. An `invitationId` that is unknown, expired, used or sent to another address responds 404 with `invitation_not_found`. Sending neither or both of `token` and `invitationId` responds 400 with `invalid_request`. Requests without a session get 401. Dashboards the user doesn't own and groups they don't belong to respond 404 with `not_found`; a group member who isn't an admin gets 403 with `forbidden`.

#### Scenario: Invalid level
- **WHEN** a client invites an address with the level `edit`
- **THEN** the server responds 400 with `invalid_share_level`

#### Scenario: Revoke
- **WHEN** Ann revokes the open invitation of `bob@example.org` to "Work"
- **THEN** the response is 204, the address is no longer listed and its link answers `invitation_not_found`

#### Scenario: Viewer can't invite
- **WHEN** Ann shared "Work" with Bob and Bob invites an address to it
- **THEN** the server responds 404 with `not_found`

### Requirement: Invited addresses are shown as invited
The share dialog SHALL list the dashboard's open invitations with the email address and the status "Invited" / "Eingeladen". The group page SHALL show them the same way, to group admins only. Each has a revoke action. After it is accepted, the person is shown by name and slug, without the email address. An expired invitation is no longer listed.

#### Scenario: Listed in the share dialog
- **WHEN** Ann invited `bob@example.org` to "Work" and opens its share dialog
- **THEN** the dialog shows `bob@example.org` with "Invited"

#### Scenario: Group member's view
- **WHEN** Carol, a group member who isn't an admin, opens "Family", which has an open invitation
- **THEN** Carol doesn't see the invited address

#### Scenario: After accepting
- **WHEN** Bob accepted the invitation and Ann opens the share dialog
- **THEN** the dialog shows Bob's name and slug and not the email address

### Requirement: Invitation page
`/invitation?token=…` SHALL send signed-out visitors to `/login` and return them to the invitation after signing in. For signed-in users it shows who invited them and to what, and an "Accept" / "Annehmen" action. After accepting, the browser shows the dashboard or the group. When the link is unknown, expired or used, the page says the invitation is no longer valid. When the account has another address, the page says to sign in with the address the invitation was sent to, without showing that address. Leaving the page without accepting changes nothing.

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens an invitation link
- **THEN** the browser shows `/login`, and after signing in the invitation

#### Scenario: Accept
- **WHEN** Bob opens the link and activates "Accept" for "Work"
- **THEN** the browser shows `/ann/work`

#### Scenario: Expired
- **WHEN** Bob opens the link 15 days after it was sent
- **THEN** the page says the invitation is no longer valid

### Requirement: Removing invitations
Expired invitations SHALL be deleted within a day after they expire. Deleting the dashboard, the group or the inviter's account deletes its invitations, and deleting an account also deletes the invitations sent to its address. A group invitation stays open when the admin who created it leaves the group; the remaining admins can revoke it.

#### Scenario: Expired invitation deleted
- **WHEN** an invitation has been expired for more than a day
- **THEN** the stored address no longer exists

#### Scenario: Account deleted
- **WHEN** Bob's account is deleted and has an open invitation
- **THEN** the invitation and its stored address no longer exist
