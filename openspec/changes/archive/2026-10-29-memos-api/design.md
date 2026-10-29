# Design

## Context

Connected apps follow one pattern. `httpapi.appAPI(app, mux)` adds CORS, answers preflights and puts the authenticated `apptokens` token into the request context. Each app has its own package (`karakeep`, `linkwarden`) with a `Service` over the favourites and notes stores, and a mux mounted under `/api/<app>/`. Notes in a token's scope are read through `notes.Store.ScopedNotes` and `ScopedNote` with a `notes.Scope` built from the token's note scope. Karakeep's `createNote` and `updateNote` already do Markdown in and out with `notes.ToMarkdown` and `notes.ReplaceContentFromMarkdown`, keeping task IDs.

The `app_tokens` table allows `app IN ('karakeep', 'linkwarden')`; `root_id` is already nullable (it becomes `NULL` when the folder is deleted). The token list in the frontend treats a `null` root as "Deleted folder".

MoeMemos (Android, `MemosV1Api.kt`) detects the API by calling `api/v1/status` (legacy) and then `api/v1/instance/profile`. It only accepts versions 0.27.0–0.31.0. It sends `Authorization: Bearer` to the configured host, syncs by listing every memo (`pageSize=200`, `state`, `filter creator == "users/<id>"`), and treats a memo as changed remotely when its `updateTime` differs from the last one it saw.

## Goals / Non-Goals

**Goals:**
- MoeMemos can log in, sync, create, edit, tick off tasks, pin, archive, unarchive and delete, with Boardly notes as the source of truth.
- Reuse the Karakeep note code paths and the app token plumbing. No new storage.

**Non-Goals:**
- Attachments (next change), public or protected memos and the explore feed, memo relations and comments, reactions, Memos' own web API beyond what MoeMemos calls, and a CEL filter engine.
- The legacy Memos API (MoeMemos' "V0" path for Memos 0.21–0.26).

## Decisions

**A new package `backend/internal/memos`**, shaped like `linkwarden`: a `Service` with the notes store and the base URL, plain REST handlers, and Memos' error type `{"code","message","details"}`. It is mounted with `appAPI(apptokens.AppMemos, …)` under `/api/memos/`. The preflight's allowed methods get `PATCH`, which all three apps can live with.

**Tokens: `memos` app with note scope and no root.** The migration `0021_memos_app_tokens.sql` rebuilds `app_tokens` (like `0013`) with the app check extended to `memos`, keeping `note_scope` and `root_category_id`. Creating a `memos` token stores `root_id` `NULL`, sets the note scope from `rootCategoryId` like Karakeep, and rejects a `rootId`. Considered: requiring a dummy root folder so existing code stays the same. Rejected because the dialog and the list would show a folder that means nothing. The frontend decides by app whether to show the folder, so `rootId` `null` on a Memos token is never shown as "Deleted folder". Folder deletion never touches Memos tokens, because they have no root. Category deletion already revokes tokens by `root_category_id`, whatever their app.

**Memo ID is the note ID.** `name` is `memos/<note id>`. Note IDs are random strings without `/` or `|`, which MoeMemos uses as separators. Users are `users/<user id>`. The `users/{id}:getStats` path is a single segment for `ServeMux`, so the `users/{id}` handler splits off a `:getStats` suffix itself.

**Title as a leading H1.** Output: `# <title>\n\n` + `ToMarkdown(content)`, or only the content without a title. Input: parse the Markdown into a document. If the first block is a level-1 heading whose plain text is 1–200 characters, it becomes the title and is removed from the blocks; otherwise the title is cleared. For updates, the remaining Markdown goes through `ReplaceContentFromMarkdown` against the old content, so task IDs stay. A content string equal to the memo's current content short-circuits to "no change", like Karakeep's text check, so that MoeMemos re-saving a memo doesn't bump versions. Edge case: a note without a title whose content starts with an H1 gets that heading as its title the first time MoeMemos saves it. That's acceptable and matches what the user sees in MoeMemos.

**Last write wins for MoeMemos.** Memos' PATCH has no version, so the handler reads the note's current version and patches against it, retrying once if the version changed between the read and the write. The web editor then sees `version_conflict` on its next save and shows "changed elsewhere". That is the existing behaviour for concurrent edits.

**Creation time from the app.** `notes.NewNote` gets an optional `CreatedAt` (the import path already sets one through `ImportNote`), clamped to now when it is in the future. The time of last change is the write time, so MoeMemos picks up the server's `updateTime` after pushing.

**Times with millisecond precision.** Memos writes RFC 3339 times. Boardly writes stored times in UTC truncated to milliseconds, always the same way, so MoeMemos' equality check on `updateTime` is stable. Seconds would be too coarse: two edits in the same second would look like one.

**Listing and paging.** Scoped notes are filtered by `state` (the archived flag), sorted by creation time descending, then by ID. They are paged with an opaque `pageToken`, a base64url-encoded offset. Offsets can skip or repeat a memo when notes are created during a sync. MoeMemos tolerates that, and the next sync fixes it. The filter parser splits on `&&` and recognises two patterns with regular expressions: `creator == "users/<id>"` and `visibility in [...]` or `visibility == ...`. Every other condition is ignored, so new MoeMemos filters degrade to "everything in scope" instead of an error.

**Attachments answer but don't store.** `GET` returns an empty list, because MoeMemos lists attachments during sync and an error there would block the sync. Uploads and memos with attachments fail with code 3 and a message, so the user sees why the photo didn't go through.

**Logo.** The Memos logo from `usememos/memos` (MIT) is added to `appTokens/logos` with its source in the README table, like the other logos.

## Risks / Trade-offs

- [A later MoeMemos release raises its minimum version above 0.31] → Login then shows "unsupported version". The version string is one constant; bump it and check the API calls of the new release.
- [MoeMemos treats a note moved out of the token's category as deleted on the server] → It may then delete or re-upload its local copy, depending on its conflict handling. Documented in the README: moving a note out of the category removes it from the app.
- [Re-saving in MoeMemos normalises Markdown] → Boardly's Markdown writer is deterministic and the unchanged-content check avoids churn. Content that MoeMemos rewrites is read as Markdown, with the same losses as Karakeep text bookmarks.
- [Filters ignored too eagerly] → An unknown filter returns more memos than asked for. MoeMemos only sends the two recognised forms today.
- [Last write wins can drop a web edit in progress] → The web editor detects the conflict and keeps its text on screen until reload, as for two browser tabs.

## Migration Plan

The migration rebuilds `app_tokens` in a transaction, like `0013`, keeping all rows. Rolling back the binary: older code rejects nothing at read time, but `memos` tokens would fail its app check on authentication, so they stop working until the new version is back.
