# link-widget Specification

## Purpose
A dashboard widget that shows one link, either one of the owner's favourites or a plain URL, with a title, an image or icon, and a target.

## Requirements

### Requirement: Link sources
The settings dialog of a link widget SHALL offer two sources: "Favourite" / "Favorit" (pick an existing link from the owner's favourites) and "URL" (an absolute `http` or `https` URL of at most 2048 characters that isn't stored as a favourite). The dialog can't be confirmed without a link. With the source "Favourite", a "New favourite…" / "Neuer Favorit…" button SHALL open the same "New link" dialog as on `/favourites`, on top of the settings and with a choice of folder (the root folder unless another is chosen). New favourites are not entered in the settings dialog itself. Saving that dialog SHALL create the favourite and choose it as the widget's link. The favourite stays when the widget dialog or the dashboard edit is cancelled. A layout with a link widget without a valid URL or favourite is rejected with 400 and `invalid_layout`.

#### Scenario: Pick a favourite
- **WHEN** Ann picks "Link", chooses the source "Favourite", selects "Mail" and confirms
- **THEN** a link widget for "Mail" is added to the draft

#### Scenario: New favourite
- **WHEN** Ann activates "New favourite…", enters `https://nas.example.org`, chooses the folder "Home" and saves
- **THEN** the link appears in "Home" on `/favourites`, it is chosen in the link settings, and the widget uses it once she confirms them

#### Scenario: Plain URL
- **WHEN** Ann chooses "URL", enters `https://example.org` and confirms
- **THEN** the widget links to `https://example.org`, and no favourite is created

#### Scenario: Invalid URL
- **WHEN** Ann enters `javascript:alert(1)` as the URL
- **THEN** the dialog shows a translated validation message and can't be confirmed

### Requirement: Following the favourite
A link widget with a favourite as its source SHALL always show the favourite's current URL and, when the widget has no title of its own, the favourite's current title.

#### Scenario: Favourite renamed
- **WHEN** Ann changes the favourite "Mail" to the title "Webmail" and the URL `https://webmail.example.org`, and opens the dashboard
- **THEN** the widget shows "Webmail" and links to `https://webmail.example.org`

### Requirement: Deleted favourite
When the favourite behind a link widget no longer exists, the widget SHALL work as a URL-only link with the favourite's last known URL and title, for the owner and for visitors. The next layout save SHALL store the widget as a URL-only link, with that title as its own title unless it already had one. It no longer refers to the favourite after that.

#### Scenario: Favourite deleted
- **WHEN** Ann deletes the favourite "Mail" and opens the dashboard
- **THEN** the widget still shows "Mail" and links to the last URL

#### Scenario: Stored as URL
- **WHEN** Ann then edits the dashboard and saves
- **THEN** the widget's settings show the source "URL" with that URL and the title "Mail"

### Requirement: Link widget settings
Besides its source and the common width, a link widget SHALL have an optional title (at most 200 characters after trimming; empty means none), an icon, a target ("New tab" / "Neuer Tab", the default, or "Same tab" / "Gleicher Tab") and a size (`S`, `M` or `L`, default `M`). Values outside these rules are rejected with 400 and `invalid_layout`.

#### Scenario: Defaults
- **WHEN** Ann picks "Link" in the widget picker
- **THEN** the settings show the icon "Website icon", the target "New tab" and the size M

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

### Requirement: Link widget height
A link widget SHALL always be as tall as its content. Its settings dialog SHALL NOT offer a height choice, and the server SHALL store a link widget with the height "auto", whatever height a layout sends for it.

#### Scenario: No height choice
- **WHEN** Ann opens the settings of a link widget
- **THEN** the dialog offers the width but no height

#### Scenario: Fixed height sent
- **WHEN** a layout with a link widget of height 3 is saved
- **THEN** the widget is stored and returned with the height "auto"

### Requirement: Link widget rendering
A link widget SHALL show its icon as chosen, and a globe for "None", for a website without a known favicon, or when an image can't be loaded. Next to it, it shows its title on the first line and its URL on the second, or only its URL when it has no title. Each line uses the available width and is cut off with "…" when it doesn't fit. The full title and URL are available as a tooltip. The size sets the icon size, the font size and the padding, on the same scale as the favourite explorer. Activating the widget opens the URL in a new tab or in the same tab, as set. In edit mode, activating it doesn't open the link.

#### Scenario: With title
- **WHEN** a widget has the title "Mail" and the URL `https://mail.example.org`
- **THEN** it shows "Mail" on the first line and `https://mail.example.org` on the second

#### Scenario: Without title
- **WHEN** a widget with a plain URL has no title
- **THEN** it shows only the URL

#### Scenario: Long text
- **WHEN** the title is longer than the widget is wide
- **THEN** the title fills the width and ends with "…"

#### Scenario: Fallback icon
- **WHEN** a widget's icon is "Website icon" and no favicon is known for its site
- **THEN** it shows a globe icon

#### Scenario: Same tab
- **WHEN** a widget's target is "Same tab" and a visitor activates it
- **THEN** the URL opens in the current tab

### Requirement: Link widget on public dashboards
Link widgets SHALL be shown to visitors of public dashboards. A visitor's response contains the widget's URL, title and image, and nothing else from the owner's favourites.

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with a link widget for the favourite "Mail"
- **THEN** the visitor sees the widget, and the response contains no other favourite and none of the favourite's tags

### Requirement: Link widget details button
For the owner, outside edit mode, a link widget whose source is a favourite that still exists SHALL show an info button to the right of the link, which opens the favourite's details (see the favourite-details spec). Activating it SHALL NOT follow the link. A link widget with the source "URL", or whose favourite was deleted, SHALL show no info button.

#### Scenario: Favourite link
- **WHEN** Ann views her dashboard with a link widget for the favourite "Mail" and activates its info button
- **THEN** the details of "Mail" open, and no new tab opens

#### Scenario: Plain URL
- **WHEN** a link widget has the source "URL"
- **THEN** it shows no info button

#### Scenario: Deleted favourite
- **WHEN** the favourite behind a link widget no longer exists
- **THEN** the widget shows no info button

### Requirement: Domain hint for a plain URL
With the source "URL", the link widget settings SHALL show the same domain hint under the URL field as the "New link" dialog: the number of the owner's favourite links on the entered URL's domain, linking to `/favourites?domain=<domain>` in a new tab. No hint is shown while the URL is invalid or when no favourite has that domain.

#### Scenario: Plain URL on a known domain
- **WHEN** Ann has two favourites on `nas.example.org`, chooses "URL" and enters `https://nas.example.org/files`
- **THEN** the settings show "2 favourites on nas.example.org"

#### Scenario: No favourite on the domain
- **WHEN** Ann chooses "URL" and enters `https://example.org`, and none of her favourites is on `example.org`
- **THEN** no hint is shown
