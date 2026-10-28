## MODIFIED Requirements

### Requirement: Settings tabs
The settings page SHALL show the tabs "Profile" / "Profil", "Passkeys" / "Passkeys", "Connected apps" / "Verbundene Apps" and "Import / Export" / "Import / Export", in this order. "Profile" is selected by default. The selected tab is reflected in the URL (`?tab=profile`, `?tab=passkeys`, `?tab=apps` or `?tab=data`), so it survives a reload; an unknown value selects "Profile". Each tab label stays on one line. When the tabs don't fit the width of the page, the tab bar scrolls horizontally on its own without making the page wider than the screen, and the selected tab is scrolled into view.

#### Scenario: Default tab
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and its content is shown

#### Scenario: Passkeys tab after reload
- **WHEN** a signed-in user selects "Passkeys" and reloads the page
- **THEN** the URL contains `?tab=passkeys` and the "Passkeys" tab is still selected

#### Scenario: Connected apps tab after reload
- **WHEN** a signed-in user selects "Connected apps" and reloads the page
- **THEN** the URL contains `?tab=apps` and the "Connected apps" tab is still selected

#### Scenario: Import / Export tab after reload
- **WHEN** a signed-in user selects "Import / Export" and reloads the page
- **THEN** the URL contains `?tab=data` and the "Import / Export" tab is still selected

#### Scenario: Narrow screen
- **WHEN** a signed-in user opens `/settings` on a screen too narrow for all tabs
- **THEN** the tab labels stay on one line, the tab bar can be swiped sideways to reach the other tabs, and the page itself does not scroll horizontally

#### Scenario: Selected tab off screen
- **WHEN** a signed-in user opens `/settings?tab=data` on a screen too narrow for all tabs
- **THEN** the tab bar is scrolled so that the "Import / Export" tab is visible
