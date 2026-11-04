# Proposal

## Why

Many operators of a public Boardly instance must publish an imprint and a privacy policy. In
Austria and Germany, for example, the imprint must be easy to reach from every page. Boardly has
no admin area, and it shouldn't get one, so operators can't add these pages today.

## What Changes

- New optional configuration for two legal pages, the **imprint** and the **privacy policy**.
  Each page comes from exactly one source:
  - Markdown files in a directory named by `BOARDLY_LEGAL_DIR` (for example a read-only bind
    mount): `imprint.md` / `privacy.md` as the fallback, and optional language variants
    `imprint.de.md`, `imprint.en.md`, `privacy.de.md` and `privacy.en.md`.
  - An external URL from `BOARDLY_IMPRINT_URL` / `BOARDLY_PRIVACY_URL`, with no language variants.
  - A page with neither source doesn't exist.
- The server reads and checks the files once at startup and stops with an error naming the
  problem in these cases: a page has both a file and a URL, a file in the directory has an unknown
  name or language, a file is unreadable, or a URL isn't absolute http(s).
- New public pages `/imprint` and `/privacy` show the Markdown content in the visitor's language.
  When that language has no file, they fall back to the plain `<page>.md`, then to any other variant.
- New unauthenticated API route `GET /api/legal/{page}` returns a page's content for a language.
- The web app's HTML document says which legal pages exist and where they link to, so the links
  render without an extra request.
- A **slim footer on every page** of the app shell has "Imprint" and "Privacy" (only those that
  are configured) and "Open source · GitHub". External legal links open in a new tab.
- **The landing page's own footer is removed.** Its "Open source · GitHub" moves into the global
  footer. The GitHub link in the landing hero stays.
- Documentation: the README covers the new variables and the file layout, and an example privacy
  policy lists what Boardly processes.

## Capabilities

### New Capabilities
- `legal-pages`: Imprint and privacy policy from operator-provided Markdown files or external
  URLs. Covers the configuration, language variants with fallback, startup checks, the link
  targets in the HTML document, the `/imprint` and `/privacy` pages and the public API that serves
  their content.

### Modified Capabilities
- `app-shell`: New global footer with the legal links and the GitHub link. The legal pages get a
  page width (narrow).
- `landing-page`: The "Landing page footer" requirement is removed, because the global footer
  replaces it.
- `media-player`: The mini player bar must not cover the global footer.

## Impact

- Backend: `internal/config` (new variables and checks), a new package that loads the legal pages
  and turns their Markdown into note documents with the existing `notes.FromMarkdown`,
  `internal/web` (meta tags in `index.html`), `internal/httpapi` (public route) and `main.go` wiring.
- Frontend: `AppShell` (footer), `Landing.tsx` (footer removed), new routes `/imprint` and
  `/privacy` that render with the existing `NoteRenderer`, a small helper that reads the meta
  tags, and en/de translation keys.
- API: one new public, read-only route, documented in
  `backend/internal/httpapi/openapi/boardly.yaml`. The `api-docs` requirements don't list single
  routes, so that spec doesn't change.
- Deployment: optional new env vars and an optional read-only volume. Without them, nothing
  changes apart from the footer.
- No new dependencies. goldmark is already used.
- Export and import are unaffected: legal pages are instance configuration, not user data.
