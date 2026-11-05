# Spec Delta

## MODIFIED Requirements

### Requirement: Fetched values only fill empty fields
A fetched description SHALL only be stored when the link has no description at that moment, and a fetched preview only when the link has no preview at that moment. A description or preview that the user, a connected app or an earlier fetch set is never overwritten by a fetch. Changing a link's URL SHALL remove its preview before the new fetch starts, and SHALL keep its description. The server SHALL NOT remove a description or preview in any other case except the user clearing or removing it and deleting the link. Removing a preview SHALL NOT start a new fetch.

When the offline copy capture that was queued by creating a link or changing its URL takes a screenshot (see the `offline-copies` capability), and the page declares neither `og:image` nor `twitter:image`, the screenshot SHALL become the link's preview, as long as the link has no preview when the conversion finishes. The preview is converted like any fetched preview, so it is at most `BOARDLY_PREVIEW_WIDTH` pixels wide. It is a separate image from the offline copy's screenshot. A capture requested with the offline copy's refresh SHALL NOT set the preview.

#### Scenario: Description written by the user
- **WHEN** a link has the description "My notes" and its URL is changed to a page with `og:description` "Something else"
- **THEN** the link keeps the description "My notes"

#### Scenario: URL changed
- **WHEN** a link has a preview and its URL is changed to another page with an `og:image`
- **THEN** the old preview is removed and the link gets the new page's image as preview

#### Scenario: Screenshot before the fetch finished
- **WHEN** the Linkwarden extension uploads a screenshot for a new link before the page fetch for that link has finished
- **THEN** the link keeps the screenshot as preview, and the fetched image is not stored

#### Scenario: Removed preview stays removed
- **WHEN** the user removes the preview of a link whose page has an `og:image`
- **THEN** the link has no preview and no fetch is started for it

#### Scenario: Screenshot as preview
- **WHEN** the screenshot sidecar is configured and Ann saves a link to a page without `og:image` and `twitter:image`
- **THEN** after the capture, the link's preview is the page's screenshot, at most `BOARDLY_PREVIEW_WIDTH` pixels wide

#### Scenario: Page has a preview image
- **WHEN** Ann saves a link to a page with an `og:image`
- **THEN** the link's preview is made from the `og:image`, and the screenshot is only in the offline copy

#### Scenario: Refresh doesn't restore a removed preview
- **WHEN** Ann removed the screenshot preview of a link and then refreshes its offline copy
- **THEN** the link still has no preview
