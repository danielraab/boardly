## MODIFIED Requirements

### Requirement: Offline copy button in the details
The details dialog of a link SHALL have an "Offline copy" / "Offline-Kopie" button between "Edit" and "Open". It is a square button that shows only the archive icon, with "Offline copy" / "Offline-Kopie" as its accessible name and its tooltip, so the dialog's buttons fit narrow screens. It is shown for every link, also when the link has no offline copy yet. Activating it SHALL show the offline copy dialog of the link in place of the details, as "Edit" does with the edit dialog. Closing the offline copy dialog returns to the details. When the buttons don't fit on one line, they SHALL wrap instead of overflowing the dialog.

#### Scenario: Button order
- **WHEN** Ann opens the details of "Go tour"
- **THEN** the dialog shows the buttons "Edit", "Offline copy" and "Open" in this order

#### Scenario: Icon-only button with a tooltip
- **WHEN** Ann opens the details of "Go tour"
- **THEN** the "Offline copy" button shows no text, only the archive icon, and its tooltip is "Offline copy"

#### Scenario: Open the offline copy
- **WHEN** Ann activates "Offline copy" in the details of "Go tour" and then closes the offline copy dialog
- **THEN** the details of "Go tour" are shown again
