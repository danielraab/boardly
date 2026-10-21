# Spec Delta

## ADDED Requirements

### Requirement: Dashboard settings from the dashboard page
"Settings" in the dashboard actions menu SHALL open the same dialog as the edit action on `/dashboards`, filled with the dashboard's current values, including the slug rename confirmation. Cancelling saves nothing and keeps the page as it was. After a successful save, the page shows the updated dashboard:
- On `/<user-slug>/<dashboard-slug>`, when the slug changed, the browser navigates to `/<user-slug>/<new-slug>`.
- On `/`, when the default flag was removed, the browser navigates to `/<user-slug>/<slug>` with the dashboard's saved slug.
- Otherwise the page stays at its address and shows the new title and column settings.

Navigating to the new address doesn't redirect the old one: the old address shows not found, as for any rename.

#### Scenario: Open with current values
- **WHEN** Ann picks "Settings" on `/ann/home`, a public default dashboard titled "Home"
- **THEN** the dialog shows the title "Home", the slug `home`, and "Default dashboard" and "Public" both checked

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

## MODIFIED Requirements

### Requirement: Dashboard address
A dashboard SHALL be shown at `/<user-slug>/<dashboard-slug>`. Both parts are matched ignoring case. The page shows the dashboard's title and its sections, or a translated placeholder that says the dashboard is empty when it has none. The owner can always see their dashboards. Anyone else, signed in or not, can see a dashboard only if it is public. In every other case (unknown user, unknown dashboard, a private dashboard of another user) the page shows the not-found page. Fixed routes of the app, such as `/login/verify`, always take precedence over this route.

The web app loads it with `GET /api/u/{userSlug}/{dashboardSlug}`. This works with or without a session and responds 404 `not_found` in exactly the cases above. The response contains the dashboard (`slug`, `title`), its column settings and layout, whether the viewer is the owner, the owner's slug, and the switcher entries (`slug`, `title`): all of the owner's dashboards for the owner, only public ones for everyone else. For the owner, the response also contains the dashboard's `id`, `isDefault` and `isPublic`. For anyone else, these three fields are left out.

#### Scenario: Owner opens private dashboard
- **WHEN** Ann opens `/ann/work`, which isn't public and has no sections
- **THEN** she sees the title "Work" and the empty-dashboard placeholder

#### Scenario: Visitor opens public dashboard
- **WHEN** a signed-out visitor opens `/ann/home`, which is public and has sections
- **THEN** they see the title and the sections

#### Scenario: Visitor opens private dashboard
- **WHEN** Bob opens `/ann/work`, which isn't public
- **THEN** the not-found page is shown, the same as for `/ann/no-such-dashboard`

#### Scenario: Case-insensitive address
- **WHEN** Ann opens `/Ann/Work`
- **THEN** the dashboard "Work" is shown

#### Scenario: Owner fields
- **WHEN** Ann requests `GET /api/u/ann/home` for her public default dashboard
- **THEN** the response contains its `id`, `isDefault` true and `isPublic` true

#### Scenario: Visitor fields
- **WHEN** a signed-out visitor requests `GET /api/u/ann/home`
- **THEN** the response contains no `id`, `isDefault` or `isPublic`
