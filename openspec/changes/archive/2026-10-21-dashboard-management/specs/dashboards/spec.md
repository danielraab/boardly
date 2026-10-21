# Spec Delta

## ADDED Requirements

### Requirement: Slug format
User slugs and dashboard slugs SHALL use the same format: 1–32 characters from `a-z`, `0-9` and `-`. Hyphens are allowed anywhere, also at the start or end. They are stored in lowercase. The server lowercases a submitted slug before checking it, and a value that still doesn't match is rejected with 400 and `error.code` `invalid_slug`. In the web app, slug inputs turn typed uppercase letters into lowercase as you type. They show a validation hint and block saving while the value doesn't match.

#### Scenario: Uppercase input
- **WHEN** the user types "Work" into a slug field
- **THEN** the field shows "work"

#### Scenario: Invalid characters
- **WHEN** a client submits the slug `my_board`, `my.board`, an empty value or a 33-character value
- **THEN** the server responds 400 with `invalid_slug`

#### Scenario: Hyphens at the ends
- **WHEN** a client submits the slug `-work-`
- **THEN** it is accepted

#### Scenario: Hint in the form
- **WHEN** the user enters `my_work` in a slug field
- **THEN** a hint explains the allowed format and the save action is disabled

### Requirement: Dashboard entity
A signed-in user SHALL be able to own any number of dashboards. Each dashboard has a title, a slug, a default flag and a public flag. The title is trimmed and must be 1–100 characters long; otherwise the server responds 400 with `invalid_title`. A dashboard's slug is unique among the same user's dashboards. A duplicate is rejected with 409 and `slug_taken`. Two users can use the same dashboard slug. A user has no dashboards until they create one.

#### Scenario: Same slug for two users
- **WHEN** Ann and Bob each create a dashboard with the slug `home`
- **THEN** both dashboards are created

#### Scenario: Duplicate slug for one user
- **WHEN** Ann already has a dashboard `home` and creates or renames another to `home`
- **THEN** the server responds 409 with `slug_taken` and nothing changes

#### Scenario: Empty title
- **WHEN** a client creates a dashboard with the title "   "
- **THEN** the server responds 400 with `invalid_title`

### Requirement: At most one default dashboard
A user SHALL have zero or one default dashboard. Making a dashboard the default, on create or on edit, removes the flag from the previous default in the same operation. Clearing the flag leaves the user without a default. Deleting the default dashboard also leaves the user without a default.

#### Scenario: Switch default
- **WHEN** "Home" is the default and the user marks "Work" as default
- **THEN** "Work" is the default and "Home" no longer is

#### Scenario: Delete default
- **WHEN** the user deletes their default dashboard
- **THEN** the remaining dashboards keep existing and none of them is the default

### Requirement: Dashboards API
The API SHALL provide, for signed-in users only and scoped to their own dashboards:
- `GET /api/dashboards`: list all of them with `id`, `slug`, `title`, `isDefault` and `isPublic`, sorted by title, ignoring case.
- `POST /api/dashboards`: create one from `title`, `slug`, `isDefault` and `isPublic`. Responds 201 with the dashboard.
- `PATCH /api/dashboards/{id}`: change any of those fields.
- `DELETE /api/dashboards/{id}`: delete one. Responds 204.

Another user's dashboard ID is answered with 404 `not_found`. Requests without a session get 401.

#### Scenario: List own dashboards
- **WHEN** Ann has "Work" and "home" and requests `GET /api/dashboards`
- **THEN** the response lists "home" then "Work" and none of Bob's dashboards

#### Scenario: Foreign dashboard
- **WHEN** Ann sends `PATCH /api/dashboards/{id}` with the ID of Bob's dashboard
- **THEN** the server responds 404 with `not_found` and Bob's dashboard is unchanged

### Requirement: Dashboards page
`/dashboards` SHALL be available only to signed-in users; signed-out visitors are sent to `/login`. It shows the heading "Dashboards" / "Dashboards" and lists the user's dashboards with title, address (`/<user-slug>/<dashboard-slug>`, as a link), a "Default" badge and a "Public" badge where they apply. Each dashboard has edit and delete actions. Without dashboards, the page shows an empty state with a create action.

#### Scenario: Signed out
- **WHEN** a signed-out visitor opens `/dashboards`
- **THEN** the browser shows `/login`

#### Scenario: List with badges
- **WHEN** Ann (slug `ann`) has a public default dashboard "Home" (slug `home`)
- **THEN** the page shows "Home" with the link `/ann/home` and the badges "Default" and "Public"

### Requirement: Create and edit dashboards
The dashboards page SHALL offer a dialog to create a dashboard and the same dialog to edit one, with fields for title, slug, "Default dashboard" and "Public". When creating, the slug follows the title until the user edits the slug field: the title is lowercased, `ä`, `ö`, `ü` and `ß` become `ae`, `oe`, `ue` and `ss`, other accents are removed, each run of other characters becomes one hyphen, and the result is trimmed of hyphens and cut to 32 characters. Server errors such as `slug_taken` are shown at the matching field.

#### Scenario: Slug from title
- **WHEN** the user types the title "Küche & Haus" in the create dialog
- **THEN** the slug field shows `kueche-haus`

#### Scenario: Manual slug is kept
- **WHEN** the user edits the slug to `kitchen` and then changes the title
- **THEN** the slug stays `kitchen`

#### Scenario: Taken slug
- **WHEN** the user saves a dashboard with a slug they already use
- **THEN** the dialog stays open and shows that the slug is taken at the slug field

### Requirement: Confirm slug renames
When the user changes the slug of an existing dashboard, the edit dialog SHALL show a hint that links using the old address stop working. Saving then asks for confirmation, showing the old and the new address. Cancelling the confirmation keeps the dialog open and saves nothing. Old addresses are not redirected.

#### Scenario: Rename confirmed
- **WHEN** the user changes `home` to `start`, saves and confirms
- **THEN** the dashboard is at `/<user-slug>/start` and `/<user-slug>/home` shows not found

#### Scenario: Rename cancelled
- **WHEN** the user changes the slug, saves and cancels the confirmation
- **THEN** nothing is saved and the edit dialog is still open

### Requirement: Delete dashboards
Deleting a dashboard SHALL ask for confirmation with the dashboard's title, and then delete it.

#### Scenario: Delete
- **WHEN** the user deletes "Work" and confirms
- **THEN** "Work" disappears from the list and its address shows not found

### Requirement: Dashboard address
A dashboard SHALL be shown at `/<user-slug>/<dashboard-slug>`. Both parts are matched ignoring case. The page shows the dashboard's title and a translated placeholder that says the dashboard is empty. The owner can always see their dashboards. Anyone else, signed in or not, can see a dashboard only if it is public. In every other case (unknown user, unknown dashboard, a private dashboard of another user) the page shows the not-found page. Fixed routes of the app, such as `/login/verify`, always take precedence over this route.

The web app loads it with `GET /api/u/{userSlug}/{dashboardSlug}`. This works with or without a session and responds 404 `not_found` in exactly the cases above. The response contains the dashboard (`slug`, `title`), whether the viewer is the owner, the owner's slug, and the switcher entries (`slug`, `title`): all of the owner's dashboards for the owner, only public ones for everyone else.

#### Scenario: Owner opens private dashboard
- **WHEN** Ann opens `/ann/work`, which isn't public
- **THEN** she sees the title "Work" and the empty-dashboard placeholder

#### Scenario: Visitor opens public dashboard
- **WHEN** a signed-out visitor opens `/ann/home`, which is public
- **THEN** they see the title and the placeholder

#### Scenario: Visitor opens private dashboard
- **WHEN** Bob opens `/ann/work`, which isn't public
- **THEN** the not-found page is shown, the same as for `/ann/no-such-dashboard`

#### Scenario: Case-insensitive address
- **WHEN** Ann opens `/Ann/Work`
- **THEN** the dashboard "Work" is shown

### Requirement: User address redirects home
Opening `/<user-slug>` alone SHALL redirect to `/`, whether or not the user slug exists.

#### Scenario: User path
- **WHEN** anyone opens `/ann`
- **THEN** the browser shows `/`

### Requirement: Dashboard switcher
On pages that show a dashboard (`/<user-slug>/<dashboard-slug>`, and `/` when it shows the default dashboard), the header SHALL show the dashboard's title next to the product name when there is more than one dashboard to choose from. For the owner, those are all of their dashboards. For anyone else, they are the owner's public dashboards. The title is a dropdown trigger. The dropdown lists the choices, sorted by title, with the current one marked. Picking another one navigates to `/<user-slug>/<dashboard-slug>`. With only one choice, or on any other page, no switcher is shown.

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
