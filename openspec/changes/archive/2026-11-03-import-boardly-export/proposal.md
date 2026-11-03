# Proposal

## Why

Since `export-all-data`, users can download all their data as a Boardly export, but they can't bring it back: not into a new account, not onto another server, and not after deleting something by mistake. An export that can't be imported is only half a backup.

## What Changes

- New endpoint `POST /api/import` that imports a Boardly export (the JSON file or the ZIP with files, format `boardly` version 1) into the signed-in user's account. The body is the file itself, at most 10 MB, like the bookmark import.
- The import always **adds** to the account and never deletes or overwrites anything. Every imported item gets a new ID. On an empty account this amounts to a full restore.
- The user picks the **parts** to import: favourites (with tags), dashboards (with widgets), notes (with categories and attachments) and media sources. Images are imported when an imported item uses them.
- **References** between imported items are rewritten to the new IDs: widget configurations, note content and attachments, icons, link previews, parent categories. A reference to an item that wasn't imported in the same run is kept as it is, so it still works when restoring into the same account. An icon whose image isn't available is removed.
- **Merge rules:** favourites and notes follow the bookmark import's rules (import folder and category, folders and categories reused by name, duplicates skipped by the same checkbox). A skipped duplicate or reused folder counts as the target of references to it. Dashboards are always added, are never public and never the default, and get a free slug (`home` → `home-2`). Media sources with a stream URL the user already has are skipped. The default folder and note category are set from the export only when the user has none.
- Files in a ZIP must meet this server's rules for images and attachments. A file that doesn't is skipped and counted in the report.
- The import section in **Settings → Import / Export** is renamed "Import" / "Importieren", gets the format "Boardly export (JSON or ZIP)" and, for it, a checkbox per part. A file over 10 MB shows a message with the limit and, for the Boardly format, the hint to export without files.

## Capabilities

### New Capabilities
- `data-import`: The import of a Boardly export: the endpoint and its errors, the parts, ID rewriting and references, the merge rules per part, files from the ZIP, the report, and the Boardly options in the import form.

### Modified Capabilities
- `bookmark-import`: The "Import form" requirement gets the new title "Import" / "Importieren", the format "Boardly export (JSON or ZIP)" and a file input that also accepts `.zip`. When that format is chosen, the form follows `data-import`.

## Impact

- Backend: new import code for the Boardly format (reading JSON or ZIP, building the ID map, rewriting references), extensions of the favourites and notes importers so they report the new ID of each source item and keep more fields (descriptions, preview images, icons, colours, pinned, attachments), transaction-aware ways to store images, attachments, dashboards with layouts and media sources, a new route.
- Frontend: import section with the new format, part checkboxes, the new endpoint, the report and the size message; en/de translations.
- README: import of a Boardly export.
- No database migration, no new dependencies, no new configuration.
