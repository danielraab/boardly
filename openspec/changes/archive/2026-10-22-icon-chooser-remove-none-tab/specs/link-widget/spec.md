# Spec Delta

## MODIFIED Requirements

### Requirement: Link widget icon
A link widget's icon SHALL be one of:
- "Website icon" / "Website-Icon" (the default): the favicon of the origin of the link's current URL, looked up when the dashboard is served. It follows the link: when a favourite's URL moves to another site, the widget shows that site's favicon.
- "Own image" / "Eigenes Bild": an image the owner uploaded (PNG, JPEG, WebP, GIF or ICO, at most 1 MB).
- "Symbol": one of a fixed set of symbols.
- "None" / "Keines": a globe.

The settings dialog SHALL show the current icon and its kind on one button, and that button SHALL open a chooser with three tabs on a single line: "Website icon", "Own image" and "Symbol". It starts on the tab of the current kind, or on "Website icon" when the kind is "None". The "Website icon" tab shows the favicon found for the URL, or says that none was found. The "Own image" tab uploads an image and shows upload errors translated. The "Symbol" tab shows the symbols as a grid with their translated names, which can be filtered by name and moved through with the arrow keys. Next to "Done", the chooser SHALL show a "Remove icon" / "Icon entfernen" button that sets the kind to "None". A choice takes effect in the dialog right away; the chooser closes with "Done", Escape or a click outside. A layout whose icon is "Own image" without an image, or "Symbol" with an unknown symbol, is rejected with 400 and `invalid_layout`.

#### Scenario: Website icon
- **WHEN** Ann enters `https://github.com` as the URL of a new link widget
- **THEN** the icon button shows GitHub's favicon, and so does the widget once she confirms

#### Scenario: Website icon follows the favourite
- **WHEN** the favourite behind a widget with the icon "Website icon" changes its URL from `https://gitlab.com` to `https://github.com`
- **THEN** the widget shows GitHub's favicon

#### Scenario: Own image
- **WHEN** Ann opens the icon chooser, switches to "Own image", uploads a PNG and confirms the settings
- **THEN** the widget shows that image

#### Scenario: Unsupported upload
- **WHEN** Ann uploads an SVG in the "Own image" tab
- **THEN** the tab shows a translated message that the file type isn't supported, and the icon stays as it was

#### Scenario: Symbol
- **WHEN** Ann switches to "Symbol", types "mail" in the filter and chooses "Mail"
- **THEN** the icon button and the widget show the Mail symbol

#### Scenario: Remove the icon
- **WHEN** Ann opens the icon chooser of a widget with the Mail symbol and activates "Remove icon"
- **THEN** the icon button shows a globe and the kind "None", and so does the widget once she confirms

#### Scenario: Tabs on one line
- **WHEN** Ann opens the icon chooser on a phone with the language set to German
- **THEN** the three tabs are shown on one line

#### Scenario: Open with no icon
- **WHEN** Ann opens the icon chooser of a widget whose icon is "None"
- **THEN** the chooser shows the "Website icon" tab
