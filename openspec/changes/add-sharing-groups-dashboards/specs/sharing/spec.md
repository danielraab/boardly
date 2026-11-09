# Spec Delta

## Purpose

Lets the owner of a dashboard give specific users and groups access to it, and shows each user what others have shared with them.

## ADDED Requirements

### Requirement: Shares
The owner of a dashboard SHALL be able to share it with a group they belong to, and with a person who accepted an invitation (see the `invitations` capability), at the level `read`. No share means no access. A dashboard has at most one share per user and one per group. A share belongs to the dashboard's owner, and only the owner can see or remove the dashboard's shares. Deleting the dashboard, the user or the group removes the share.

#### Scenario: Share through an accepted invitation
- **WHEN** Bob accepts Ann's invitation to the dashboard "Work"
- **THEN** Bob can open `/ann/work`

#### Scenario: Remove a share
- **WHEN** Ann removes the share of "Work" with Bob
- **THEN** Bob gets the not-found page at `/ann/work`

#### Scenario: Deleted group
- **WHEN** Ann shared "Work" with the group "Family" and the group is deleted
- **THEN** the share no longer exists and the group's former members who have no other access get the not-found page

### Requirement: Effective access
A viewer's access to a dashboard SHALL be full when they own it. Otherwise it SHALL be the highest level among all shares that match the viewer: the share with the viewer and the shares with every group the viewer belongs to. Shares only add access, so no share or group can take away access another one gives. A viewer with no matching share and a dashboard that isn't public has no access, and every response is the same as for a dashboard that doesn't exist.

#### Scenario: Access through a group
- **WHEN** Ann shared "Work" with "Family" and Bob is a member of "Family"
- **THEN** Bob can open `/ann/work`

#### Scenario: Access removed with the membership
- **WHEN** Bob has access to "Work" only through "Family" and is removed from "Family"
- **THEN** Bob gets the not-found page at `/ann/work`

#### Scenario: Two sources of access
- **WHEN** Ann shared "Work" with Bob and with "Family", Bob is a member of "Family", and Ann removes the share with Bob
- **THEN** Bob can still open `/ann/work` through "Family"

### Requirement: Viewers of a shared dashboard
A person a dashboard is shared with SHALL see it as a visitor of a public dashboard does: they get the same fields, and widgets with "Show to other viewers" off are left out. They can't edit the dashboard, its layout or its shares, and `GET /api/dashboards` doesn't list it for them.

#### Scenario: Widget that isn't shown to others
- **WHEN** Ann shared "Work" with Bob and a note widget on it has "Show to other viewers" off
- **THEN** Bob doesn't get that widget

#### Scenario: No management
- **WHEN** Ann shared "Work" with Bob and Bob sends `PATCH /api/dashboards/{id}` with its ID
- **THEN** the server responds 404 with `not_found` and "Work" is unchanged

### Requirement: Shares of accounts scheduled for deletion
While the owner's account is scheduled for deletion, their dashboards SHALL NOT be visible through shares, and they SHALL NOT appear on the "Shared with me" page of others. When the deletion is cancelled, the shares work again.

#### Scenario: Owner scheduled for deletion
- **WHEN** Ann shared "Work" with Bob and Ann's account is scheduled for deletion
- **THEN** Bob gets the not-found page at `/ann/work`

#### Scenario: Deletion cancelled
- **WHEN** Ann cancels the scheduled deletion
- **THEN** Bob can open `/ann/work` again

### Requirement: Shares API
The API SHALL provide, for signed-in users only and for dashboards they own:
- `GET /api/shares/dashboard/{id}`: responds with `shares`, each with its `id`, its `level` and either a `user` (`id`, `slug`, `name`) or a `group` (`id`, `name`), and with `invitations`, the open invitations as described in the `invitations` capability. Users are sorted before groups, each by name, ignoring case.
- `POST /api/shares/dashboard/{id}`: share with a group, from `groupId` and `level`. Responds 201 with the share.
- `DELETE /api/shares/dashboard/{id}/{shareId}`: remove a share. Responds 204.

The errors are:
- 404 `not_found` for a dashboard the user doesn't own, an unknown share or an unknown resource type in the path;
- 400 `invalid_share_level` for any level other than `read`;
- 404 `group_not_found` when the group doesn't exist or the owner isn't a member of it;
- 409 `already_shared` when the dashboard is already shared with that group.

Requests without a session get 401.

#### Scenario: Share with a group and list
- **WHEN** Ann, a member of "Family", sends `POST /api/shares/dashboard/{id}` with its `groupId` and the level `read`, and then lists the shares
- **THEN** the first response is 201 and the list contains a share with "Family" and the level `read`

#### Scenario: No email address of a person with access
- **WHEN** Ann lists the shares of a dashboard shared with Bob
- **THEN** the entry for Bob doesn't contain Bob's email address

#### Scenario: Level edit is not available yet
- **WHEN** a client sends the level `edit`
- **THEN** the server responds 400 with `invalid_share_level`

#### Scenario: Group the owner isn't in
- **WHEN** Ann sends the ID of a group Ann isn't a member of
- **THEN** the server responds 404 with `group_not_found`

#### Scenario: Viewer can't manage shares
- **WHEN** Ann shared "Work" with Bob and Bob requests `GET /api/shares/dashboard/{id}` for it
- **THEN** the server responds 404 with `not_found`

#### Scenario: Duplicate share
- **WHEN** Ann shares "Work" with "Family" a second time
- **THEN** the server responds 409 with `already_shared`

### Requirement: Share dialog
The share action SHALL open a dialog titled "Share <title>" / "<title> teilen" for the dashboard, available only to its owner. It SHALL contain:
- a "Public" / "Öffentlich" switch with the existing hint, which saves right away;
- a field to invite someone by email address, with an explanation that registered users get a link to accept;
- a choice of the groups the owner belongs to, with a link to `/groups` when they belong to none;
- the list of people and groups with access, each showing "Can view" / "Kann ansehen" and a remove action, and the open invitations as described in the `invitations` capability;
- a hint, shown when the list contains a group or a group is selected: "Group admins decide who is in a group. Anyone they add gets access." / "Gruppen-Admins entscheiden, wer in einer Gruppe ist. Alle, die sie hinzufügen, erhalten Zugriff."

Server errors are shown in the dialog. There is no way to search for or look up people.

#### Scenario: Invite someone
- **WHEN** Ann opens the share dialog of "Work", enters `bob@example.org` and invites that address
- **THEN** `bob@example.org` appears in the list with "Invited", whether or not an account exists for it

#### Scenario: Make public
- **WHEN** Ann turns on "Public" in the dialog
- **THEN** the dashboard is public without any further save action

#### Scenario: Group hint
- **WHEN** Ann selects the group "Family" in the dialog
- **THEN** the hint about group admins is shown

#### Scenario: Remove access
- **WHEN** Ann removes Bob in the dialog
- **THEN** Bob disappears from the list and no longer has that share

### Requirement: Shared with me page
`/shared` SHALL be available only to signed-in users; signed-out visitors are sent to `/login`. It shows the heading "Shared with me" / "Mit mir geteilt" and lists the dashboards shared with the user, directly or through a group, each once, with its title linking to `/<owner-slug>/<dashboard-slug>` and the owner's name, or their slug when they have no name. The list is sorted by title, ignoring case, then by owner slug. It doesn't list the user's own dashboards or dashboards that are only public. Above the list, the page shows the user's open invitations (see the `invitations` capability), each with who sent it, what it is for and an "Accept" / "Annehmen" action. After accepting, the invitation disappears and a dashboard appears in the list. Without invitations and dashboards, the page shows an empty state.

The page loads `GET /api/shared-with-me`, which responds with `dashboards`, each with `ownerSlug`, `ownerName`, `slug`, `title` and `level`. Requests without a session get 401.

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/shared`
- **THEN** the browser shows `/login`

#### Scenario: Direct and group share
- **WHEN** Bob accepted an invitation to "Work" and Ann also shared it with a group Bob belongs to
- **THEN** Bob's page lists "Work" once, with Ann's name and the link `/ann/work`

#### Scenario: Own dashboard shared with a group
- **WHEN** Ann shared "Work" with a group Ann belongs to
- **THEN** Ann's page doesn't list "Work"

#### Scenario: Public dashboard
- **WHEN** Ann's dashboard "Home" is public but isn't shared with Bob
- **THEN** Bob's page doesn't list "Home"

#### Scenario: Open invitation
- **WHEN** Bob has an open invitation from Ann to "Work" and opens the page
- **THEN** the page shows the invitation with Ann's name and "Work", and "Work" isn't in the list until Bob activates "Accept"

#### Scenario: Accept on the page
- **WHEN** Bob activates "Accept" for the invitation to "Work"
- **THEN** the invitation disappears and "Work" is in the list

#### Scenario: Empty
- **WHEN** nothing is shared with Bob and Bob has no open invitations
- **THEN** the page shows the empty state
