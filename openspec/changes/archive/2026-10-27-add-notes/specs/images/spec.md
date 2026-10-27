# Spec Delta

## MODIFIED Requirements

### Requirement: Cleanup of unused uploads
An uploaded image that no favourite, no dashboard widget, no note and no note category refers to SHALL be deleted, file and record, once it is older than 24 hours. A preview image that no link refers to SHALL be deleted, file and record, once it is older than one hour. The server checks this on start and then once an hour. Favicons are not deleted by this cleanup.

#### Scenario: Unused upload
- **WHEN** Ann uploaded an image 25 hours ago and never used it
- **THEN** the image is deleted and its URL responds 404

#### Scenario: Recent upload
- **WHEN** Ann uploaded an image 1 hour ago for a draft she hasn't saved yet
- **THEN** the image is kept

#### Scenario: Preview of a deleted folder's link
- **WHEN** Ann deleted a folder with its contents two hours ago, and a link in it had a preview
- **THEN** the preview image is deleted and its URL responds 404

#### Scenario: Note icon
- **WHEN** Ann uploaded an image 25 hours ago and uses it as the icon of a note or a note category
- **THEN** the image is kept
