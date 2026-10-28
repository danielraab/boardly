# Proposal

## Why

Boardly has notes now, but the two ways content gets into Boardly from outside still only know links. The Karakeep import drops text bookmarks, which are notes in all but name, and sharing plain text (a message, a selection) into the installed app ends up as the title of a link form with an empty URL. Both should create notes.

## What Changes

- **Markdown and plain text to note content**: the server can build a note document from Markdown (Karakeep text bookmarks) and from plain text (shared text). Unsupported Markdown (images, tables, HTML) is kept as text.
- **Notes API**: `POST /api/notes` also accepts `text`, plain text that the server turns into the content, as an alternative to `content`.
- **Default category for new notes**: a per-user setting in Settings → Profile, like the default folder for new links. `GET /api/me` returns it as `noteCategoryId`; deleting the category clears it.
- **Karakeep import creates notes**: a text bookmark becomes a note with its title, Markdown text, creation date, Karakeep note and summary, and its tags as a final `#tag #tag` line. The import takes an import category (`categoryId`, top level allowed), picked on the import form. Manual lists that contain imported notes are recreated as categories below it, reusing categories of the same title. With "skip duplicates", a note with the same title and content as an existing note or an earlier one in the file is skipped. The report counts notes, categories and duplicate notes. Text bookmarks are no longer counted as "not a link".
- **Share page saves notes**: `/share` gets a switch between "Link" and "Note". Without a URL in the shared data it starts as a note. The note form has a title, the text and a category, preselected from the default category.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `note-document`: new requirement for building content from Markdown and from plain text.
- `notes`: the Notes API accepts `text` on create.
- `user-profile`: new default category for new notes, with validation.
- `bookmark-import`: import category, Karakeep text bookmarks as notes, lists as categories, duplicate notes, report and form changes.
- `pwa`: the share page can save the shared data as a note.

## Impact

- Backend: new Markdown/plain-text converter in `internal/notes` (new dependency `github.com/yuin/goldmark`); `importer` returns notes besides the link tree; the import runs links and notes in one transaction across `favourites` and `notes`; `POST /api/notes` accepts `text`; migration adding `users.note_category_id`; `PATCH /api/me` validates it.
- Frontend: import form (category picker for Karakeep, report fields), Profile tab (default category form), share page (link/note switch, note form), en/de strings.
- API: new query parameter `categoryId` and report fields on `POST /api/favourites/import`; new field `noteCategoryId` on `/api/me`; new field `text` on `POST /api/notes`. All additive.
- Follow-up change `karakeep-text-bookmarks` builds on the converter to serve notes through the Karakeep API.
