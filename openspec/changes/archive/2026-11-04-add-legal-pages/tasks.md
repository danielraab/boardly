# Tasks

## 1. Configuration

- [x] 1.1 Add `LegalDir`, `ImprintURL` and `PrivacyURL` to `internal/config`. Read them from `BOARDLY_LEGAL_DIR`, `BOARDLY_IMPRINT_URL` and `BOARDLY_PRIVACY_URL`, trim them, treat blank as unset, and reject URLs that aren't absolute http(s) with an error naming the variable. Verify with table tests in `config_test.go` (unset, blank, valid URL, `example.at/privacy`, `ftp://…`).

## 2. Loading the legal pages

- [x] 2.1 Create `internal/legal` with `Load(dir string, imprintURL, privacyURL string) (*Pages, error)`. It scans the directory, matches `^(imprint|privacy)(\.(en|de))?\.md$`, ignores dotfiles, subdirectories and names without `.md`, rejects files that are unknown, unreadable, blank or larger than 1 MiB, rejects a page with both files and a URL, and converts the content with `notes.FromMarkdown(src, nil)`. Each error names the file or variable. Verify with `legal_test.go` against `t.TempDir()` for every scenario in "Legal configuration is checked at startup", including the ignored files.
- [x] 2.2 Add `Pages.Lookup(page, lang)`, which returns the doc and the language it is in (`""` for the fallback) in the order requested language → `<page>.md` → `en` → `de`, and `Pages.Href(page)`, which returns `/imprint` / `/privacy`, the URL or `""`. Verify with tests for every scenario in "Language of a legal page" and for unknown or unsupported `lang`.
- [x] 2.3 Add a test that raw HTML (`<script>alert(1)</script>`) in a legal file ends up as text in the doc. Verify that it passes.
- [x] 2.4 Call `legal.Load` in `main.go` after `config.Load`, so an error stops the server with a non-zero status. Verify with a `main_test.go` case that starts with an unknown file in `BOARDLY_LEGAL_DIR` and expects a failure naming it.

## 3. HTML document

- [x] 3.1 Switch `web.Handler` / `web.NewHandler` to a `web.Options` struct (`AppTitle`, `AnalyticsSnippet`, `MapStyleURL`, `ImprintHref`, `PrivacyHref`) and update every call site and test. Verify that `go test ./internal/web/... .` passes unchanged in behaviour.
- [x] 3.2 Insert `<meta name="boardly-imprint" content="…" />` and `<meta name="boardly-privacy" …>` (HTML-escaped) before `</head>` for each configured page, on `index.html` and the SPA fallback only. Verify with `web_test.go` cases: file page (`/imprint`), external URL with `&` (escaped), not configured (no tag), deep link `/settings`, and no tag in a JS bundle.

## 4. API

- [x] 4.1 Add the public route `GET /api/legal/{page}` (no `requireUser`) to `internal/httpapi`, with `Legal *legal.Pages` in `Deps`. It returns `{"page", "language" (null for fallback), "doc"}` for file pages, and `404 not_found` for unknown, unconfigured and external pages. Verify with `legal_test.go` in httpapi: signed-out `200` with `lang=de`, fallback with `language: null`, external `404`, `GET /api/legal/terms` `404`.
- [x] 4.2 Document `GET /api/legal/{page}` in `backend/internal/httpapi/openapi/boardly.yaml`, marked public, with the `page` path enum, the `lang` query parameter, the response schema (reusing the note document schema) and the `404` error. Verify that `TestOpenAPIDocumentsMatchRoutes` passes.

## 5. Frontend: legal pages

- [x] 5.1 Add `lib/legalLinks.ts`, which reads the `boardly-imprint` / `boardly-privacy` meta tags and returns each page's href or nothing, and tells internal hrefs from external ones. Verify with `legalLinks.test.ts` (missing tag, `/imprint`, external URL, blank content).
- [x] 5.2 Add an API client function and `features/legal/LegalPage.tsx`. It shows the not-found component when the page isn't served from files (without calling the API), fetches with query key `["legal", page, language]`, shows loading and a translated error with retry, renders `NoteRenderer` read-only at the narrow page width, and sets the document title from `legal.imprint` / `legal.privacy` with `usePageTitle`. Verify with `LegalPage.test.tsx`: content rendered, refetch on language switch, not-found for missing and external pages, document title "Impressum - Boardly", error state.
- [x] 5.3 Add the file routes `routes/imprint.tsx` and `routes/privacy.tsx` and regenerate `routeTree.gen.ts`. Verify that `/imprint` and `/privacy` render `LegalPage` in a route test and that `pnpm build` succeeds.

## 6. Frontend: global footer

- [x] 6.1 Add `components/AppFooter.tsx` with the legal group (only configured pages; internal links via router `Link`, external ones in a new tab with `noopener noreferrer`, the `ArrowUpRight` marker and the "(opens in new tab)" name) and "Open source · GitHub" (GitHub mark, marker, accessible name, fixed repository URL). It wraps at narrow widths. Verify with `AppFooter.test.tsx` for every "Global footer" scenario that can be checked in jsdom (both from files, external privacy, nothing configured, GitHub link name and URL with a custom app title).
- [x] 6.2 Render `AppFooter` in `AppShell` between `main` and `MiniPlayer`, and reduce `main`'s bottom padding. Verify in an `AppShell` test that the `contentinfo` landmark is present on the landing page, the login page and a signed-in page, and that it comes before the mini player's spacer in the DOM.
- [x] 6.3 Remove the footer from `features/landing/Landing.tsx` and its test ("ends with an 'Open source · GitHub' footer"), and keep the hero's GitHub link. Verify that `Landing.test.tsx` passes and the landing page has exactly one `contentinfo` landmark.

## 7. Translations

- [x] 7.1 Add en/de keys for `legal.imprint` ("Imprint" / "Impressum"), `legal.privacy` ("Privacy" / "Datenschutz"), the legal page's loading and error texts, and the footer (`shell.footer.openSource`, moved from `landing.footer.openSource`). Remove the unused landing key. Verify that `i18n.test.ts` (same key set in both catalogs) passes.

## 8. Documentation

- [x] 8.1 Document `BOARDLY_LEGAL_DIR`, `BOARDLY_IMPRINT_URL` and `BOARDLY_PRIVACY_URL` in the README configuration table. Add a "Legal pages (imprint and privacy policy)" section covering the file names and language fallback, the compose example with `./legal:/legal:ro`, "restart to apply", the supported Markdown subset, and that the Vite dev server shows no legal links. Verify by reading it through against the specs.
- [x] 8.2 Add an example directory `docs/legal-example/` with `imprint.md` and `privacy.md`, so it can be mounted as it is. The privacy example lists what Boardly processes (email address and sign-in, passkeys, session cookie, IP addresses in logs and rate limits, favicon and preview fetching, map tiles after consent, optional analytics snippet, account deletion and data export) and is marked as a template that the operator must adapt. Verify that the README links to them and that loading `docs/legal-example/` with `legal.Load` succeeds in a test.
- [x] 8.3 Add a CHANGELOG entry for the legal pages and the global footer. Verify that it is under the unreleased section.

## 9. Final check

- [x] 9.1 Run `make check` and `make test` and confirm both pass. In a built app (`make build` or `docker compose up --build`) with a mounted `legal/` directory, confirm by hand: the footer shows on landing, login, a dashboard and settings, `/imprint` switches content when the language changes, an external privacy URL opens in a new tab, and the mini player doesn't cover the footer.
