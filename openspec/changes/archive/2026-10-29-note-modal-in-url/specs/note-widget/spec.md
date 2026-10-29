## ADDED Requirements

### Requirement: Note editor modal in the address
Opening a note's editor in a modal from a dashboard widget SHALL add `note=<note ID>` to the dashboard's address as a new browser history entry, keeping the dashboard's path. The browser's back button, including Android's, SHALL close the modal and go back to the dashboard's address without `note`. Closing the modal with its close button, Escape or a click outside SHALL do the same, so no extra history entry stays behind; when the modal was opened from an address that already contained `note` (a reload or a link), closing it replaces the address with the one without `note` instead. Unsaved changes are saved when the modal closes in any of these ways. An address with `note` SHALL open the dashboard with the modal of that note for the owner outside edit mode. For anyone other than the owner, in edit mode, or when the note doesn't exist, the parameter is removed from the address and no modal opens. Starting edit mode removes `note`.

#### Scenario: Back closes the modal
- **WHEN** Ann activates a row "Groceries" on `/ann/home` and then presses the back button
- **THEN** the modal closes, the address is `/ann/home`, and a second back leaves the dashboard

#### Scenario: Address while open
- **WHEN** Ann opens "Groceries" from a widget on `/ann/home`
- **THEN** the address is `/ann/home?note=<ID of Groceries>`

#### Scenario: Close button
- **WHEN** Ann closes the modal with its close button
- **THEN** the address is `/ann/home` again and back leaves the dashboard

#### Scenario: Reload
- **WHEN** Ann reloads `/ann/home?note=<ID of Groceries>`
- **THEN** the dashboard shows the modal of "Groceries", and closing it leaves the address `/ann/home`

#### Scenario: Unknown note
- **WHEN** Ann opens `/ann/home?note=gone` and no such note exists
- **THEN** the dashboard is shown without a modal and the address is `/ann/home`

#### Scenario: Visitor
- **WHEN** a visitor opens `/ann/home?note=<ID of Groceries>` of a public dashboard
- **THEN** no modal opens and the address is `/ann/home`
