# Design

## Context

"Famodash" appears in roughly 70 files outside the archive: product name in UI and mails, the `FAMODASH_` config prefix, cookie and storage key names, the Go module name `famodash`, the `famo_` token prefix, and ops files (Dockerfile, compose, Makefile, CI).

## Goals / Non-Goals

**Goals:**
- No occurrence of "famodash" (any case) or the `famo_` token prefix remains outside `openspec/changes/archive/`.
- `make check` and `make test` pass after the rename.

**Non-Goals:**
- Backward compatibility with `FAMODASH_*` variables, the old DB filename, cookies, storage keys or `famo_` tokens.
- Rewriting archived OpenSpec changes.
- Renaming the GitHub repository (done by the owner in GitHub settings).
- A new logo; the existing app icon stays.

## Decisions

- **Clean break instead of fallbacks.** The project is days old with few installs. Fallbacks would keep the old name in the code forever. A README migration note covers existing installs: rename env vars, `mv famodash.db boardly.db` (or set `BOARDLY_DB_PATH`), sign in again, recreate app tokens.
- **Mechanical case-preserving replacement**: `FAMODASH` → `BOARDLY`, `Famodash` → `Boardly`, `famodash` → `boardly`. Applied to all files outside the archive and `.git`, then reviewed by diff. The Go mail import alias `famomail` becomes `boardlymail`.
- **Token prefix `bdly_`.** Same length as `famo_` (5 characters), so the 12-character public prefix still carries 7 random characters, and it is easy to recognise in leaks.
- **The "favourites and more dashboard" expansion is dropped**; Boardly is not an acronym.
- **Specs**: requirements that mention the name get MODIFIED deltas with the same text renamed. Purpose sections of live specs are edited directly, since deltas don't carry them.
- **DB migrations**: only comments mention the cookie names; they're updated. Migrations aren't checksummed, so editing comments is safe.

## Risks / Trade-offs

- Existing deployments break on upgrade until they're reconfigured → README migration note, BREAKING marks in the proposal.
- Mobile app/extension users must re-pair with a new token → documented.
- A blind replace could hit unintended strings (e.g. URLs to the GitHub repo) → review the full diff; GitHub redirects renamed repositories.
