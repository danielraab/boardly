## Context

Boardly already speaks the Karakeep and Linkwarden APIs as a server, so the mapping between their bookmarks, lists or collections and tags and Boardly's favourites tree is already settled (`karakeep-api`, `linkwarden-api`). Users switching from those tools still have to move their existing bookmarks by hand. This change adds the reverse direction once, from an export file.

The current state that matters:
- `favourites.Store.Create` inserts one node per transaction, always sets `created_at` to now, and queues a favicon and a preview fetch per link (`queueFetches`, then `notifyFetches` after commit).
- The validation helpers (`NormalizeURL`, `NormalizeLinkTitle`, `NormalizeTags`, `NormalizeFolderName`, `NormalizeDescription`, `SameURL`/`comparableURL`) define the rules of link nodes.
- The settings page has URL-backed tabs (`profile`, `passkeys`, `apps`), and the "Connect app" dialog already picks a folder with `FavouritePicker mode="folder"`.
- `golang.org/x/net` is already a dependency.

The format findings below come from a real Karakeep export (JSON and HTML, 168 bookmarks, 19 lists) provided by the user. No Linkwarden export was available.

Karakeep JSON (verified):
```
{ "bookmarks": [ { "createdAt": 1792670482,            // Unix seconds
                   "title": "…" | null,
                   "tags": ["Heizung", "0-10V"],          // names
                   "lists": ["nkzc…", …],                  // list ids, may be several
                   "content": {"type":"link","url":"…"} | {"type":"text","text":"…"},
                   "note": "…" | null,
                   "archived": false } ],                  // sorted newest first
  "lists": [ { "id", "name", "description", "icon",
               "type": "manual" | "smart", "query": "#kotlin" | null,
               "parentId": "…" | null } ] }             // parent may follow child
```
There is no description, summary or favourited field in the sample. The parser still reads `summary` when present.

Karakeep HTML (verified): flat Netscape list, `<DT><A HREF ADD_DATE TAGS>`, no `<H3>`, no `<DD>`, no text bookmarks. The downloaded file ends with a `<script>` injected by Cloudflare's email obfuscation.

Linkwarden JSON (not verified): taken from Linkwarden's migration export (`GET /api/v1/migration`), which serializes the user with `collections[]`, each with `id`, `name`, `parentId` and `links[]`. Each link has `name`, `url`, `description`, `createdAt` (ISO 8601), `type` and `tags[]` of `{name}`.

## Goals / Non-Goals

**Goals:**
- One-time import of Karakeep JSON, Linkwarden JSON and Netscape HTML files into a user-chosen import folder, keeping structure, tags, creation dates and file order.
- Never create a duplicate URL in the user's tree.
- All or nothing, with a clear report of what was skipped.
- Parsers that are tolerant of unknown fields and junk, and testable without a database.

**Non-Goals:**
- Syncing, or pulling from a running Karakeep or Linkwarden instance over its API.
- Export. The tab reserves the place, and a later change will add it.
- Text and asset bookmarks, smart lists, and archived, favourited or pinned flags. These are possible once Boardly has notes or such flags.
- Importing preview images or archives from the source. Boardly fetches its own.
- Merging tags or descriptions into links that already exist.
- Undo, beyond deleting the import folder.

## Decisions

### File upload, not API pull
Import uses an uploaded file. A one-time migration fits a file: it works after the old instance is gone, it adds no outbound calls to user-given hosts, and it is easy to test with fixtures. Netscape HTML adds browsers as a source for the cost of one parser.
*Alternative:* URL plus API key of a live instance. That gives richer data, but needs two API clients, paging, key handling and SSRF exposure. Rejected for now.

### The user picks the format
The form sends `format` explicitly instead of the server sniffing it. That keeps the error clear ("not a Karakeep export") and the parsers independent. Sniffing could be added later without an API change, as an `auto` value.

### Two layers: parsers produce a common model, and the store applies it
```
 raw body ─▶ httpapi/import.go ─▶ importer.ParseKarakeep    ┐
                                    importer.ParseLinkwarden  ├─▶ favourites.ImportTree
                                    importer.ParseNetscape    ┘          │
                                                                         ▼
                                              favourites.Store.Import(ctx, user, folderID, tree)
                                                         │  one tx
                                                         ▼
                                                  favourites.ImportReport
```
- The model types live in `favourites` (`ImportTree`, `ImportFolder{Name, Items}`, `ImportLink{URL, Title, Tags, Description, CreatedAt}`, where `Items` is an ordered list of folders and links). That lets `importer` import `favourites` for its validation helpers without an import cycle.
- The new package `internal/importer` does everything that needs no database. It parses, resolves Karakeep and Linkwarden parent IDs into a tree, picks the first manual list, builds and cuts descriptions, cuts titles and folder names, drops invalid tags, and drops entries with invalid URLs. It returns the tree plus parse-time counts (`notLinks`, `smartLists`, `invalidUrls`, `shortenedDescriptions`).
- `Store.Import` does everything that needs the database. It checks the import folder, reuses existing folders by name, checks duplicates against the tree, and handles positions, `created_at`, tags and fetch jobs. It adds `links`, `folders` and `duplicates` to the report.
- Parsers are pure functions over `io.Reader` and are tested with synthetic fixtures.

*Alternative:* call `Store.Create` per entry. Rejected: one transaction per link breaks "all or nothing", it is slow for thousands of links, and it can't set `created_at`.

### Store.Import in a single transaction
1. Check that `folderID` is a folder of the user (root allowed). Otherwise return `ErrInvalidParent` or `ErrNotFound`, which the handler maps to `invalid_folder`.
2. Load the user's existing link URLs once into a set keyed by `comparableURL`. Each imported URL is checked against the set and then added to it, which also catches repeats within the file.
3. Walk the tree depth-first. For each folder level, load the existing child folders once (name compared via `strings.ToLower` after trimming) and the next position (`COUNT(*)` of children). Reuse a matching folder or insert a new one. Two source siblings with the same name hit the same entry because new folders are added to the per-level map.
4. Insert links with `created_at = CreatedAt` (or now), `updated_at = now`, and the next position. Then `setTags` and `queueFetches` within the same transaction, exactly as `Create` does.
5. Commit, then call `notifyFetches(true)` once.

Reusing `setTags` and `queueFetches` keeps tag matching and fetch behaviour identical to normal creation. Integer IDs for apps are assigned the same way as for `Create`, since the insert is the same.

### Keeping created_at
`created_at` is only set at insert, so importing it needs no migration, just a value other than now. Karakeep's cursor pagination (`createdAt`, `id`) and Linkwarden's sorting already handle arbitrary times. Folders get the import time, because Karakeep lists carry no creation time in the export.

### Order
Within a folder, the file's order is kept, and the imported nodes go after existing children. The Karakeep and Linkwarden parsers emit subfolders first (in `lists` or `collections` order), then links (in bookmark order). Netscape keeps document order. Karakeep exports are newest first, so the newest link ends up on top, which matches the apps.

### Netscape parsing with the x/net/html tokenizer
Netscape files are malformed HTML (unclosed `<DT>`, `<p>` after `<DL>`). The `html.Parse` tree builder would rearrange them, so the parser uses `html.NewTokenizer` with a folder stack:
- `<H3>…</H3>` remembers a pending folder name.
- `<DL>` pushes the pending folder, or a transparent level for the outer list.
- `</DL>` pops a level.
- `<A>` adds a link.
- The text of a `<DD>` directly after an `<A>` becomes that link's description, until the next `<DT>`, `<DL>` or `</DL>`.

Everything else, such as `<script>`, `<META>` and `<TITLE>`, is ignored. The file counts as a Netscape file when it has the `NETSCAPE-Bookmark-file-1` doctype or at least one `<DT>`. Entities are decoded by the tokenizer.

### Description composition
Parts are joined with `"\n\n"` in this order: source description, `Note: <note>`, `Summary: <summary>`. When the result is over `MaxDescriptionLength` runes, it becomes 2047 runes plus `…`. This is done in the importer, so the store only sees valid descriptions.

### Endpoint and limits
`POST /api/favourites/import?format=…&folderId=…` uses `requireUser`. The body is the raw file as `application/octet-stream`, read with `http.MaxBytesReader` set to 10 MB. Multipart was the first plan, but the API's CSRF defence (`checkCSRF`) only accepts JSON and octet-stream bodies, because form types can be posted cross-site without a preflight. The image upload already sends raw octet-stream for the same reason, so the import does too, and the defence stays unchanged. 10 MB is a constant: the 168-bookmark Karakeep export is 53 KB, so 10 MB covers tens of thousands of bookmarks. A `BOARDLY_*` setting can be added if someone hits the limit. Error codes are `invalid_request`, `invalid_folder`, `invalid_import_file` and `file_too_large`.

### Frontend
- `SettingsTab` gets `"data"`, and `routes/settings.tsx` accepts it.
- The new `features/settings/ImportExportTab.tsx` holds the import form. Format radio buttons reuse `AppLogo` for Karakeep and Linkwarden. The file input accepts `.json,.html,.htm`. The folder picker is `FavouritePicker mode="folder"` with no initial selection.
- `features/settings/import/api.ts` posts the selected `File` as the octet-stream body, with `format` and `folderId` in the query, and maps error codes to translated messages.
- On success, the favourites query (`favouritesQueryOptions`) is invalidated, along with any queries derived from the tree.
- Translations go under `settings.import.*` and `settings.tabs.data` in `en.json` and `de.json`.

## Risks / Trade-offs

- [The Linkwarden format is unverified] → The parser is lenient: it ignores unknown fields, accepts `tags` as objects or strings, and tolerates a missing `parentId`. The design records the assumed shape, and a task asks the user to verify with a real Linkwarden export before archiving.
- [The Karakeep format changes] → Unknown fields are ignored. Only `bookmarks[].content.url` is strictly required for a link to be imported.
- [A large import holds the SQLite write lock for a while] → All work is in one transaction without network calls. A few thousand inserts take well under a second. The 10 MB limit bounds the worst case.
- [A flood of fetch jobs after a big import] → Jobs are persisted and processed by the existing workers. Favicons are de-duplicated per origin, as for any link. No new mechanism is needed.
- [The "first list wins" order is arbitrary] → It follows the order of `lists` in the export, which is good enough. Only 1 in 168 sample bookmarks was in two lists.
- [Note text blocks the page-description fetch] → This is accepted by design: fetched values only fill empty fields.
- [Personal data in the sample files] → The user's exports are not committed. Fixtures are synthetic but copy the structure: nested lists with the parent after the child, a smart list, a text bookmark, a bookmark in two lists, a note, and a trailing `<script>`.
- [Case-insensitive folder reuse merges folders the user meant to keep apart] → This is accepted and was decided explicitly. Deleting the import folder undoes an import into a fresh folder.

## Open Questions

- Verify the Linkwarden parser against a real Linkwarden JSON export, and adjust the field mapping if needed.
