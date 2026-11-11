# Design

## Context

See `proposal.md` for the motivation and the `sharing` and `groups` specs for the behaviour.

Today a dashboard's visibility is decided in `dashboards.Store.Lookup`: the owner sees it, and anyone else only when `is_public` is set and the owner isn't scheduled for deletion. Widgets are resolved with `Viewer{OwnerID, IsOwner}` and always read data as the dashboard's owner. Every store method filters on `user_id`. Boardly has no admin role, no groups and no way to look up other users, and that stays so. Sign-in already answers the same for every address (`auth-magic-link`), and invitations follow it.

This is the first of four changes, so its schema has to fit the later ones.

## Goals / Non-Goals

**Goals:**
- A sharing model and access rule that the later changes extend without changing them.
- Read-only dashboard sharing without touching how widgets load data.

**Non-Goals:**
- The level `edit`. The schema allows it, the API rejects it until a later change needs it.
- Sharing anything other than dashboards, and public access to anything other than dashboards.
- Notifications other than the invitation email.
- Accepting automatically on registration.
- Connected apps. Karakeep, Linkwarden and Memos stay limited to the token owner's data.

## Decisions

### One `shares` table, with a foreign key per resource type

```
user_groups(id, name, created_at, updated_at)
user_group_members(group_id, user_id, role, joined_at, PK(group_id, user_id))
invitations(id, inviter_id, email, token_hash, locale,
       dashboard_id | note_id | note_category_id | favourite_folder_id | group_id,   -- exactly one
       level | role,                                                                 -- share or group
       created_at, expires_at)
invitation_notices(email PK, sent_at)
shares(id, owner_id,
       dashboard_id | note_id | note_category_id | favourite_folder_id,   -- exactly one
       user_id | group_id,                                                -- exactly one
       level CHECK (level IN ('read','edit')), created_at)
```

All references are real foreign keys with `ON DELETE CASCADE`, and CHECK constraints enforce "exactly one". Partial unique indexes allow one share per resource and principal; this change adds the two for `dashboard_id`.

All four resource columns exist from the start, although three stay unused until later changes. SQLite can't change a CHECK without rebuilding the table, as migration 0008 had to, and the "exactly one resource" check is the one that matters most.

Alternatives: a `resource_kind` + `resource_id` pair loses cascading deletes and needs cleanup code in every delete path. One table per resource type repeats the logic four times.

`owner_id` repeats the resource's owner, because it makes "remove the shares this person made with this group" a single statement and every share listing a simple filter. The store always sets it from the resource, never from the request.

The tables are named `user_groups` and `user_group_members` because `GROUPS` is an SQL keyword.

### Public stays `dashboards.is_public`

Only dashboards can be public, so public doesn't need to be a principal. Keeping the column means no data migration and no change to the API or export format. The UI moves to the share dialog and keeps using `PATCH /api/dashboards/{id}`.

### Access is resolved first, stores stay owner-scoped

The rule for later changes: a handler resolves the viewer's access and the resource's real owner, then calls the existing store with the owner's ID. A handler that forgets this only finds the viewer's own data, so it fails closed.

In this change `Lookup` adds one `EXISTS` to its visibility check and to the switcher query, matching a share with the viewer or with any of their groups. A signed-out viewer has the ID `""`, which matches nothing. Non-owners keep `IsOwner: false`, so widgets, the response fields and the frontend behave exactly as for visitors of a public dashboard.

### A generic path for shares

`/api/shares/{type}/{id}`, with only `dashboard` valid for now. Later changes add resource types without adding routes. Unknown types answer 404.

### Invitations by email, with one path for every address

Registration is open by default, so anything that confirms an account would expose users. The request does the same work for every address: validate, insert the invitation, queue a job. It never reads `users`.

A new `invitation_mail` job in the existing `jobs` package decides what to send, after the response has gone out. It looks up the address, and either creates the token and mails the link, or mails the notice. Sending in the request would differ in timing, because a notice can be skipped by its limit.

The job generates the token itself and stores only its hash, so the plain token is never in the job's payload. A retry generates a new one. `token_hash` is empty until then, and a notice never has one, so nobody can accept without proving the address. Someone with no account proves it by signing up, and then accepts from the list in the app.

The notice carries nothing the inviter typed, so the server can't be used to send chosen text to arbitrary addresses. The limits are 30 invitations per user per hour with the existing `ratelimit` package, and one notice per address per 24 hours, recorded in `invitation_notices`. That record is separate from the invitation, because revoking and repeating an invitation would otherwise send another notice each time. It has no reference to any user and is removed after 24 hours by the sweep, or with the account of that address.

Alternatives: a lookup by slug or email reveals who has an account. Accepting automatically on registration would give access or a group membership without consent. A token in the notice adds nothing, because signing in already proves the address, and it would make the notice look more like phishing.

`GET /api/invitations` matches open invitations against the session's address, so an invitation made before the account existed appears once the person signs up. Sign-up is the proof of ownership, because an account is only created by redeeming a link or code mailed to that address. This also gives registered users a way to accept when the email is lost.

Accepting requires the session's account to have the invited address, so a forwarded link is useless. It runs in one transaction that creates the share or membership and deletes the invitation. An existing share or membership is left unchanged.

The preview and accept calls take the token in the body, like the magic link. The page is `/invitation?token=…` with no sub-path, so it can't shadow any `/<user-slug>/<dashboard-slug>` address.

### Groups are shared objects with roles

A group has no single owner; the admins manage it. The last-admin check runs in the same transaction as the change it guards.

Leaving or being removed deletes that person's shares with the group, in the same transaction. A person who is no longer in a group shouldn't keep giving its members access.

On account deletion, `accounts.deleteAccount` first promotes the member with the earliest `joined_at` (ties broken by user ID) in groups where the account is the only admin, and deletes groups where it is the only member. The cascades do the rest.

### Package layout

`internal/groups` holds groups and memberships. `internal/shares` holds shares and the "shared with me" query. `internal/invitations` holds invitations, the job and the sweep. `internal/dashboards` keeps the lookup. The migration is `0028_sharing.sql`. On the frontend the feature lives in `features/sharing`, `features/groups` and `features/invitations`, because `features/share` is already the "Save link" page.

### The rename touches only texts

`showIfPublic` stays as the stored key. Renaming it would mean migrating every data widget's stored config and accepting both names on import, for no user-visible gain. Besides the label, the hints change from "Visitors of a public dashboard" to wording that also fits shared viewers.

## Roadmap for the later changes

These decisions are agreed and recorded here so that the next changes build on them. They aren't part of this change.

**2. Favourite folders (read and edit).** Sharing a folder includes everything below it, and access is the highest level on the item or any parent. Reuse `favourites.Scope`. The shared root can't be renamed, moved or deleted by others. Moves between owners are refused. Shared folders appear under "Shared with me", not in the viewer's own tree. Tag suggestions must not reveal tags from outside the shared folder.

**3. Notes and categories (read and edit).** Single notes and categories can be shared, and categories inherit like folders. Needs a clear conflict flow, because two editors will hit the version check that stops autosave. `pinned` and `archived` stay owner-only. Attachments uploaded to a shared note are stored as the note owner's. Attachment links are served to anyone with the ID, so ending a share doesn't invalidate a link someone already has.

**4. Dashboard editing and widget ownership.**
- `dashboard_widgets` gets an owner, set to the dashboard's owner for existing widgets. Each widget resolves as its owner.
- Editors can change the layout and sections and create, edit and delete their own widgets with their own data.
- Only a widget's owner can change its data settings, including "Show to other viewers". This also applies to the dashboard's owner, who could otherwise read another person's private data by repointing their widget.
- The dashboard's owner can delete any widget. Editors can't delete a section that contains someone else's widget.
- `PUT /layout` deletes and reinserts the whole tree, and hidden widgets and redacted configs aren't sent to non-owners. The save must compare with the stored tree by widget ID and keep the stored owner and locked settings.
- When a widget's owner loses edit access, the dashboard's owner and editors see the reason and the name, and read-only viewers and public visitors see "This content isn't available". Editors get an opaque placeholder for widgets only their owner can see.
- The export includes only widgets owned by the dashboard's owner.

## Risks / Trade-offs

- [A group admin can add someone who then gets access to what others shared with the group] → The share dialog says so whenever a group is involved, and sharers can leave the group or remove the share.
- [Any account can make the server send mail to arbitrary addresses] → The limits above, a notice with no text from the inviter, and no notice more than once a day per address.
- [An invited address stays "Invited" after a registered person ignored it, and also after a notice] → Intended, because any difference would reveal the account. The entry disappears at expiry.
- [Accepting reveals to the inviter which account an address belongs to] → Accepting is the recipient's own decision.
- [Someone signs up more than 14 days after the invitation] → The invitation is gone and the inviter has to invite again. The notice names the 14 days.
- [Addresses of people without an account are stored] → At most 14 days plus the cleanup. Operators write their own privacy policy, so the README points this out.
- [A user already has the slug `shared`, `groups` or `invitation`] → Their dashboards at `/shared/<dashboard-slug>` keep working, because the new pages have no sub-paths. Only the bare addresses change, and those only ever redirected to `/`.
- [Unused columns in `shares` until later changes] → Accepted, to avoid rebuilding the table three times.
- [Existing dashboards with `showIfPublic` widgets are unchanged, but a widget marked for public view is now also shown to people it's shared with] → That's the intended meaning of "Show to other viewers".

## Migration Plan

`0028_sharing.sql` only adds tables, so existing data is unchanged and public dashboards keep working. To roll back, deploy the previous version; the extra tables are ignored by it.
