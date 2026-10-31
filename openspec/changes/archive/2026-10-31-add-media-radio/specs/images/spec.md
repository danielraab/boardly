# Spec Delta

## MODIFIED Requirements

### Requirement: Cleanup of unused uploads
An uploaded image that no favourite, no dashboard widget, no note, no note category and no media source refers to SHALL be deleted, file and record, once it is older than 24 hours. A preview image that no link refers to SHALL be deleted, file and record, once it is older than one hour. The server checks this on start and then once an hour. Favicons are not deleted by this cleanup.

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

#### Scenario: Station icon
- **WHEN** a station's logo was stored 25 hours ago and the station still uses it as its icon
- **THEN** the image is kept

### Requirement: Private address protection
Unless `BOARDLY_FETCH_ALLOW_PRIVATE` is `true`, the server SHALL NOT connect to loopback, private, link-local, unique-local, carrier-grade NAT, multicast or unspecified addresses, in IPv4 or IPv6, when it fetches favicons, checks a media stream URL or fetches a station logo (see the `media-library` capability). The check SHALL apply to the address actually connected to after DNS resolution, and to every redirect. `BOARDLY_FETCH_ALLOW_PRIVATE` defaults to `false`.

#### Scenario: Name that resolves to a private address
- **WHEN** private addresses are not allowed and a link's host name resolves to `10.0.0.5`
- **THEN** no connection is made and no favicon is stored

#### Scenario: Redirect to a private address
- **WHEN** a public page redirects to `http://127.0.0.1/`
- **THEN** the redirect isn't followed

#### Scenario: Allowed
- **WHEN** `BOARDLY_FETCH_ALLOW_PRIVATE=true` and a link points to `http://192.168.1.10:8096`
- **THEN** the server fetches that site's favicon

#### Scenario: Station URL on a private address
- **WHEN** private addresses are not allowed and a pasted stream URL's host resolves to `10.0.0.5`
- **THEN** no connection is made and the station isn't saved
