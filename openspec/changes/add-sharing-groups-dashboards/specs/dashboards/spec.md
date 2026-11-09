# Spec Delta

## MODIFIED Requirements

### Requirement: Dashboards page
`/dashboards` SHALL be available only to signed-in users; signed-out visitors are sent to `/login`. It shows the heading "Dashboards" / "Dashboards" and lists the user's dashboards with title, address (`/<user-slug>/<dashboard-slug>`, as a link), a "Default" badge and a "Public" badge where they apply. Each dashboard has share, edit and delete actions. The share action opens the share dialog (see the `sharing` capability). Without dashboards, the page shows an empty state with a create action.

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/dashboards`
- **THEN** the browser shows `/login`

#### Scenario: List with badges
- **WHEN** Ann (slug `ann`) has a public default dashboard "Home" (slug `home`)
- **THEN** the page shows "Home" with the link `/ann/home` and the badges "Default" and "Public"

### Requirement: Create and edit dashboards
The dashboards page SHALL offer a dialog to create a dashboard and the same dialog to edit one, with fields for title, slug, "Default dashboard", desktop columns, mobile columns and the mobile breakpoint (a choice of 640, 768 and 1024 px). The column fields start with the defaults when creating. The mobile column count can't be set higher than the desktop count. When creating, the slug follows the title until the user edits the slug field: the title is lowercased, `ä`, `ö`, `ü` and `ß` become `ae`, `oe`, `ue` and `ss`, other accents are removed, each run of other characters becomes one hyphen, and the result is trimmed of hyphens and cut to 32 characters. Server errors such as `slug_taken` are shown at the matching field. Whether a dashboard is public is set in the share dialog (see the `sharing` capability); a new dashboard starts as not public.

#### Scenario: Slug from title
- **WHEN** the user types the title "Küche & Haus" in the create dialog
- **THEN** the slug field shows `kueche-haus`

#### Scenario: Manual slug is kept
- **WHEN** the user edits the slug to `kitchen` and then changes the title
- **THEN** the slug stays `kitchen`

#### Scenario: Taken slug
- **WHEN** the user saves a dashboard with a slug they already use
- **THEN** the dialog stays open and shows that the slug is taken at the slug field

#### Scenario: Column defaults in the dialog
- **WHEN** the user opens the create dialog
- **THEN** the column fields show 6 desktop columns, 2 mobile columns and 768 px

#### Scenario: Mobile above desktop
- **WHEN** the user sets 3 desktop columns and 4 mobile columns
- **THEN** a hint is shown and saving is disabled

### Requirement: Dashboard address
A dashboard SHALL be shown at `/<user-slug>/<dashboard-slug>`. Both parts are matched ignoring case. The page shows the dashboard's title and its sections, or a translated placeholder that says the dashboard is empty when it has none. The owner can always see their dashboards. Anyone else can see a dashboard only if the owner's account isn't scheduled for deletion and the dashboard is public or, for a signed-in viewer, shared with them directly or through a group they belong to (see the `sharing` capability). In every other case (unknown user, unknown dashboard, a dashboard of another user that is neither public nor shared with the viewer, any dashboard of another user whose account is scheduled for deletion) the page shows the not-found page. Fixed routes of the app, such as `/login/verify`, always take precedence over this route.

The web app loads it with `GET /api/u/{userSlug}/{dashboardSlug}`. This works with or without a session and responds 404 `not_found` in exactly the cases above. The response contains the dashboard (`slug`, `title`), its column settings and layout, whether the viewer is the owner, the owner's slug, and the switcher entries (`slug`, `title`): all of the owner's dashboards for the owner, only the ones the viewer can see for everyone else. For the owner, the response also contains the dashboard's `id`, `isDefault` and `isPublic`. For anyone else, these three fields are left out.

#### Scenario: Owner opens private dashboard
- **WHEN** Ann opens `/ann/work`, which isn't public and has no sections
- **THEN** she sees the title "Work" and the empty-dashboard placeholder

#### Scenario: Visitor opens public dashboard
- **WHEN** a signed-out visitor opens `/ann/home`, which is public and has sections
- **THEN** they see the title and the sections

#### Scenario: Visitor opens private dashboard
- **WHEN** Bob opens `/ann/work`, which isn't public and isn't shared with Bob
- **THEN** the not-found page is shown, the same as for `/ann/no-such-dashboard`

#### Scenario: Owner scheduled for deletion
- **WHEN** Ann's account is scheduled for deletion and a signed-out visitor opens her public dashboard `/ann/home`
- **THEN** the not-found page is shown, while Ann herself still sees the dashboard

#### Scenario: Deletion cancelled
- **WHEN** Ann cancels her scheduled deletion and a signed-out visitor opens `/ann/home`
- **THEN** they see the dashboard again

#### Scenario: Case-insensitive address
- **WHEN** Ann opens `/Ann/Work`
- **THEN** the dashboard "Work" is shown

#### Scenario: Owner fields
- **WHEN** Ann requests `GET /api/u/ann/home` for her public default dashboard
- **THEN** the response contains its `id`, `isDefault` true and `isPublic` true

#### Scenario: Visitor fields
- **WHEN** a signed-out visitor requests `GET /api/u/ann/home`
- **THEN** the response contains no `id`, `isDefault` or `isPublic`

#### Scenario: Viewer opens a shared dashboard
- **WHEN** Ann shared her private dashboard `/ann/work` with Bob and Bob opens it
- **THEN** Bob sees the title and the sections

#### Scenario: Shared dashboard signed out
- **WHEN** a signed-out visitor opens `/ann/work`, which is shared with Bob but isn't public
- **THEN** the not-found page is shown

#### Scenario: Fields for a shared viewer
- **WHEN** Bob requests `GET /api/u/ann/work` for a dashboard shared with Bob
- **THEN** the response contains no `id`, `isDefault` or `isPublic`

### Requirement: Dashboard switcher
On pages that show a dashboard (`/<user-slug>/<dashboard-slug>`, and `/` when it shows the default dashboard), the header SHALL show the dashboard's title next to the product name when there is more than one dashboard to choose from. For the owner, those are all of their dashboards. For anyone else, they are the owner's dashboards that the viewer can see. The title is a dropdown trigger. The dropdown lists the choices, sorted by title, with the current one marked. Picking another one navigates to `/<user-slug>/<dashboard-slug>`. With only one choice, or on any other page, no switcher is shown.

#### Scenario: Owner switches
- **WHEN** Ann, who has "Home" and "Work", is on `/` showing "Home" and picks "Work" in the switcher
- **THEN** the browser shows `/ann/work` and the switcher shows "Work"

#### Scenario: Single dashboard
- **WHEN** Ann has only one dashboard and opens it
- **THEN** the header shows no switcher

#### Scenario: Visitor sees only public dashboards
- **WHEN** Ann has the public dashboards "Home" and "Links" and the private "Work", and a visitor opens `/ann/home`
- **THEN** the switcher lists "Home" and "Links" but not "Work"

#### Scenario: One public dashboard
- **WHEN** Ann has exactly one public dashboard and a visitor opens it
- **THEN** the header shows no switcher

#### Scenario: Other pages
- **WHEN** Ann, who has several dashboards, opens `/settings`
- **THEN** the header shows no switcher

#### Scenario: Viewer sees shared and public dashboards
- **WHEN** Ann has the public dashboard "Home", "Work" shared with Bob and the private "Taxes", and Bob opens `/ann/work`
- **THEN** the switcher lists "Home" and "Work" but not "Taxes"

### Requirement: Dashboard settings from the dashboard page
"Settings" in the dashboard actions menu SHALL open the same dialog as the edit action on `/dashboards`, filled with the dashboard's current values, including the slug rename confirmation. Cancelling saves nothing and keeps the page as it was. After a successful save, the page shows the updated dashboard:
- On `/<user-slug>/<dashboard-slug>`, when the slug changed, the browser navigates to `/<user-slug>/<new-slug>`.
- On `/`, when the default flag was removed, the browser navigates to `/<user-slug>/<slug>` with the dashboard's saved slug.
- Otherwise the page stays at its address and shows the new title and column settings.

Navigating to the new address doesn't redirect the old one: the old address shows not found, as for any rename.

#### Scenario: Open with current values
- **WHEN** Ann picks "Settings" on `/ann/home`, a public default dashboard titled "Home"
- **THEN** the dialog shows the title "Home", the slug `home`, and "Default dashboard" checked

#### Scenario: Change the title
- **WHEN** Ann changes the title to "Start" in the dialog on `/ann/home` and saves
- **THEN** the page stays at `/ann/home` and shows the title "Start"

#### Scenario: Change the slug
- **WHEN** Ann changes the slug from `home` to `start` on `/ann/home`, saves and confirms
- **THEN** the browser shows `/ann/start` with the dashboard

#### Scenario: Remove the default on /
- **WHEN** Ann is on `/` showing her default dashboard "Home" (slug `home`), unchecks "Default dashboard" and saves
- **THEN** the browser shows `/ann/home` with the dashboard

#### Scenario: Change the slug on /
- **WHEN** Ann is on `/` showing her default dashboard, changes only its slug, saves and confirms
- **THEN** the browser stays at `/` and shows the dashboard
