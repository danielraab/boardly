# Proposal

## Why

A dashboard can only be private or public to the whole internet, and notes and favourites can't be shared at all. Users need to give specific people and groups access. This is the first of four changes that build a user and group permission system. It lays the foundation and delivers read-only dashboard sharing.

## What Changes

- Add shares: the owner of a dashboard can give a group, or a person who accepted an invitation, read access. No share means no access. Shares only add access; there is no deny.
- Add email invitations to a dashboard or a group. The inviter gets the same result for every address, and the address is listed as "Invited". An address with an account gets an email with a link to accept. An address without one gets a notice that somebody tried to share something on Boardly, without any details or a link. After signing up with that address within the 14 days, they find the invitation on the "Shared with me" page and can accept it, so nobody has to invite them again.
- Add groups. Any signed-in user can create one and becomes its first group admin. Memberships have the role group admin or group member. Admins invite people and manage members, roles and the group, and a group always keeps at least one admin.
- Any member of a group can share their own dashboards with it. When someone leaves or is removed from a group, their shares with that group end.
- Add a "Shared with me" page that lists the dashboards shared with the viewer, and a "Groups" page.
- Add a share dialog to the dashboards page and the dashboard actions menu. It holds the invitations, the people and groups with access, and the "Public" setting, which moves out of the dashboard dialog. Only dashboards can be public, and `isPublic` stays in the API.
- There is no way to look up or search for users. Once someone has accepted, others see only their name and slug.
- Dashboards shared with a viewer open at their usual address and appear in the owner's dashboard switcher for that viewer. Without access the viewer gets the same not-found answer as for a dashboard that doesn't exist.
- Rename the widget setting "Show on public dashboard" to "Show to other viewers". It also applies to people a dashboard is shared with. The stored `showIfPublic` key stays.
- Reserve the user slugs `shared`, `groups` and `invitation` for the new pages.
- Accounts scheduled for deletion have their shared dashboards hidden, like their public ones. When an account is deleted, its shares and memberships go with it, and its groups stay administered or are deleted when empty.

The three later changes are out of scope and outlined in `design.md`: sharing favourite folders, sharing notes and categories, and dashboard editing with widget ownership.

## Capabilities

### New Capabilities

- `sharing`: shares, how access is decided, the shares API, the share dialog and the "Shared with me" page.
- `invitations`: email invitations to dashboards and groups, the same result for every address, the emails, open invitations in the app, accepting and the invitation page.
- `groups`: groups, membership roles, the groups API and the "Groups" page.

### Modified Capabilities

- `dashboards`: who can open a dashboard and see it in the switcher, the share action, and "Public" moving to the share dialog.
- `dashboard-editing`: the actions menu gets "Share".
- `dashboard-layout`: the layout in the lookup follows the new visibility rules.
- `account-deletion`: shared dashboards are hidden during the grace period, and shares, memberships, invitations and groups are handled on deletion.
- `app-shell`: the user menu links to the new pages, and the page widths include them.
- `user-profile`: `shared`, `groups` and `invitation` become reserved user slugs.
- `data-export`: groups, memberships, shares and invitations are listed as data left out of the export.
- `favourite-explorer-widget`, `feed-widget`, `link-list-widget`, `media-widget`, `note-list-widget`, `note-widget`, `recent-favourites-widget`, `recent-notes-widget`, `selected-notes-widget`: the setting's label becomes "Show to other viewers".

## Impact

- **Backend:** a migration adds `user_groups`, `user_group_members`, `shares` and `invitations`. There are new `groups`, `shares` and `invitations` packages, two mail templates in en and de, a background job that sends the mail, a sweep for expired invitations, a change to the dashboard lookup, and a step in account deletion.
- **API:** new routes under `/api/groups`, `/api/shares`, `/api/invitations` and `/api/shared-with-me`, documented in `boardly.yaml`. Existing routes keep their shape.
- **Frontend:** new `/shared`, `/groups` and `/invitation` pages, a share dialog, changes to the dashboard dialog, the user menu, the reserved slug list and the en and de translations.
- **Users who already have the slug `shared`, `groups` or `invitation`:** they keep their slug and their dashboard addresses keep working, because the new pages have no sub-paths. Only new choices of these slugs are refused.
- **Export and import:** the export format is unchanged and stays at its current version. Groups, memberships, shares and invitations involve other people, so they are neither exported nor imported. The new tables are listed as left out in the export's table coverage test, and an imported dashboard keeps its `isPublic` value as before.
- **Privacy:** the address of an invited person is stored even when they have no account, for at most 14 days plus a day for the cleanup. Operators supply their own privacy policy, so the README tells them to mention this.
- **Connected apps:** Karakeep, Linkwarden and Memos are unaffected and stay limited to the token owner's data.
