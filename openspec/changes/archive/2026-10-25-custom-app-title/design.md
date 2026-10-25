# Design

## Context

See proposal.md for why. The frontend is built once by Vite into `backend/internal/web/dist` and embedded into the binary. `internal/web` serves those files, with `index.html` as the fallback for client-side routes (already `no-cache`). The manifests live in `frontend/public/` and are copied to `dist` unchanged. The product name is hardcoded in `index.html`, both manifests, `usePageTitle.ts`, `AppShell.tsx`, `Landing.tsx`, three i18n strings per language and the four magic-link templates.

Staging builds the Dockerfile on every deploy, so a build-time value would also work for it. But the emails are rendered by the Go server, and configuration is done only through `BOARDLY_*` variables. A single runtime setting covers both sides and also works with the prebuilt image.

## Goals / Non-Goals

**Goals:**
- One setting that feeds the HTML document, the manifests, the web app and the sign-in email.
- The right title on first paint, without waiting for an API call.

**Non-Goals:**
- Making the title configurable in the Vite dev server: in development, `index.html` comes from Vite, so the app shows "Boardly". That's acceptable for a staging/branding setting.
- Translating the app title per language. It's one value for every language.

## Decisions

### Runtime env var, not a Docker build arg
`BOARDLY_APP_TITLE` is read in `internal/config` like every other setting. A build arg (`VITE_APP_TITLE`) would be less code on the frontend side, but the mail templates would need a second way in (ldflags or an env var anyway), and the published image couldn't be renamed.

### Server fills in the HTML document and the manifests
`web.Handler` takes the app title. When the handler is created, it reads `index.html` and the two manifests from the embedded FS once and builds their final bytes. Requests serve those bytes from memory.

- `index.html`: the source contains `<title>Boardly</title>` and `<meta name="application-name" content="Boardly" />`. The handler replaces exactly these two strings with the `html.EscapeString`-escaped title. If a marker is missing (e.g., because the build changed the markup), that's a programming error: a Go test on `NewHandler` with a fixture FS covers the replacement, and a frontend test (or a check in the existing `index.html` test, if any) pins the two markers.
- Manifests (`*.webmanifest`): decode into `map[string]json.RawMessage`, set `name` and `short_name`, and encode again. Key order changes, which doesn't matter for manifests. JSON encoding takes care of escaping.
- Both are served with `Cache-Control: no-cache`. The manifest's URL isn't hashed, so without it a browser could keep the old name.

Alternative considered: a `GET /api/config` endpoint that the frontend fetches. Rejected: the tab would first show "Boardly", and the manifest (installed app name) would stay "Boardly".

### Frontend reads the meta tag
A small module (`src/lib/appTitle.ts`) exports `appTitle()`, which reads `document.querySelector('meta[name="application-name"]')?.content`, trimmed, with the fallback "Boardly". Reading one meta tag is cheap, and a function lets tests set the tag; no context or hook is needed. `usePageTitle`, `AppShell` and `Landing` import it. Tests set the meta tag (or rely on the fallback).

### i18n through interpolation
The three strings get `{{appName}}`, and their call sites pass `{ appName: appTitle() }`. A global i18next default variable (`interpolation.defaultVariables`) was considered: it would save the call sites, but it is fixed when i18next initializes, before tests can set the meta tag, so a custom title couldn't be tested. Passing it explicitly keeps it testable.

### Mail gets the title via `MagicLinkData.AppName`
`auth.Service` gets an `AppTitle` field (set from config in `main.go`) and passes it into `MagicLinkData`. Templates use `{{.AppName}}`; `html/template` escapes it in the HTML part. With "Boardly", the golden files stay byte-for-byte the same.

## Risks / Trade-offs

- [Vite changes the markup of `index.html` and the replacement silently stops working] → The replacement only targets exact strings that Vite leaves alone (it doesn't minify HTML by default). The handler test plus one test against the real `dist/index.html` (skipped when the frontend isn't built) catch it.
- [Long titles overflow the header on narrow screens] → No length limit in the spec. The header link should truncate with an ellipsis; check this during implementation.
- [Installed PWAs keep their old name] → Browsers update the installed name only at their own pace, and some ask the user. `no-cache` makes sure they see the new manifest; the rest is up to the browser.

## Migration Plan

No migration. Without `BOARDLY_APP_TITLE`, everything stays "Boardly". On staging, set `BOARDLY_APP_TITLE=Boardly Staging` and redeploy. Rollback: remove the variable.
