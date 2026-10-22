# Proposal

## Why

The project is renamed from "Famodash" to "Boardly" (GitHub issue #22). "Boardly" is shorter, easier to say and spell, and describes the product (a dashboard start page) without needing the "favourites and more dashboard" explanation. The project is still young, so a clean rename now is cheaper than one later.

## What Changes

- Use "Boardly" as the product name in the web app (header, landing page, page title, translations, PWA manifests), in sign-in mails and in the Karakeep/Linkwarden-compatible APIs wherever the product name is reported.
- **BREAKING** Rename all configuration variables from `FAMODASH_*` to `BOARDLY_*`. There is no fallback to the old names.
- **BREAKING** The default database file becomes `./data/boardly.db` (container: `/data/boardly.db`). Existing installs rename their file or set `BOARDLY_DB_PATH`.
- **BREAKING** Cookies are renamed (`boardly_session`, `boardly_login_request`, `boardly_webauthn`), so existing sessions end and users sign in again.
- **BREAKING** App token secrets start with `bdly_` instead of `famo_`. Existing app tokens stop working and must be created again.
- Browser storage keys move from `famodash.*` to `boardly.*` (theme, language, explorer state, share folder); stored preferences reset once.
- Rename the binary, Go module, npm package, container image, compose project, service and volume, Make targets, CI tags and dev scripts.
- Update README (with a migration note for existing installs), CLAUDE/OpenSpec project config and the live specs. Archived OpenSpec changes stay as they are, as a historical record.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `app-hosting`: configuration variables use the `BOARDLY_` prefix, default database file `boardly.db`, binary name `boardly`.
- `app-shell`: the global header shows the name "Boardly".
- `app-tokens`: token secrets start with `bdly_`; the product name shown in the create dialog and reported to Karakeep is "Boardly".
- `auth-magic-link`: the development mail fallback names the `BOARDLY_` variables.
- `auth-passkey`: passkey requirements name Boardly and `BOARDLY_` variables.
- `auth-session`: the session cookie is `boardly_session`.
- `images`: image storage and private address protection name `BOARDLY_` variables.
- `karakeep-api`: requirements refer to Boardly instead of Famodash.
- `landing-page`: the landing page presents Boardly.
- `linkwarden-api`: requirements refer to Boardly instead of Famodash, including reported names.
- `pwa`: the installed app and manifests are named Boardly.
- `theming`: the theme choice is stored under `boardly.theme`.

## Impact

- Backend: Go module path and all imports, config variable names, cookie names, token prefix, mail templates and golden files, API responses that carry the product name, tests.
- Frontend: package name, translations, `index.html`, manifests, storage keys, tests, icon render script.
- Ops: Dockerfile, `compose.yaml`, Makefile, `.env.example`, `scripts/`, CI workflow, container image name `ghcr.io/danielraab/boardly`.
- Existing deployments must rename env vars and the DB file, sign in again and recreate app tokens (documented in README).
- The GitHub repository itself is renamed outside this change.
