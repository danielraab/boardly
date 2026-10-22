# Spec Delta

## MODIFIED Requirements

### Requirement: Settings tabs
The settings page SHALL show the tabs "Profile" / "Profil", "Passkeys" / "Passkeys" and "Connected apps" / "Verbundene Apps", in this order. "Profile" is selected by default. The selected tab is reflected in the URL (`?tab=profile`, `?tab=passkeys` or `?tab=apps`), so it survives a reload; an unknown value selects "Profile".

#### Scenario: Default tab
- **WHEN** a signed-in user opens `/settings`
- **THEN** the "Profile" tab is selected and its content is shown

#### Scenario: Passkeys tab after reload
- **WHEN** a signed-in user selects "Passkeys" and reloads the page
- **THEN** the URL contains `?tab=passkeys` and the "Passkeys" tab is still selected

#### Scenario: Connected apps tab after reload
- **WHEN** a signed-in user selects "Connected apps" and reloads the page
- **THEN** the URL contains `?tab=apps` and the "Connected apps" tab is still selected
