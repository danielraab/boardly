# Spec Delta

## MODIFIED Requirements

### Requirement: Themes meet contrast minimums
In every theme, `ink` and `muted` text SHALL have a contrast ratio of at least 4.5:1 against both `paper` and `surface`, `danger` and `success` text SHALL have at least 4.5:1 against `paper` and `surface`, `on-danger` SHALL have at least 4.5:1 against `danger`, and `accent` SHALL have at least 3:1 against `paper` and `surface`. `success` text SHALL also have at least 4.5:1 against every note colour.

#### Scenario: Contrast check for all themes
- **WHEN** the contrast ratios of the token pairs are computed for each of the 7 themes
- **THEN** every pair meets its minimum

### Requirement: No theme-independent colors
UI components SHALL take all colors from the theme tokens. Error messages SHALL use `danger`, confirmations that something succeeded SHALL use `success`, destructive buttons SHALL use `danger` with `on-danger` text, and popups and dialogs SHALL use `shadow` for their shadow color.

#### Scenario: Error text in a dark theme
- **WHEN** the theme `black` is active and the login form shows an error
- **THEN** the error text uses the Black theme's `danger` color

#### Scenario: Saved status in a dark theme
- **WHEN** the theme `black` is active and a note was saved
- **THEN** the save status uses the Black theme's `success` color
