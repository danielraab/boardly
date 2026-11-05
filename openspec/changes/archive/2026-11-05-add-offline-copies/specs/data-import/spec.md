# Spec Delta

## MODIFIED Requirements

### Requirement: Imported favourites
With `favourites`, the tree below the file's root SHALL be imported into the import folder following the bookmark import's rules for the folder structure, imported links, import order and duplicate URLs (see the `bookmark-import` capability). Each link keeps its URL, title, description, tags, creation time and, when the image is imported, its preview image. Each link also keeps its offline copy, with its capture time, title and the parts whose files are imported (see "Files from the archive"). A copy is `ready` when it has at least one imported part, and otherwise the link has no offline copy. A link skipped as a duplicate keeps the existing link's copy, and the file's copy is not imported. Imported links get the same background fetches as links of a bookmark import, so no offline copy is captured for them.

#### Scenario: Duplicate link
- **WHEN** Ann has `https://go.dev/` and imports an export with `https://go.dev` and `skipDuplicates` `true`
- **THEN** the link is skipped and counted as a duplicate, and references to it point to Ann's existing link

#### Scenario: Description and preview kept
- **WHEN** a ZIP export has a link with a description and a preview image
- **THEN** the imported link has that description and preview image

#### Scenario: Offline copy restored
- **WHEN** a ZIP export has the link "Go tour" with an offline copy with an article and a screenshot, captured on 2026-10-01
- **THEN** the imported link's offline copy is `ready`, captured on 2026-10-01, with that article and screenshot, and no capture is queued

#### Scenario: JSON export with offline copies
- **WHEN** a JSON export has links whose `offlineCopy` has `capturedAt` and no files
- **THEN** the imported links have no offline copy

#### Scenario: Export from before offline copies
- **WHEN** a version 1 export without `offlineCopy` on its links is imported
- **THEN** the links are imported without offline copies

### Requirement: Files from the archive
An image, attachment or offline copy part of a ZIP export SHALL be imported from the file named in its `file`, or in the `article` or `screenshot` of an `offlineCopy`. Images and attachments SHALL meet this server's rules for uploaded images and attachments (allowed types and maximum size). An article SHALL be HTML of at most `BOARDLY_OFFLINE_MAX_SIZE_MB` megabytes, and it SHALL be sanitized again like a captured article before it is stored (see "Readable article" in the `offline-copies` capability). A screenshot SHALL be a JPEG, PNG or WebP within the preview image limits, and it is converted like a captured screenshot. A file that doesn't meet these rules, that is missing from the archive, or, for images and attachments, whose `file` is `null`, SHALL be skipped and counted in the report. The items that use it are imported without it (see "References between imported items"). An offline copy part whose path is `null` is not imported and not counted, because the copy may simply not have that part. A JSON export has no files, so no image, attachment or offline copy part is imported from it.

#### Scenario: Attachment type not allowed here
- **WHEN** a ZIP export has a note with a `.docx` attachment and this server doesn't allow that type
- **THEN** the note is imported without the attachment, and the report counts one skipped file

#### Scenario: Missing file
- **WHEN** a ZIP export lists an image whose file isn't in the archive
- **THEN** the image is skipped and counted in the report

#### Scenario: Article with a script
- **WHEN** a ZIP export's offline copy article was edited to contain `<script>`
- **THEN** the imported article contains no script

#### Scenario: Screenshot that isn't an image
- **WHEN** an offline copy's `screenshot` file is a text file
- **THEN** the screenshot is skipped and counted in the report, and the copy is imported with its article only

### Requirement: Data import report
A successful import SHALL return a report with, for each imported part:

- `favourites`: the bookmark import's counts of links, folders, duplicates, invalid URLs and shortened descriptions, and the number of created tags,
- `dashboards`: the number of added dashboards and of skipped widgets,
- `notes`: the number of imported notes and created categories, and of duplicate notes,
- `media`: the number of added media sources, of duplicates and of invalid ones,

and `files`: the number of imported images, attachments and offline copy parts, and of skipped files. A part that wasn't imported is `null`.

#### Scenario: Report of a full import
- **WHEN** an export with 10 links, 2 dashboards, 5 notes and 3 stations, one of them already saved, is imported with all parts
- **THEN** the report has 10 links, 2 dashboards, 5 notes, 2 media sources and 1 duplicate media source

#### Scenario: Offline copy files counted
- **WHEN** a ZIP export with one link whose offline copy has an article and a screenshot is imported with `favourites`
- **THEN** the report's `files` counts 2 imported files
