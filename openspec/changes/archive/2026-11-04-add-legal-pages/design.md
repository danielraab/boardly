# Design

## Context

See proposal.md for the motivation. These parts of the current code shape the approach:

- **Configuration** comes only from `BOARDLY_*` variables, which `internal/config.Load` checks at
  startup. Invalid values stop the server with a message that names the variable.
- **`internal/web`** already rewrites `index.html` once at startup. It fills in the app title, the
  `boardly-map-style` meta tag and the analytics snippet by replacing markers. The frontend reads
  these meta tags synchronously (`lib/appTitle.ts`, `mapConsent.ts`).
- **`notes.FromMarkdown`** (goldmark: CommonMark plus task lists, strikethrough, tables and
  linkify) turns Markdown into the note document (`notes.Doc`). Raw HTML becomes text.
  **`NoteRenderer`** shows such a document read-only in the app's styles.
- **`AppShell`** is a `min-h-screen` flex column: header, then `main` (`flex-1`), then `MiniPlayer`.
  `MiniPlayer` adds an `h-20` spacer after the content while the bar is shown.
- The landing page has its own `<footer>` with "Open source · GitHub".

## Goals / Non-Goals

**Goals:**
- Operators configure the two legal pages with files or URLs alone. No UI, database or admin role.
- The pages are safe by construction: the operator's Markdown can't inject markup.
- A broken configuration fails at startup, not when a visitor opens the page.

**Non-Goals:**
- Editing the pages in the web app, or reloading changed files while the server runs.
- Language variants of external URLs.
- Legal pages other than the imprint and the privacy policy. The page set is fixed.
- Images, attachments or raw HTML in the legal content.
- Showing the legal links in the Vite dev server without a running Go server (see Risks).

## Decisions

### 1. A new `internal/legal` package loads everything at startup

`config` gets `LegalDir`, `ImprintURL` and `PrivacyURL`. It trims them and checks the URLs (absolute
http/https, the same check as `BOARDLY_MAP_STYLE_URL`). `legal.Load(dir, urls)` runs in `main`
after `config.Load`:

1. Read the directory entries. Skip dotfiles, subdirectories and names without `.md`.
2. Match each `.md` name against `^(imprint|privacy)(\.(en|de))?\.md$`. Any other name is an error.
3. For each file: stat it (more than 1 MiB is an error), read it, reject it when it is blank after
   trimming, and convert it with `notes.FromMarkdown(src, nil)`.
4. Check that no page has both files and a URL.

The result is an immutable `legal.Pages` value. For each page it holds either a URL or a map from
language (`""` for the fallback) to `notes.Doc`. Its methods are `Lookup(page, lang)`, which applies
the fallback order from the spec, and `Href(page)`, which returns `/imprint`, the URL or `""`.

*Alternatives:* reading the files on every request, or watching them, would let edits show without
a restart. That brings back runtime errors and caching questions for a file that changes about once
a year, and the analytics snippet already works with "restart to apply". Inline Markdown in env
vars is painful in compose files once it spans several lines (escaping, `$` interpolation).

### 2. Convert to the note document on the server and render with `NoteRenderer`

Reusing the notes pipeline means no new dependency on either side, no HTML sanitiser and no second
Markdown dialect to document. It also gives consistent typography for headings, lists, tables and
links. `FromMarkdown` is called with `att = nil`, so images stay as links. A `geo:` link alone in
its paragraph would become a location chip. That's harmless and unlikely in legal text, so it isn't
special-cased.

*Alternative:* render the Markdown to HTML with goldmark and inject it. That needs an HTML policy and
`dangerouslySetInnerHTML`, and it would look different from the rest of the app.

### 3. Link targets as meta tags in `index.html`, content through `GET /api/legal/{page}`

`web.Handler` gets the two hrefs and fills in `<meta name="boardly-imprint" content="…" />` and
`<meta name="boardly-privacy" …>`. When a page isn't configured, its tag is left out. Today's
`Handler` already takes positional strings, so this is a good time to switch it to an options struct
(`web.Options{AppTitle, AnalyticsSnippet, MapStyleURL, ImprintHref, PrivacyHref}`). A small
`lib/legalLinks.ts` reads the tags, like `appTitle()`.

The content is only needed on `/imprint` and `/privacy`. It is fetched there with TanStack Query,
keyed by `["legal", page, language]`. The route is public: it goes on the main mux without
`requireUser`, like `/api/health`. Response:

```json
{ "page": "imprint", "language": "de", "doc": { "...": "note document" } }
```

`language` is `null` for the fallback file. Errors use the shared format: `404 not_found` for
unknown pages, pages that aren't configured and external pages.

*Alternative:* embed the whole content in `index.html`. That costs nothing extra per request, but it
makes every page load bigger for a page that is rarely opened, and the escaping is fiddly.

### 4. Frontend routes and page

The new file routes `routes/imprint.tsx` and `routes/privacy.tsx` share one
`features/legal/LegalPage.tsx`, which gets a `page` prop. The page:
- shows the not-found component when `legalLinks()[page]` isn't `/imprint` or `/privacy` (missing
  or external), without calling the API,
- sets the document title from the translated name (`legal.imprint` / `legal.privacy`) with
  `usePageTitle`,
- uses the narrow page width, shows a loading state while fetching and a translated error with a
  retry when the request fails, and renders `NoteRenderer` with the default text size and no
  `onToggleTask` (read-only).

The page doesn't add its own `<h1>`, because the operator's Markdown brings its headings. The
document title still comes from the translated name, as the spec requires.

### 5. The global footer lives in `AppShell`, before `MiniPlayer`

`components/AppFooter.tsx` is rendered between `main` and `MiniPlayer`. Because of that order,
`MiniPlayer`'s existing spacer comes after the footer, and the bar never covers it. No new spacing
code is needed. `main` keeps `flex-1`, so the footer stays at the bottom on short pages. `main`'s
`pb-16` is reduced, because the footer now provides the space at the end.

Layout: the same `container` class as header and main, a top border (`border-line`), `text-sm
text-muted`, and `flex flex-wrap justify-between gap-x-6 gap-y-2`. The legal group is on the left and
"Open source · GitHub" on the right. External links reuse the pattern of the landing page's GitHub
link: `target="_blank" rel="noopener noreferrer"`, an `ArrowUpRight` marker and the
`common.opensInNewTab` suffix in the accessible name. The landing footer and the
`landing.footer.openSource` key move into `AppFooter`, renamed to `shell.footer.*`.

## Risks / Trade-offs

- [The Vite dev server serves `index.html` itself, so it has no `boardly-*` legal meta tags] →
  Like the map style and the analytics snippet, legal links show only when the app is served by Go
  (`make build` / the container). The README says so. The footer still shows "Open source ·
  GitHub" in dev.
- [The footer appears on every page and takes vertical space, for example on dashboards] → It is a
  single slim line after the content and never fixed. On long pages, users only see it when they
  scroll to the end.
- [Strict file-name checks can stop a server that used to start, for example after someone drops
  `impressum.md` into the directory] → The error names the file and the expected names, and the
  check only applies when `BOARDLY_LEGAL_DIR` is set.
- [Docker bind mounts of a single file can go stale when an editor replaces the file] → The README
  recommends mounting the directory (`./legal:/legal:ro`), not single files.
- [Note-document features (for example location chips) show up in legal content when the Markdown
  happens to use them] → This is accepted. The README lists the supported Markdown subset.

## Migration Plan

Nothing to migrate. All new variables are optional. Existing instances get the footer with "Open
source · GitHub" and no legal links until they configure some. To roll back, deploy the previous
image. The variables are then ignored.
