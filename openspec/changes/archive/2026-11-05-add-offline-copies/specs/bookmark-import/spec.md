# Spec Delta

## MODIFIED Requirements

### Requirement: Fetches for imported links
Every imported link SHALL get the same favicon and page-metadata fetches as a link created in Boardly. As for any link, fetched values only fill empty fields, so an imported description is kept. The fetches run in the background after the import, so the response doesn't wait for them. Imported links SHALL NOT get an offline copy captured (see the `offline-copies` capability). The user can request one for each link from its offline copy dialog.

#### Scenario: Preview after import
- **WHEN** a link without a description is imported
- **THEN** its description and preview image are later filled from the page's metadata

#### Scenario: Imported description kept
- **WHEN** a link with the description "Note: Read later" is imported
- **THEN** the metadata fetch doesn't replace that description

#### Scenario: No offline copy after import
- **WHEN** Ann imports a bookmark file with 200 links
- **THEN** no offline copy capture is queued, and each imported link's offline copy has the status `none`
