# favourite-details Specification

## Purpose
Lets the owner of a dashboard look at everything about one of their favourite links (preview, URL, description and tags) without leaving the dashboard, and go on to open or edit it.

## Requirements

### Requirement: Favourite details dialog
The details dialog of a favourite link SHALL be read-only and show, from the owner's current favourites:
- the link's website icon followed by its title as the dialog title, or its URL when it has no title;
- its preview image, when it has one; without a preview this part is left out;
- its URL as a link that opens in a new tab;
- its description, or "No description" / "Keine Beschreibung" in muted text when it has none;
- its tags, when it has any; without tags this part is left out.

The dialog closes with a close button, Escape or a click outside.

#### Scenario: Link with all details
- **WHEN** Ann opens the details of "Go tour", which has a preview, the description "A tour of Go" and the tag "learning"
- **THEN** the dialog is titled "Go tour" with the website icon before it, and shows the preview, `https://go.dev/tour`, "A tour of Go" and "learning"

#### Scenario: No description
- **WHEN** Ann opens the details of a link without a description
- **THEN** the dialog shows "No description" in muted text

#### Scenario: No preview and no tags
- **WHEN** Ann opens the details of a link without a preview and without tags
- **THEN** the dialog shows no preview image and no tags section

#### Scenario: URL is a link
- **WHEN** Ann activates the URL in the details
- **THEN** the link opens in a new tab

### Requirement: Open and edit from the details
The details dialog SHALL have an "Open" / "Öffnen" button that opens the link in a new tab, and an "Edit" / "Bearbeiten" button that opens the edit dialog of the link, the same as on `/favourites`. Saving the edit dialog SHALL update the favourite; the details and the widgets of the dashboard then show the new values without reloading the page. Cancelling the edit dialog SHALL return to the unchanged details.

#### Scenario: Open
- **WHEN** Ann activates "Open" in the details of "Mail"
- **THEN** `https://mail.example.org` opens in a new tab

#### Scenario: Edit
- **WHEN** Ann activates "Edit" in the details of "Mail", changes the description to "Work mail" and saves
- **THEN** the details show "Work mail", and a widget showing "Mail" reflects any changed title or URL

#### Scenario: Cancel edit
- **WHEN** Ann activates "Edit", changes the title and cancels
- **THEN** the details still show the old title

### Requirement: Details only for the owner
Info buttons that open the details dialog SHALL be shown only to the signed-in owner of the dashboard, only outside edit mode, and only for links that exist in the owner's favourites. They SHALL have the accessible name "Details for <title>" / "Details zu <title>", with the link's URL when it has no title, and be reachable with the keyboard. Visitors of a public dashboard SHALL see no info buttons, and the dashboard lookup response SHALL NOT contain any more of the owner's favourites than before.

#### Scenario: Visitor
- **WHEN** a visitor opens Ann's public dashboard with link, link list and explorer widgets
- **THEN** no widget shows an info button

#### Scenario: Edit mode
- **WHEN** Ann edits the layout of her dashboard
- **THEN** no widget shows an info button

#### Scenario: Keyboard
- **WHEN** Ann moves the focus with Tab through a link widget for "Mail"
- **THEN** the focus reaches the link and then the button "Details for Mail", and Enter opens the details

### Requirement: Delete from the details
Deleting a link from the edit dialog opened from its details (see "Delete from the link edit dialog" in the favourites spec) SHALL close the edit dialog and the details dialog, and SHALL refresh the dashboard's widgets without reloading the page. Widgets that showed the link SHALL then behave as their specs describe for a deleted favourite.

#### Scenario: Delete from a link list
- **WHEN** Ann opens the details of "Mail" in a link list, activates "Edit", then "Delete", and confirms
- **THEN** no dialog is open and the link list no longer shows "Mail"

#### Scenario: Delete the favourite of a link widget
- **WHEN** Ann deletes "Mail" this way from the details of a link widget for "Mail"
- **THEN** no dialog is open and the widget keeps linking to `https://mail.example.org` as a URL-only link
