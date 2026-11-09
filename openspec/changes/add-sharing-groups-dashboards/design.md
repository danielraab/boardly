# Design

## Context

See `proposal.md` for the motivation and the `sharing` and `groups` specs for the behaviour.

Today a dashboard's visibility is decided in `dashboards.Store.Lookup`: the owner sees it, and anyone else only when `is_public` is set and the owner isn't scheduled for deletion. Widgets are resolved with `Viewer{OwnerID, IsOwner}` and always read data as the dashboard's owner. Every store method filters on `user_id`. Boardly has no admin role, no groups and no way to look up other users.

This is the first of four changes, so its schema has to fit the later ones.

## Goals / Non-Goals

**Goals:**
- A sharing model and access rule that the later changes extend without changing them.
- Read-only dashboard sharing without touching how widgets load data.

**Non-Goals:**
- The level `edit`. The schema allows it, the API rejects it until a later change needs it.
- Sharing anything other than dashboards, and public access to anything other than dashboards.
- Notifying people about shares or memberships.
- Connected apps. Karakeep, Linkwarden and Memos stay limited to the token owner's data.

## Decisions

### One `shares` table, with a foreign key per resource type

```
user_groups(id, name, created_at, updated_at)
user_group_members(group_id, user_id, role, joined_at, PK(group_id, user_id))
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

### Finding people only by exact slug

Registration is open by default, so a search or a list would expose users. Slugs are already public in dashboard addresses. `GET /api/users/{slug}` answers only for an exact match, returns no email address and is rate limited per user with the existing `ratelimit` package.

### Groups are shared objects with roles

A group has no single owner; the admins manage it. The last-admin check runs in the same transaction as the change it guards.

Leaving or being removed deletes that person's shares with the group, in the same transaction. A person who is no longer in a group shouldn't keep giving its members access.

On account deletion, `accounts.deleteAccount` first promotes the member with the earliest `joined_at` (ties broken by user ID) in groups where the account is the only admin, and deletes groups where it is the only member. The cascades do the rest.

### Package layout

`internal/groups` holds groups and memberships. `internal/shares` holds shares and the "shared with me" query. `internal/dashboards` keeps the lookup. The migration is `0028_sharing.sql`. On the frontend the feature lives in `features/sharing` and `features/groups`, because `features/share` is already the "Save link" page.

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
- [People are added to groups without accepting, which can put a dashboard on someone's "Shared with me" page unasked] → Adding requires knowing the exact slug, and anyone can leave a group at any time.
- [Exact-slug lookup confirms that a slug exists and shows the name] → Slugs are already public in dashboard addresses, the lookup is signed-in only and rate limited.
- [A user already has the slug `shared` or `groups`] → Their dashboards at `/shared/<dashboard-slug>` keep working, because the new pages have no sub-paths. Only the bare `/shared` and `/groups` addresses change, and those only ever redirected to `/`.
- [Unused columns in `shares` until later changes] → Accepted, to avoid rebuilding the table three times.
- [Existing dashboards with `showIfPublic` widgets are unchanged, but a widget marked for public view is now also shown to people it's shared with] → That's the intended meaning of "Show to other viewers".

## Migration Plan

`0028_sharing.sql` only adds tables, so existing data is unchanged and public dashboards keep working. To roll back, deploy the previous version; the extra tables are ignored by it.
