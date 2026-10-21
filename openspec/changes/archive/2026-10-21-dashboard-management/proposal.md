# Proposal

## Why

Dashboards are the main feature in the README, but a signed-in user still sees a "more is coming" placeholder at `/`. The grid and widget editor need dashboards to exist first: something to create, name, address by URL, pick as the start page and share. This change adds those dashboard entities, how they are managed and how they are addressed. The view and editor come in the next change.

## What Changes

- Every user gets a **user slug**. New accounts get a generated one: 10 random characters `[a-z0-9]`, with `-2`, `-3`, … appended if it's already taken. Existing accounts get one in the migration. Users can change their slug in Settings → Profile. User slugs are unique across all users.
- Users can create any number of **dashboards**. Each has a title, a slug that is unique among that user's dashboards, a **default** flag (at most one per user) and a **public** flag.
- **Slug rules** (users and dashboards): stored lowercase, `[a-z0-9-]`, 1–32 characters (hyphens are allowed anywhere, also at the start or end). Typed uppercase letters are lowercased. User slugs can't be a reserved word (`api`, `login`, `settings`, …). The form shows a hint and won't submit one.
- Changing a slug shows a hint that old links stop working, and saving asks for confirmation.
- New page **`/dashboards`** (signed-in only) to create, edit and delete dashboards, reached from a new "Dashboards" / "Dashboards" item in the user menu.
- New URL **`/<user-slug>/<dashboard-slug>`** shows a dashboard. Owners see all of their own dashboards. Anyone, signed in or not, can see public dashboards. Everything else is not found. Fixed routes always take precedence over these dynamic paths. `/<user-slug>` alone redirects to `/`.
- **`/` for signed-in users** shows the default dashboard while keeping the URL `/`. Without dashboards it shows "Create your first dashboard". With dashboards but no default it shows a "Select your default dashboard" dropdown.
- **Header dashboard switcher**: on a dashboard page, the dashboard's title appears next to "Famodash" as a dropdown when there is more than one dashboard to choose from. Owners get all of their own dashboards; visitors get the owner's public ones. Picking one navigates to `/<user-slug>/<dashboard-slug>`.
- A dashboard page shows its title and an empty-dashboard placeholder for now.

Not part of this change: the grid, widgets and the WYSIWYG editor (next change), redirects from old slugs, and ordering dashboards by hand.

## Capabilities

### New Capabilities

- `dashboards`: dashboard entities (title, slug, default, public), the `/dashboards` management page, slug rules shared with user slugs, the `/<user-slug>/<dashboard-slug>` route with its visibility rules, the `/<user-slug>` redirect, and the header dashboard switcher.

### Modified Capabilities

- `app-shell`: the header gains the dashboard switcher next to the product name, and the signed-in user menu gains a "Dashboards" link to `/dashboards`.
- `home-page`: `/` for signed-in users shows the default dashboard, or a "create your first dashboard" or "select your default dashboard" state, instead of the placeholder.
- `user-profile`: the Profile tab shows and edits the user slug, with uniqueness, the reserved-word check and a rename confirmation. `PATCH /api/me` accepts `slug`, and `GET /api/me` returns it.

## Impact

- **Backend**: migration `0006` (`users.slug` with backfill and a unique index; new `dashboards` table). New package `internal/slugs` for validation, the reserved words and generation. Slug generation in `users.EnsureByEmail`, plus `SetSlug`. New package `internal/dashboards` (store). New handlers for `/api/dashboards` and the public lookup `/api/u/{userSlug}/{dashboardSlug}`, with optional authentication on the lookup. `PATCH /api/me` is extended.
- **Frontend**: new routes `/dashboards`, `/$userSlug` and `/$userSlug/$dashboardSlug`. A new feature folder `features/dashboards`. `Home.tsx` is rewritten. `AppShell` gets the switcher, `UserMenu` the new item, and the Settings Profile tab the slug field. New translation keys (en and de).
- **Next change**: the dashboard view and editor (grid, widgets) builds on the `dashboards` table and the placeholder route.
