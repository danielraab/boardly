# Spec Delta

## MODIFIED Requirements

### Requirement: Imported dashboards
With `dashboards`, every dashboard of the file SHALL be added with its title, layout settings and widgets in their order. An imported dashboard SHALL NOT be public and SHALL NOT be the default dashboard. Its slug is the slug of the file when the user has no dashboard with it, otherwise the slug followed by `-2`, `-3` and so on, the first one that is free. A widget whose configuration this server doesn't accept SHALL be left out and counted in the report. Widget heights in an older export SHALL follow the same rule as on an upgrade: a section with height `auto` whose only child is a content widget with a fixed height takes over that height, and all other widget heights are dropped.

#### Scenario: Never public
- **WHEN** an export has the public default dashboard "Home"
- **THEN** the imported "Home" is neither public nor the default dashboard

#### Scenario: Slug taken
- **WHEN** Ann has dashboards with the slugs `home` and `home-2` and imports an export with the dashboard slug `home`
- **THEN** the imported dashboard has the slug `home-3`

#### Scenario: Unknown widget kind
- **WHEN** an export has a widget of a kind this server doesn't know
- **THEN** the dashboard is imported without that widget, and the report counts one skipped widget

#### Scenario: Older export with a fixed-height only child
- **WHEN** an export has an `auto`-height section whose only child is a content widget with a height of 6 rows
- **THEN** the imported section has a height of 6 rows and the imported widget has no height

#### Scenario: Older export with several fixed-height widgets
- **WHEN** an export has a section with two content widgets with fixed heights
- **THEN** both widgets are imported without a height, and nothing is counted as left out

#### Scenario: Export without widget heights
- **WHEN** an export's content widgets have no height
- **THEN** the dashboards are imported as usual
