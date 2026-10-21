# Spec Delta

## ADDED Requirements

### Requirement: Dashboard column settings
Each dashboard SHALL have a desktop column count (1–12, default 6), a mobile column count (1 up to the desktop column count, default 2) and a mobile breakpoint of 640, 768 or 1024 pixels (default 768). Values outside these ranges are rejected with 400 and `invalid_columns`. Changing them never changes the stored widths of the dashboard's sections.

#### Scenario: Defaults
- **WHEN** a dashboard is created without column settings
- **THEN** it has 6 desktop columns, 2 mobile columns and a 768 px breakpoint

#### Scenario: Mobile wider than desktop
- **WHEN** a client sets 4 desktop columns and 5 mobile columns
- **THEN** the server responds 400 with `invalid_columns` and nothing changes

#### Scenario: Unsupported breakpoint
- **WHEN** a client sets the mobile breakpoint to 800
- **THEN** the server responds 400 with `invalid_columns`

## MODIFIED Requirements

### Requirement: Dashboards API
The API SHALL provide, for signed-in users only and scoped to their own dashboards:
- `GET /api/dashboards`: list all of them with `id`, `slug`, `title`, `isDefault`, `isPublic`, `desktopColumns`, `mobileColumns` and `mobileBreakpoint`, sorted by title, ignoring case.
- `POST /api/dashboards`: create one from `title`, `slug`, `isDefault` and `isPublic`, and optionally `desktopColumns`, `mobileColumns` and `mobileBreakpoint`. Responds 201 with the dashboard.
- `PATCH /api/dashboards/{id}`: change any of those fields.
- `DELETE /api/dashboards/{id}`: delete one. Responds 204.

Another user's dashboard ID is answered with 404 `not_found`. Requests without a session get 401.

#### Scenario: List own dashboards
- **WHEN** Ann has "Work" and "home" and requests `GET /api/dashboards`
- **THEN** the response lists "home" then "Work" and none of Bob's dashboards

#### Scenario: Foreign dashboard
- **WHEN** Ann sends `PATCH /api/dashboards/{id}` with the ID of Bob's dashboard
- **THEN** the server responds 404 with `not_found` and Bob's dashboard is unchanged

#### Scenario: Change columns
- **WHEN** Ann sends `PATCH /api/dashboards/{id}` with `desktopColumns` 8
- **THEN** the response shows 8 desktop columns and the other fields are unchanged

### Requirement: Create and edit dashboards
The dashboards page SHALL offer a dialog to create a dashboard and the same dialog to edit one, with fields for title, slug, "Default dashboard", "Public", desktop columns, mobile columns and the mobile breakpoint (a choice of 640, 768 and 1024 px). The column fields start with the defaults when creating. The mobile column count can't be set higher than the desktop count. When creating, the slug follows the title until the user edits the slug field: the title is lowercased, `ä`, `ö`, `ü` and `ß` become `ae`, `oe`, `ue` and `ss`, other accents are removed, each run of other characters becomes one hyphen, and the result is trimmed of hyphens and cut to 32 characters. Server errors such as `slug_taken` are shown at the matching field.

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
A dashboard SHALL be shown at `/<user-slug>/<dashboard-slug>`. Both parts are matched ignoring case. The page shows the dashboard's title and its sections, or a translated placeholder that says the dashboard is empty when it has none. The owner can always see their dashboards. Anyone else, signed in or not, can see a dashboard only if it is public. In every other case (unknown user, unknown dashboard, a private dashboard of another user) the page shows the not-found page. Fixed routes of the app, such as `/login/verify`, always take precedence over this route.

The web app loads it with `GET /api/u/{userSlug}/{dashboardSlug}`. This works with or without a session and responds 404 `not_found` in exactly the cases above. The response contains the dashboard (`slug`, `title`), its column settings and layout, whether the viewer is the owner, the owner's slug, and the switcher entries (`slug`, `title`): all of the owner's dashboards for the owner, only public ones for everyone else. For the owner, the response also contains the dashboard's `id`.

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
