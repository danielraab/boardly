# Proposal

## Why

The Karakeep mobile app and extension can save text bookmarks, but Boardly rejects them with `BAD_REQUEST` because it only stores links. Now that Boardly has notes, a text bookmark is simply a note. For that, an API key needs a part of the notes it may see, the same way it has a folder for links. Today an app's folder can't be changed after connecting, so moving it to another folder, or picking a category, means connecting the app again.

## What Changes

- **Root category for Karakeep API keys**: a Karakeep key has a root category, or the top level ("all notes"), besides its root folder. Existing Karakeep keys get the top level. Linkwarden keys have no notes access. Deleting a key's root category revokes the key, like deleting its root folder.
- **Editable connected apps** (**BREAKING** for the "tokens are immutable" rule): the root folder and, for Karakeep, the root category of an active key can be changed after creation, through `PATCH /api/app-tokens/{id}` and an "Edit" action on the Connected apps tab. Name, app, expiry and secret stay fixed.
- **Notes as Karakeep text bookmarks**: notes in the key's category subtree (all notes for the top level) are bookmarks with `content.type` `text` and their content as Markdown.
  - `createBookmark` with `type` `text` creates a note in the key's root category (uncategorized for the top level).
  - `getBookmarks`, `getBookmark`, `searchBookmarks` (title and text) and `users.stats` include notes; filtering by list or tag returns only links.
  - `updateBookmark` changes a note's title and text. Tasks are matched by their text, so they keep their IDs; only new tasks get new IDs.
  - `deleteBookmark` deletes a note.
  - Notes have no tags and aren't in lists: tagging a note or adding it to a list fails with `BAD_REQUEST`, and `getListsOfBookmark` returns no lists for it.
- **Note content as Markdown**: the server converts a note document to Markdown, which then reads back to the same document.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `app-tokens`: root category on tokens, editing root folder and category, list and dialogs, notes in the token scope.
- `karakeep-api`: text bookmarks backed by notes in every bookmark procedure; lists, tags and statistics with notes.
- `note-document`: content as Markdown, and replacing content from Markdown while keeping task IDs.
- `note-categories`: deleting a category revokes the API keys rooted at it, with a warning.

## Impact

- Backend: migration adding `app_tokens.note_scope` and `app_tokens.root_category_id`; `apptokens` create/list/update and authentication carry the category scope; `notes` gets `ToMarkdown`, scoped queries and task-ID matching; `karakeep` service merges notes into bookmark listing, search, create, update, delete and stats; category delete revokes tokens.
- Frontend: Connected apps tab (category shown, "Edit" action and dialog), create dialog (category picker for Karakeep), delete-category dialog warning, en/de strings.
- API: `rootCategoryId` and `noteScope` on app tokens; new `PATCH /api/app-tokens/{id}`. Karakeep clients see text bookmarks.
- Depends on `notes-in-import-and-share` (Markdown converter, `CategoryPicker`).
