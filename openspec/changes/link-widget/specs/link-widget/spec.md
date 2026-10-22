# Spec Delta

## Purpose

A dashboard widget that shows one link, either one of the owner's favourites or a plain URL, with a title, an image or icon, and a target.

## ADDED Requirements

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
Besides its source and the common width, a link widget SHALL have an optional title (at most 200 characters after trimming; empty means none), an optional image, an optional icon from a fixed set of icons, a target ("New tab" / "Neuer Tab", the default, or "Same tab" / "Gleicher Tab") and a size (`S`, `M` or `L`, default `M`). When a URL or favourite is chosen and the image is empty, the image SHALL be filled with the favicon of the URL's origin, if one is found. The owner can replace the image with an uploaded one or remove it. Values outside these rules are rejected with 400 and `invalid_layout`.

#### Scenario: Favicon pre-filled
- **WHEN** Ann enters `https://github.com` as the URL in a new link widget
- **THEN** the image field shows GitHub's favicon

#### Scenario: Upload an image
- **WHEN** Ann uploads a PNG as the widget's image and confirms
- **THEN** the widget shows that image instead of the favicon

#### Scenario: Remove the image
- **WHEN** Ann removes the image, chooses the icon "Mail" and confirms
- **THEN** the widget shows the "Mail" icon

### Requirement: Link widget height
A link widget SHALL always be as tall as its content. Its settings dialog SHALL NOT offer a height choice, and the server SHALL store a link widget with the height "auto", whatever height a layout sends for it.

#### Scenario: No height choice
- **WHEN** Ann opens the settings of a link widget
- **THEN** the dialog offers the width but no height

#### Scenario: Fixed height sent
- **WHEN** a layout with a link widget of height 3 is saved
- **THEN** the widget is stored and returned with the height "auto"

### Requirement: Link widget rendering
A link widget SHALL show its image if it has one, otherwise its icon, otherwise a globe icon. Next to it, it shows its title on the first line and its URL on the second, or only its URL when it has no title. Each line uses the available width and is cut off with "…" when it doesn't fit. The full title and URL are available as a tooltip. The size sets the icon size, the font size and the padding, on the same scale as the favourite explorer. Activating the widget opens the URL in a new tab or in the same tab, as set. In edit mode, activating it doesn't open the link.

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
- **WHEN** a widget has neither an image nor an icon
- **THEN** it shows a globe icon

#### Scenario: Same tab
- **WHEN** a widget's target is "Same tab" and a visitor activates it
- **THEN** the URL opens in the current tab

### Requirement: Link widget on public dashboards
Link widgets SHALL be shown to visitors of public dashboards. A visitor's response contains the widget's URL, title and image, and nothing else from the owner's favourites.

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with a link widget for the favourite "Mail"
- **THEN** the visitor sees the widget, and the response contains no other favourite and none of the favourite's tags
