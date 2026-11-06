# Spec Delta

## MODIFIED Requirements

### Requirement: Private address protection
Unless `BOARDLY_FETCH_ALLOW_PRIVATE` is `true`, the server SHALL NOT connect to loopback, private, link-local, unique-local, carrier-grade NAT, multicast or unspecified addresses, in IPv4 or IPv6, when it fetches favicons, checks a media stream URL, fetches a station logo (see the `media-library` capability), or loads a page for an offline copy (see the `offline-copies` capability). The check SHALL apply to the address actually connected to after DNS resolution, and to every redirect. For the browser that takes offline copy screenshots, it SHALL apply to every request the browser makes, as described in the `offline-copies` capability. `BOARDLY_FETCH_ALLOW_PRIVATE` defaults to `false`.

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

#### Scenario: Offline copy of a private page
- **WHEN** private addresses are not allowed and a link's host resolves to `10.0.0.5`
- **THEN** neither the server nor the browser connects to it, and the link's offline copy is `failed` with `blocked_address`

### Requirement: Background jobs survive restarts
Favicon fetches, preview fetches, preview conversions and offline copy captures that the server has started or queued SHALL be stored in the database, and SHALL run, or run again from the start, after the server restarts. None of them is dropped because many are waiting. A job SHALL be queued in the same transaction as the link change that causes it, so a saved link always has its pending jobs. Pending favicon fetches for the same origin SHALL be combined into one, pending preview fetches for the same link into one for its latest URL, and pending offline copy captures for the same link into one for its latest URL. A job that was interrupted by restarts three times SHALL be dropped and logged, so a broken image can't stop the server from starting. A job that ran and failed is not repeated.

#### Scenario: Restart while fetching
- **WHEN** Ann saves a link and the server restarts before its favicon and preview were fetched
- **THEN** after the restart the server fetches them, and the link gets its favicon and preview

#### Scenario: Restart during a conversion
- **WHEN** the Linkwarden extension uploaded a screenshot and the server restarts before it was converted
- **THEN** after the restart the screenshot is converted and becomes the link's preview

#### Scenario: Many links of one site
- **WHEN** a connected app saves 50 links to `https://github.com/` in a row before GitHub's favicon is fetched
- **THEN** the server fetches GitHub's favicon once

#### Scenario: Image that keeps crashing the server
- **WHEN** a conversion was interrupted by a restart three times
- **THEN** it isn't started again, the link's preview is unchanged and the server logs a warning

#### Scenario: Restart during a capture
- **WHEN** Ann saves a link and the server restarts before its offline copy was captured
- **THEN** after the restart the capture runs, and the link's copy becomes `ready`

#### Scenario: Interrupted capture dropped
- **WHEN** a capture was interrupted by a restart three times
- **THEN** it isn't started again, the copy is `failed` with `fetch_failed`, and the server logs a warning
