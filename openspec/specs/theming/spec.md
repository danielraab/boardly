# theming Specification

## Purpose

Defines the color themes of the web app: the available themes and their tokens, contrast minimums, the System default that follows the OS color scheme, the theme menu, remembering the choice in the browser, and applying the theme before first paint.
## Requirements
### Requirement: Available themes
The web app SHALL offer exactly these 7 themes, identified by a stable id: `white` (White / Weiß, light), `black` (Black / Schwarz, dark), `sand` (Sand / Sand, light), `sage` (Sage / Salbei, light), `fjord` (Fjord / Fjord, dark), `dusk` (Dusk / Dämmerung, dark) and `ember` (Ember / Glut, dark). Each theme SHALL define every color token used by the UI: `paper`, `surface`, `ink`, `muted`, `line`, `accent`, `danger`, `on-danger` and `shadow`.

#### Scenario: Theme changes the page colors
- **WHEN** the theme `dusk` is active
- **THEN** the page background, cards, menus, texts, borders and focus ring use the Dusk colors

### Requirement: Themes meet contrast minimums
In every theme, `ink` and `muted` text SHALL have a contrast ratio of at least 4.5:1 against both `paper` and `surface`, `danger` text SHALL have at least 4.5:1 against `paper` and `surface`, `on-danger` SHALL have at least 4.5:1 against `danger`, and `accent` SHALL have at least 3:1 against `paper` and `surface`.

#### Scenario: Contrast check for all themes
- **WHEN** the contrast ratios of the token pairs are computed for each of the 7 themes
- **THEN** every pair meets its minimum

### Requirement: No theme-independent colors
UI components SHALL take all colors from the theme tokens. Error messages SHALL use `danger`, destructive buttons SHALL use `danger` with `on-danger` text, and popups and dialogs SHALL use `shadow` for their shadow color.

#### Scenario: Error text in a dark theme
- **WHEN** the theme `black` is active and the login form shows an error
- **THEN** the error text uses the Black theme's `danger` color

### Requirement: Images follow the theme mode
Where an image comes in a light and a dark variant, such as passkey provider icons, the web app SHALL show the dark variant while a dark theme is active and the light variant while a light theme is active, falling back to the other variant when one is missing. It SHALL switch without reload when the active theme changes.

#### Scenario: Passkey icon in a dark theme
- **WHEN** the theme `fjord` is active and the passkey list shows a provider with light and dark icons
- **THEN** the dark icon is shown

#### Scenario: Only a light icon available
- **WHEN** the theme `black` is active and a provider has only a light icon
- **THEN** the light icon is shown

### Requirement: System theme is the default
The theme choice SHALL be either `system` or one of the 7 theme ids. With `system`, the web app SHALL show `black` when the browser reports `prefers-color-scheme: dark` and `white` otherwise, and SHALL switch between them without reload when that preference changes. When nothing is stored, the choice SHALL be `system`.

#### Scenario: First visit with dark OS setting
- **WHEN** a visitor without a stored choice opens the web app in a browser that prefers a dark color scheme
- **THEN** the Black theme is shown

#### Scenario: OS setting changes while open
- **WHEN** the choice is `system` and the OS color scheme changes from light to dark
- **THEN** the page switches from White to Black without reload

#### Scenario: Explicit theme ignores the OS setting
- **WHEN** the choice is `sand` and the OS color scheme changes to dark
- **THEN** the Sand theme stays active

### Requirement: Theme choice is remembered in the browser
An explicit choice made in the theme menu SHALL be stored in the browser's localStorage under `boardly.theme` and SHALL apply to later visits in the same browser. Choosing System SHALL store `system`. An unknown stored value SHALL be treated as `system`. If localStorage is unavailable, the choice SHALL still apply for the current page view.

#### Scenario: Choice survives a reload
- **WHEN** a visitor chooses Fjord and reloads the page
- **THEN** the Fjord theme is shown

#### Scenario: Invalid stored value
- **WHEN** localStorage contains `boardly.theme` = `neon`
- **THEN** the web app behaves as if the choice were `system`

### Requirement: Theme applied before first paint
The active theme SHALL be applied to the document root before the page is first painted, so that loading or reloading the page never briefly shows a different theme.

#### Scenario: Reload with a dark theme
- **WHEN** the stored choice is `ember` and the visitor reloads the page
- **THEN** the document root already has the Ember theme before the app's JavaScript bundle runs

### Requirement: Native controls follow the theme
The document SHALL declare `color-scheme: dark` while a dark theme is active and `color-scheme: light` while a light theme is active, so that scrollbars, form controls and autofill match the theme.

#### Scenario: Dark theme color scheme
- **WHEN** the theme `fjord` is active
- **THEN** the computed `color-scheme` of the document root is `dark`

### Requirement: Theme menu
The theme switcher SHALL be an icon button with an accessible name such as "Theme" / "Farbschema". Activating it SHALL open a menu that lists System first, then White, Black, Sand, Sage, Fjord, Dusk and Ember. Each theme entry SHALL show a color swatch of that theme and its translated name, and the current choice SHALL be marked as selected. Choosing an entry SHALL apply it immediately without reload and close the menu. The menu SHALL close on Escape and on a click outside it, and SHALL be fully usable with the keyboard.

#### Scenario: Switch theme from the menu
- **WHEN** a visitor opens the theme menu and chooses Sage
- **THEN** the menu closes, the Sage theme is active and Sage is marked as selected the next time the menu opens

#### Scenario: System is marked by default
- **WHEN** a visitor without a stored choice opens the theme menu
- **THEN** System is marked as selected

#### Scenario: Keyboard use
- **WHEN** a visitor focuses the theme button, presses Enter and then presses Escape
- **THEN** the menu opens, then closes, and focus returns to the theme button

### Requirement: Compact buttons with rounded corners
All buttons SHALL use a compact, consistent size and SHALL have rounded corners instead of a fully rounded pill shape. Text buttons of the same kind (primary, secondary, destructive) SHALL have the same size on every page and in every dialog. Chips and badges are not buttons and MAY stay pill-shaped.

#### Scenario: Dialog and page buttons match
- **WHEN** a user sees the "Save" button in a dialog and the "Add passkey" button on the settings page
- **THEN** both have the same height, padding and font size

#### Scenario: No pill-shaped buttons
- **WHEN** any page or dialog is rendered
- **THEN** no button is fully rounded; its corners are rounded but its sides are straight

