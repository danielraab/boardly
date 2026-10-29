# Proposal

## Why

Boardly notes can be opened in the Karakeep app today, but only as text bookmarks, which doesn't make a good note app. [Memos](https://usememos.com) is an open-source note server, and MoeMemos is a good Android client for it, working offline and syncing in the background. If Boardly answers the Memos API, people can write and tick off their Boardly notes from MoeMemos on the phone.

## What Changes

- A new connected app **Memos**. Its tokens have a note scope like Karakeep tokens: all notes, or one root category with its subcategories. The category is optional and can be changed after creation. Memos tokens have **no root folder**, because Memos has no bookmarks.
- Boardly answers the Memos v1 REST API that MoeMemos uses at `<BOARDLY_BASE_URL>/api/memos`, with the bearer token as MoeMemos' access token. It reports itself as Memos `0.31.0`, the newest version MoeMemos supports.
- Memos are the notes in the token's note scope. A memo's content is the note as Markdown, with the note's title as a leading `# Title` heading. Saving a memo reads a leading level-1 heading back as the title, so changing the heading renames the note and removing it clears the title.
- A memo's `pinned` is the note's pinned flag, and its `state` `ARCHIVED` is the note's archived flag (from `note-pin-archive`).
- Visibility is always `PRIVATE`: Boardly has no public notes, so MoeMemos' explore list is empty. Tags, location and statistics are answered with empty values.
- Attachments aren't supported yet: listing them returns none and uploading fails with a clear message. They come in a later change.

## Capabilities

### New Capabilities
- `memos-api`: the Memos v1 endpoints that MoeMemos uses, their wire format, and how memos map onto the notes in a token's note scope.

### Modified Capabilities
- `app-tokens`: the app `memos` with a note scope and no root folder; create and edit validation, list, connected apps tab, create and edit dialogs; bearer authentication, CORS and scope rules extended to `/api/memos/`.

## Impact

- Backend: a migration that allows the app `memos` in `app_tokens`; `backend/internal/apptokens` validation; a new `backend/internal/memos` package; routes under `/api/memos/` in `backend/internal/httpapi`. It reuses the note store, the Markdown conversion and the scope code shared with Karakeep.
- Frontend: the create and edit dialogs and the connected apps list handle a third app with no folder; a Memos logo in `frontend/src/features/settings/appTokens/logos`; en/de strings.
- Depends on `note-pin-archive` for the pinned and archived flags.
- README: a "Memos-compatible server" section with MoeMemos setup.
