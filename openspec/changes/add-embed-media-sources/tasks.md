# Tasks

## 1. Provider registry and store

- [x] 1.1 Add `backend/internal/media/embed.go` with the provider registry and the `spotify` provider (playlist and album links, language segment, query string dropped, `spotify:` URIs, canonical URL, player address, oEmbed address, shape `audio`) and `ErrEmbedUnsupported`; verify with table tests for accepted links, tracks, `spotify.link`, other hosts, wrong ID lengths and `http`
- [x] 1.2 Add the oEmbed fetch over `images.GuardedTransport` with the 10 second and 64 KB limits, returning title and cover, and `ErrEmbedUnreachable`; verify with httptest cases for success, 404, oversize, timeout and a blocked private address
- [x] 1.3 Add `KindEmbed` to `Store.Create` and `Store.Update` with an injected metadata function next to `Check` and `FetchLogo`, and update every `NewStore` caller; verify with store tests for no title, own title, own icon, provider down with and without a title, and a changed link that keeps title and icon
- [x] 1.4 Add `provider`, `embedUrl` and `shape` to `Source` in `scanSource`, omitted for streams; verify with a test that a stream's JSON is unchanged and an embed's JSON has all three

## 2. API and public dashboards

- [x] 2.1 Make `title` optional for embeds in `createMedia` and map `embed_unsupported` and `embed_unreachable` in `mediaError`; verify with httptest cases for 201 without a title and both errors
- [x] 2.2 Filter sources of kind `embed` for non-owners in the `Resolve` of `backend/internal/widgets/media.go`, before the empty check, and add the embed fields to `mediaItem`; verify with tests for a mixed widget, a widget of only embeds, another signed-in user, and the owner
- [x] 2.3 Update `backend/internal/httpapi/openapi/boardly.yaml`: `kind` values on `POST /media` and `MediaSource`, optional `title`, `provider`, `embedUrl` and `shape`, the new error codes, and the media widget's data if it is documented; verify `TestOpenAPIDocumentsMatchRoutes` passes and review the schemas against the handlers by hand

## 3. Export and import

- [x] 3.1 Accept `KindEmbed` in `Store.InsertTx` through the registry without any network request, and count `ErrEmbedUnsupported` as invalid in `importMedia`; verify with import tests for a valid embed, an embed with a link no provider accepts, and a duplicate URL
- [x] 3.2 Confirm that the export needs no change and keep its format version; verify with an export test that an embed source round-trips through export and import with its kind, title, URL and icon

## 4. Player and dock

- [x] 4.1 Add `embed` with `provider`, `embedUrl` and `shape` to `PlayableSource`, and the status `open`; make `play` stop the audio for an embed and close the embed for a stream or an episode, and make `toggle` and Media Session ignore embeds; verify in `player.test.tsx` for both directions of exclusion
- [x] 4.2 Add `EmbedDock` in the app shell next to `MiniPlayer`: plain iframe with the `allow` list, title and close button, 360 px floating from `sm` up and full width below it, 352 or 152 px high from `shape`, keyed on `embedUrl`, and with bottom space reserved on the page; verify with component tests for both shapes of the layout, that a rename doesn't remount the iframe, that a changed link does, and that no provider script is added to the document
- [x] 4.3 Add the consent notice and the `useOpenEmbed` helper, with consent per provider that lasts until a reload, the "Always open" checkbox that stores it in a guarded `localStorage`, and the link in the dock that withdraws it; verify with tests for first use, confirm, a reload with and without the checkbox, withdrawing, cancel while a station plays, and storage that throws

## 5. Library and widget UI

- [x] 5.1 Rename the dialog and texts to "Add media" and "No media yet", and add the "Embed" tab with the link field, optional name, hint and error messages, in en and de; verify in `MediaPage.test.tsx` for adding a link, both errors, and that the existing search tests still pass with the new wording
- [x] 5.2 Show embeds on `/media` with the music icon, the "Open" button and open mark, the "Link" label in the edit dialog, and closing the dock on delete; verify in `MediaPage.test.tsx` for open, activating an open embed, and deleting the open embed
- [x] 5.3 Update `MediaWidgetView` for embeds (music icon, accessible name, open mark, no effect in edit mode or when already open), the settings dialog's placeholder and public note, and "These media sources no longer exist"; verify with widget tests for each case, including that nothing is requested from Spotify before activation

## 6. Documentation and final checks

- [x] 6.1 Document embeds, the known limitation on phones, the cookie note for operators' privacy policies, and that embeds never appear on public dashboards in `README.md`, and add an entry to `CHANGELOG.md`; verify both mention the Spotify embed and the public dashboard rule
- [x] 6.2 Run `make check` and `make test` once when the work is done, one after the other, and verify both pass
- [x] 6.3 Check in a real browser on desktop and on a phone that a Spotify playlist and an album open in the dock, keep playing across pages, stop when radio starts, that the 152 px layout is usable, and that a public dashboard shows no trace of them; record what the phone plays (full tracks or previews) in `design.md`
