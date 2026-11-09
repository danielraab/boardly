# Spec Delta

## Purpose

Lets users form groups of people with admin and member roles, so that items can be shared with several people at once.

## ADDED Requirements

### Requirement: Groups
A signed-in user SHALL be able to create any number of groups. A group has a name, which is trimmed and must be 1–100 characters long; otherwise the server responds 400 with `invalid_name`. Names don't have to be unique. The creator becomes the group's first group admin. A group is visible only to its members: there is no list of all groups, and for anyone else every request about the group responds as if it didn't exist.

#### Scenario: Create a group
- **WHEN** Ann creates a group named "Family"
- **THEN** the group exists with Ann as its only member and group admin

#### Scenario: Same name twice
- **WHEN** Ann and Bob each create a group named "Family"
- **THEN** both groups are created

#### Scenario: Not a member
- **WHEN** Bob isn't a member of "Family" and requests it by its ID
- **THEN** the server responds 404 with `not_found`

### Requirement: Membership roles
Each membership SHALL have the role group admin (`admin`) or group member (`member`). A group admin can rename and delete the group, invite people, remove members and change members' roles. A group member can see the group and its members and can leave the group. A person joins only by accepting an invitation sent to their email address (see the `invitations` capability), which also sets their role. Member lists show only names and slugs, never email addresses.

#### Scenario: Member tries to invite someone
- **WHEN** Bob is a group member of "Family" and tries to invite an address
- **THEN** the server responds 403 with `forbidden` and no invitation is created

#### Scenario: Promote a member
- **WHEN** Ann, a group admin, makes Bob a group admin
- **THEN** Bob can invite people and remove members

#### Scenario: No email addresses
- **WHEN** Bob requests the members of "Family"
- **THEN** the response contains no email address

### Requirement: Every group has an admin
A group with members SHALL always have at least one group admin. The last group admin can neither leave, be removed nor be given the role group member; the server responds 409 with `last_admin`. The last group admin can still delete the group.

#### Scenario: Last admin leaves
- **WHEN** Ann is the only group admin of "Family" and tries to leave
- **THEN** the server responds 409 with `last_admin` and Ann stays a group admin

#### Scenario: Last admin demoted
- **WHEN** Ann is the only group admin of "Family" and sets Ann's own role to group member
- **THEN** the server responds 409 with `last_admin`

#### Scenario: Second admin
- **WHEN** Ann and Bob are group admins of "Family" and Ann leaves
- **THEN** Ann is no longer a member and Bob stays a group admin

### Requirement: Shares end with the membership
When someone leaves a group or is removed from it, the shares they made with that group SHALL be removed. Shares other members made with the group stay.

#### Scenario: Sharer leaves
- **WHEN** Bob shared the dashboard "Trips" with "Family" and leaves "Family"
- **THEN** "Trips" is no longer shared with "Family"

#### Scenario: Others' shares stay
- **WHEN** Ann and Bob each shared a dashboard with "Family" and Bob leaves
- **THEN** Ann's dashboard stays shared with "Family"

### Requirement: Groups of deleted accounts
When an account is deleted and it was the last group admin of a group that has other members, the member who joined the group first SHALL become a group admin. A group that has no members left is deleted.

#### Scenario: Next member takes over
- **WHEN** Ann is the only group admin of "Family", Bob joined before Carol, and Ann's account is deleted
- **THEN** Bob is a group admin and Carol stays a group member

#### Scenario: Last member deleted
- **WHEN** Ann is the only member of "Family" and Ann's account is deleted
- **THEN** "Family" no longer exists

### Requirement: Groups API
The API SHALL provide, for signed-in users only:
- `GET /api/groups`: the groups the user belongs to, each with `id`, `name`, `role` (the user's own) and `memberCount`, sorted by name, ignoring case.
- `POST /api/groups`: create a group from `name`. Responds 201 with the group.
- `GET /api/groups/{id}`: the group and its `members`, each with `id`, `slug`, `name` and `role`, sorted by name, ignoring case. For group admins it also has `invitations`, the open invitations as described in the `invitations` capability.
- `PATCH /api/groups/{id}`: change `name`. Admins only.
- `DELETE /api/groups/{id}`: delete the group. Admins only. Responds 204.
- `PATCH /api/groups/{id}/members/{userId}`: change `role`. Admins only.
- `DELETE /api/groups/{id}/members/{userId}`: remove a member. Admins only, except that anyone can remove themselves to leave. Responds 204.

The errors are:
- 404 `not_found` for a group the user doesn't belong to, or an unknown member;
- 403 `forbidden` for a group member using an admin action;
- 400 `invalid_role` for any other role;
- 409 `last_admin` as described above.

Requests without a session get 401.

#### Scenario: List own groups
- **WHEN** Ann belongs to "Family" and "Work" and requests `GET /api/groups`
- **THEN** the response lists both, with Ann's role and the member count of each

#### Scenario: Change a role
- **WHEN** Ann, a group admin, sends `PATCH /api/groups/{id}/members/{userId}` with the role `admin` for Bob
- **THEN** Bob is a group admin

#### Scenario: Leave
- **WHEN** Bob, a group member, leaves "Family"
- **THEN** the response is 204 and Bob is no longer a member

### Requirement: Groups page
`/groups` SHALL be available only to signed-in users; signed-out visitors are sent to `/login`. It shows the heading "Groups" / "Gruppen", a create action, and the user's groups with their name, member count and the user's role, shown as "Group admin" / "Gruppen-Admin" or "Group member" / "Gruppenmitglied". Without groups, the page shows an empty state with a create action. Opening a group shows its members with name, slug and role. Group admins additionally get rename and delete actions, a field to invite an email address with a role, the open invitations, and a role choice and a remove action for each member. Everyone gets a "Leave group" / "Gruppe verlassen" action. Leaving and deleting ask for confirmation. Server errors, such as `last_admin`, are shown in the dialog that caused them.

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/groups`
- **THEN** the browser shows `/login`

#### Scenario: Group member's view
- **WHEN** Bob, a group member, opens "Family"
- **THEN** Bob sees the members and "Leave group", and no actions to invite, remove or rename

#### Scenario: Group admin's view
- **WHEN** Ann, a group admin, opens "Family"
- **THEN** Ann sees the actions to rename and delete the group, to invite people and to remove members

#### Scenario: Last admin leaves
- **WHEN** Ann is the only group admin and confirms "Leave group"
- **THEN** the page says that the group needs another group admin first and Ann stays in the group

#### Scenario: Empty
- **WHEN** a user without groups opens `/groups`
- **THEN** the page shows the empty state with a create action
