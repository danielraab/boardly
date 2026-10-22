## MODIFIED Requirements

### Requirement: Image storage
The server SHALL store each image as a file under `<data dir>/images/` and keep its ID, content type, size, source (`favicon` or `upload`) and, for uploads, the uploading user. Image IDs SHALL be random and unguessable. The data directory is set by `BOARDLY_DATA_DIR` and defaults to `./data`.

#### Scenario: Stored on disk
- **WHEN** a user uploads an image
- **THEN** a file for it exists under `<data dir>/images/`, and it is still served after a restart

### Requirement: Private address protection
Unless `BOARDLY_FETCH_ALLOW_PRIVATE` is `true`, the server SHALL NOT connect to loopback, private, link-local, unique-local, carrier-grade NAT, multicast or unspecified addresses, in IPv4 or IPv6, when it fetches favicons. The check SHALL apply to the address actually connected to after DNS resolution, and to every redirect. `BOARDLY_FETCH_ALLOW_PRIVATE` defaults to `false`.

#### Scenario: Name that resolves to a private address
- **WHEN** private addresses are not allowed and a link's host name resolves to `10.0.0.5`
- **THEN** no connection is made and no favicon is stored

#### Scenario: Redirect to a private address
- **WHEN** a public page redirects to `http://127.0.0.1/`
- **THEN** the redirect isn't followed

#### Scenario: Allowed
- **WHEN** `BOARDLY_FETCH_ALLOW_PRIVATE=true` and a link points to `http://192.168.1.10:8096`
- **THEN** the server fetches that site's favicon
