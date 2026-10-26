# Design

## Context

`favourites.Store.Import` loads the comparable URLs of all the user's links into `importer.urls` and skips every link whose comparable URL is in that set, adding each imported URL to it. The HTTP handler `importFavourites` reads `format` and `folderId` from the query and passes the parsed tree to `Import`.

## Goals / Non-Goals

**Goals:**
- One switch that turns the whole duplicate rule (tree and within-file) on or off.

**Non-Goals:**
- Changing how URLs are compared (`SameURL`/`comparableURL`).
- Changing folder reuse on import.

## Decisions

- **`Import` gets a `skipDuplicates bool` argument.** It's a property of the call, not of the tree, so it doesn't belong in `ImportTree`. When false, `Import` doesn't load the user's URLs and `addLink` doesn't look them up. Alternative: an options struct; with one flag a plain bool is simpler and matches the store's other methods.
- **Parse with `strconv.ParseBool`? No: accept exactly `true` and `false`.** `ParseBool` also accepts `1`, `t`, `TRUE` and so on, which the spec doesn't allow. A missing parameter means `true`, so existing clients keep today's behaviour.
- **Check the parameter before parsing the file**, next to the other request checks, so a bad value fails fast with `invalid_request`.
- **Frontend: a plain checkbox in the form**, state `skipDuplicates` defaulting to `true`, always sent explicitly in the query. Changing it resets the last result like the other fields do.

## Risks / Trade-offs

- [A user unchecks the box and imports the same file twice, doubling the links] → The checkbox is on by default, and bulk delete with undo exists for cleanup.
