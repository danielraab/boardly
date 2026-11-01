# Spec Delta

## MODIFIED Requirements

### Requirement: Dashboard address
A dashboard SHALL be shown at `/<user-slug>/<dashboard-slug>`. Both parts are matched ignoring case. The page shows the dashboard's title and its sections, or a translated placeholder that says the dashboard is empty when it has none. The owner can always see their dashboards. Anyone else, signed in or not, can see a dashboard only if it is public and the owner's account isn't scheduled for deletion. In every other case (unknown user, unknown dashboard, a private dashboard of another user, any dashboard of another user whose account is scheduled for deletion) the page shows the not-found page. Fixed routes of the app, such as `/login/verify`, always take precedence over this route.

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
