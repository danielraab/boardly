# Spec Delta

## ADDED Requirements

### Requirement: Offline copy button in the details
The details dialog of a link SHALL have an "Offline copy" / "Offline-Kopie" button between "Edit" and "Open". It is shown for every link, also when the link has no offline copy yet. Activating it SHALL show the offline copy dialog of the link in place of the details, as "Edit" does with the edit dialog. Closing the offline copy dialog returns to the details.

#### Scenario: Button order
- **WHEN** Ann opens the details of "Go tour"
- **THEN** the dialog shows the buttons "Edit", "Offline copy" and "Open" in this order

#### Scenario: Open the offline copy
- **WHEN** Ann activates "Offline copy" in the details of "Go tour" and then closes the offline copy dialog
- **THEN** the details of "Go tour" are shown again

### Requirement: Offline copy dialog
The offline copy dialog SHALL be titled with the link's title, or its URL when it has no title, and show the state of the link's offline copy (see the `offline-copies` capability):

- `ready`: "Saved <date and time>" / "Gespeichert <Datum und Uhrzeit>" in the viewer's locale, and the tabs "Article" / "Artikel" and "Screenshot". "Article" shows the article in a frame that runs no scripts and can't reach Boardly's page. "Screenshot" shows the screenshot as an image with the alternative text "Screenshot of <title>" / "Screenshot von <title>". A part the copy doesn't have is shown as a disabled tab, and the first tab with content is selected.
- `pending`: "Saving offline copy…" / "Offline-Kopie wird gespeichert…". The dialog checks the state again every 3 seconds while it is open and shows the copy when it is ready.
- `failed`: "The offline copy couldn't be saved." / "Die Offline-Kopie konnte nicht gespeichert werden." with the reason in words. The parts of an earlier capture are shown below it, when there are any.
- `none`: "No offline copy yet." / "Noch keine Offline-Kopie."

The dialog SHALL have a "Refresh" / "Aktualisieren" button that requests a new capture and then shows the `pending` state, an "Open original" / "Original öffnen" button that opens the link's URL in a new tab, and a close button. Refresh is disabled while the copy is `pending`. When the refresh is rate limited, the dialog says "Too many refreshes, try again later." / "Zu viele Aktualisierungen, versuche es später erneut." The dialog closes with the close button, Escape or a click outside. Links inside the article open in a new tab.

#### Scenario: Ready copy
- **WHEN** Ann opens the offline copy of "Go tour", which has an article and a screenshot
- **THEN** the dialog shows "Saved" with the capture time, and the "Article" tab with the article

#### Scenario: Screenshot tab
- **WHEN** Ann selects "Screenshot"
- **THEN** the dialog shows the screenshot with the alternative text "Screenshot of Go tour"

#### Scenario: No screenshot configured
- **WHEN** the copy has an article but no screenshot
- **THEN** the "Screenshot" tab is disabled and "Article" is selected

#### Scenario: Never captured
- **WHEN** Ann opens the offline copy of an imported link
- **THEN** the dialog says "No offline copy yet." and offers "Refresh"

#### Scenario: Refresh
- **WHEN** Ann activates "Refresh"
- **THEN** the dialog shows "Saving offline copy…", "Refresh" is disabled, and once the capture is done the dialog shows the new copy without being reopened

#### Scenario: Failed with an earlier copy
- **WHEN** a refresh failed because the site was down, and the copy has an article from last week
- **THEN** the dialog shows the failure and last week's article with its capture time
