## ADDED Requirements

### Requirement: Delete from the details
Deleting a link from the edit dialog opened from its details (see "Delete from the link edit dialog" in the favourites spec) SHALL close the edit dialog and the details dialog, and SHALL refresh the dashboard's widgets without reloading the page. Widgets that showed the link SHALL then behave as their specs describe for a deleted favourite.

#### Scenario: Delete from a link list
- **WHEN** Ann opens the details of "Mail" in a link list, activates "Edit", then "Delete", and confirms
- **THEN** no dialog is open and the link list no longer shows "Mail"

#### Scenario: Delete the favourite of a link widget
- **WHEN** Ann deletes "Mail" this way from the details of a link widget for "Mail"
- **THEN** no dialog is open and the widget keeps linking to `https://mail.example.org` as a URL-only link
