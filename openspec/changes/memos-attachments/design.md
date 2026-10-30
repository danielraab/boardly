# Design

## Context

Boardly keeps attachments as `attachment` blocks inside the note document (`note-document`), and the note's references are updated in the same transaction as every content change (`note-attachments`). Memos keeps them as a separate list next to the text. Today `internal/memos` returns `attachments: []` and rejects any upload or non-empty list; attachments only reach MoeMemos as Markdown links in `content`.

What MoeMemos actually does (`MemosV1Api.kt`, `MemosV1Repository.kt`, `SyncingRepository.kt` in mudkipme/MoeMemosAndroid, October 2026):

- **Upload:** `POST api/v1/attachments` with a streamed JSON body `{filename, type, content: <base64>, memo}`. `memo` is only set when the memo already exists on the server, and an upload is always followed by a memo create or update.
- **Memo push:** each create and update sends the full memo: `content`, `pinned`, `state`, `updateTime` and the complete `attachments` list as `[{name}]`. `updateMask` is sent only when a location is involved, and then it lists every field that is present.
- **Display:** an attachment's URL is `externalLink` when it is set, otherwise `<host>/file/<name>/<filename>`. After a push, MoeMemos drops local resources whose `name` isn't in the returned memo's `attachments`.
- **Removing a file:** MoeMemos pushes the memo without it, then calls `DELETE api/v1/attachments/{id}`.
- **Listing:** the attachment list screen reads the local database. The remote `GET api/v1/attachments` isn't needed for the sync.

## Goals / Non-Goals

**Goals:**
- Photos and files added in MoeMemos show up in the Boardly note, and attachments added at the end of a note in Boardly show up in MoeMemos' gallery.
- Round trips are stable: pushing an unchanged memo changes nothing.

**Non-Goals:**
- Serving files under Memos' `/file/attachments/…` path. `externalLink` is enough for the MoeMemos we target.
- New attachment types such as HEIC. Android's camera saves JPEG, and type support belongs to `note-attachments`.
- Memos "link" attachments (an `externalLink` without content).
- Placing uploaded files anywhere except the end of the note.

## Decisions

### Trailing run split (option B)

The memo's `attachments` are the attachment blocks after the note's last non-attachment block. On read, `content` is the Markdown of the document with that run cut off. On write, `blocks(content) + run(attachments)`.

- *Alternative A, inline only:* keep attachments in `content`. MoeMemos would show raw URLs in its editor and have no gallery, and uploads would still need a mapping.
- *Alternative C, every block in the list:* lossy. Editing a memo would move inline images to the end.

The split is a pure function on `notes.Doc`: `SplitTrailingAttachments(doc) (body Doc, ids []string)` and `JoinTrailingAttachments(body, ids)`, in `internal/notes` next to `WithTitleHeading` and `TitleFromHeading`. Memo `content` becomes `ToMarkdown(WithTitleHeading(title, body))`. Duplicate IDs in the run are reported once, and on write each listed ID is appended once.

### Update merging

`patchOf` builds the new document from up to three parts:

```
body  = content given ? ReplaceContentFromMarkdown(oldBody, content) : oldBody
run   = attachments given ? listed IDs : oldRun
doc   = Join(body, run)
```

`ReplaceContentFromMarkdown` gets the *old body*, without the run, so task IDs are matched as today. "Content equal to the current content" now compares against the split content. The patch has content only when the joined document differs from the stored one, so an unchanged push is not a change, even though MoeMemos always sends both fields.

### Resolving attachment names

`attachments/<id>` must name one of the owner's attachments. The note store's document validation already rejects unknown IDs with `ErrInvalidDocument`, but that is currently mapped to "The memo is too large." So the memos service checks the IDs first with one query and responds 400 with "Unknown attachment." It reuses `UserMarkdownAttachments`, which is already loaded for content parsing. That keeps the error message accurate and doesn't depend on the store's error.

### Upload body

The body is JSON with base64 inside. Decoding it with `encoding/json` into a struct whose `content` field is `[]byte` decodes standard base64 directly. The body limit is `base64len(MaxBytes) + 64 KB` through `http.MaxBytesReader`, instead of `maxNoteBytes`. The decoded file then goes through `attachments.Store.Create`, which applies size, name and type rules. The file is held in memory once as base64 and once decoded. At the default 5 MB that's fine; a streaming JSON decoder isn't worth it now.

URL-safe base64 is accepted as a fallback: decode `content` as a string, try `StdEncoding`, then `RawURLEncoding`. Protobuf JSON accepts both.

Errors map to Memos errors: `ErrTooLarge` gives 400 code 3 "Attachments can be at most N MB.", `ErrUnsupported` gives "This type of file can't be attached.", and `ErrInvalidName` gives "Invalid file name.".

### Memo for an attachment

`MemoOf` needs each trailing attachment's name, content type, size and creation time. `notes.Note.Attachments` (`NoteAttachment`) already has the first three. Add `CreatedAt time.Time` with the tag `json:"-"`, and select it in `noteAttachmentsColumn`. `NoteAttachment` is part of the web API's note JSON, and that JSON stays unchanged.

`externalLink` is built with the existing helper used for Markdown URLs (`origin + attachments.URL(id, name)`), so the two can't drift apart.

### Listing and deleting in scope

The list needs "attachments referenced by notes in scope". `ScopedNotes` already returns the in-scope notes with their `Attachments`. The handler collects and dedupes them, then sorts by creation time descending, then by ID. No new SQL is needed.

Delete checks two things: the attachment belongs to the owner (`attachments.Store.Get`), and its `NoteIDs` are either empty or include a note in scope. It responds `{}` and never deletes. Deleting for real would break other notes that use the file, and the file's lifecycle already belongs to the cleanup.

## Risks / Trade-offs

- **[A memo whose text ends with an attachment link]** Text that the user typed in MoeMemos ending in a Boardly attachment URL becomes a trailing block, and on the next sync it moves from the text into the gallery. → This is the defined behaviour. It's rare, and the file is still shown.
- **[Files added at the cursor in the web app]** These land inside the content when the user keeps typing after them, so MoeMemos shows them inline as links instead of in the gallery. → Accepted. Positions are kept, which is the point of option B.
- **[Upload followed by a failed memo push]** This leaves an unused file. → It's marked after a day and deleted by the cleanup, like an abandoned web upload.
- **[Large uploads in memory]** → The limit is applied before decoding, and the default maximum is small.
- **[Older MoeMemos without `externalLink` support]** These would build `/file/…` URLs and get 404. → Out of scope. Can be added later as one route.

## Migration Plan

No data migration. Notes that already end with attachments start showing them in MoeMemos' gallery after the next sync, and those attachments disappear from the memo text. Rollback: redeploy the previous version. Attachments uploaded in the meantime remain normal Boardly attachments.
