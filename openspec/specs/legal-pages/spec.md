# legal-pages Specification

## Purpose
Lets the operator of a Boardly instance publish an imprint and a privacy policy without an admin area: from Markdown files on disk, in several languages with a fallback, or as links to external pages.

## Requirements

### Requirement: Legal page sources
Boardly SHALL know two legal pages, the imprint (`imprint`) and the privacy policy (`privacy`). The operator configures them when the server starts:

- `BOARDLY_LEGAL_DIR` names a directory with Markdown files for the pages. In it, `<page>.md` is the page's fallback content and `<page>.<language>.md` is its content in a supported language (`en`, `de`), for example `imprint.md`, `imprint.de.md` and `privacy.en.md`.
- `BOARDLY_IMPRINT_URL` and `BOARDLY_PRIVACY_URL` set an external page as the imprint or the privacy policy. One URL serves every language.

Each page SHALL have at most one source: its files or its URL. A page with neither isn't configured. Leading and trailing whitespace is removed from all three variables, and an empty value counts as unset. Without any of them, no legal page exists. The files SHALL be read once when the server starts, so changed files take effect after a restart.

#### Scenario: Files for both pages
- **WHEN** the server starts with `BOARDLY_LEGAL_DIR=/legal` and `/legal` holds `imprint.md`, `imprint.en.md` and `privacy.md`
- **THEN** both the imprint and the privacy policy are served from these files

#### Scenario: External imprint, privacy policy from a file
- **WHEN** the server starts with `BOARDLY_IMPRINT_URL=https://example.at/impressum` and `BOARDLY_LEGAL_DIR=/legal` holding only `privacy.md`
- **THEN** the imprint links to `https://example.at/impressum` and the privacy policy is served from `privacy.md`

#### Scenario: Nothing configured
- **WHEN** the server starts without `BOARDLY_LEGAL_DIR`, `BOARDLY_IMPRINT_URL` and `BOARDLY_PRIVACY_URL`
- **THEN** neither legal page exists

#### Scenario: Only one page
- **WHEN** the server starts with `BOARDLY_LEGAL_DIR=/legal` and `/legal` holds only `imprint.de.md`
- **THEN** the imprint exists and the privacy policy doesn't

### Requirement: Legal configuration is checked at startup
The server SHALL refuse to start, with an error message that names the variable or file, when:

- `BOARDLY_LEGAL_DIR` doesn't exist, isn't a directory or can't be read,
- a Markdown file (name ending in `.md`) in the directory isn't named `imprint.md`, `privacy.md` or `<page>.<language>.md` with a known page and a supported language,
- a page's Markdown file can't be read, is empty after removing whitespace, or is larger than 1 MiB,
- `BOARDLY_IMPRINT_URL` or `BOARDLY_PRIVACY_URL` isn't an absolute `http` or `https` URL,
- a page has both a URL and at least one Markdown file.

Files that don't end in `.md`, files whose name starts with a dot, and subdirectories SHALL be ignored.

#### Scenario: Unknown Markdown file
- **WHEN** the server starts with `BOARDLY_LEGAL_DIR=/legal` and `/legal` holds `impressum.md`
- **THEN** it exits with a non-zero status and an error message that names `impressum.md`

#### Scenario: Unsupported language
- **WHEN** `/legal` holds `imprint.fr.md`
- **THEN** the server exits with a non-zero status and an error message that names `imprint.fr.md`

#### Scenario: Page with a file and a URL
- **WHEN** the server starts with `BOARDLY_IMPRINT_URL=https://example.at/impressum` and `/legal` holds `imprint.de.md`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_IMPRINT_URL` and the imprint's files

#### Scenario: Invalid URL
- **WHEN** the server starts with `BOARDLY_PRIVACY_URL=example.at/privacy`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_PRIVACY_URL`

#### Scenario: Missing directory
- **WHEN** the server starts with `BOARDLY_LEGAL_DIR=/legal` and `/legal` doesn't exist
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_LEGAL_DIR`

#### Scenario: Empty file
- **WHEN** `/legal/privacy.md` contains only blank lines
- **THEN** the server exits with a non-zero status and an error message that names `privacy.md`

#### Scenario: Other files are ignored
- **WHEN** `/legal` holds `imprint.md`, `README.txt`, `.DS_Store` and a subdirectory `old`
- **THEN** the server starts and serves the imprint from `imprint.md`

### Requirement: Language of a legal page
For a page served from files, Boardly SHALL pick the content for a requested language in this order: `<page>.<language>.md`, then `<page>.md`, then the other languages' variants with English first. An unsupported or missing requested language skips the first step. The content SHALL say which file's language it is in, or that it is the fallback.

#### Scenario: Requested language exists
- **WHEN** the imprint has `imprint.md` and `imprint.de.md` and German content is requested
- **THEN** the content of `imprint.de.md` is served

#### Scenario: Fallback file
- **WHEN** the imprint has `imprint.md` and `imprint.de.md` and English content is requested
- **THEN** the content of `imprint.md` is served

#### Scenario: Only another language
- **WHEN** the imprint has only `imprint.de.md` and English content is requested
- **THEN** the content of `imprint.de.md` is served

#### Scenario: English before other variants
- **WHEN** the privacy policy has `privacy.en.md` and `privacy.de.md` but no `privacy.md` and French content is requested
- **THEN** the content of `privacy.en.md` is served

### Requirement: Legal page content API
`GET /api/legal/{page}` SHALL return the content of a page served from files, without needing a session. The optional query parameter `lang` names the requested language. The response contains the page's content as a note document, made from the Markdown in the same way as Markdown imported into notes (headings, paragraphs, lists, task lists, quotes, code, tables, links and emphasis), and the language of the file it came from (`en`, `de`, or none for `<page>.md`). Raw HTML in the Markdown SHALL be shown as text, never as markup. The route SHALL answer `404` with the error code `not_found` for an unknown page, a page that isn't configured, and a page that is an external URL.

#### Scenario: German imprint
- **WHEN** a signed-out visitor's browser requests `GET /api/legal/imprint?lang=de` and `imprint.de.md` exists
- **THEN** the response is `200` with the document made from `imprint.de.md` and the language `de`

#### Scenario: Raw HTML stays text
- **WHEN** `privacy.md` contains `<script>alert(1)</script>` and the privacy policy is requested
- **THEN** the document contains that text as text and no script is run when the page is shown

#### Scenario: External page
- **WHEN** the imprint is set by `BOARDLY_IMPRINT_URL` and `GET /api/legal/imprint` is requested
- **THEN** the response is `404` with the error code `not_found`

#### Scenario: Unknown page
- **WHEN** `GET /api/legal/terms` is requested
- **THEN** the response is `404` with the error code `not_found`

### Requirement: Legal link targets in the HTML document
The web app's HTML document SHALL carry, on every response that serves it, including the fallback for client-side routes, one `<meta name="boardly-imprint">` and one `<meta name="boardly-privacy">` tag for each configured page. The tag's content is the link target, escaped for HTML: `/imprint` or `/privacy` for a page served from files, and the URL for an external page. A page that isn't configured SHALL have no tag.

#### Scenario: Page from files
- **WHEN** the imprint is served from files and a client requests `GET /settings`
- **THEN** the HTML document contains `<meta name="boardly-imprint" content="/imprint" />`

#### Scenario: External page
- **WHEN** `BOARDLY_PRIVACY_URL=https://example.at/datenschutz?a=1&b=2` is set and a client requests `GET /`
- **THEN** the HTML document contains a `boardly-privacy` meta tag whose content reads `https://example.at/datenschutz?a=1&b=2`, with the `&` escaped

#### Scenario: Not configured
- **WHEN** no legal page is configured and a client requests `GET /`
- **THEN** the HTML document has no `boardly-imprint` and no `boardly-privacy` meta tag

### Requirement: Legal pages in the web app
For each page served from files, the web app SHALL show the page at its path, `/imprint` or `/privacy`, to every visitor, signed in or not. The page shows the content in the visitor's current language (see "Language of a legal page") and SHALL update without a page reload when the visitor switches the language. Links in the content are shown as links, and task list items as checkboxes that can't be changed. While the content loads, the page shows a loading state. When it can't be loaded, it shows a translated error message. A path whose page isn't configured, or is an external URL, SHALL show the not-found page.

#### Scenario: Signed-out visitor
- **WHEN** a signed-out visitor opens `/imprint` and the imprint is served from files
- **THEN** the page shows the imprint's content inside the app shell

#### Scenario: Language switch
- **WHEN** a visitor reads `/privacy` in English and switches to German, and `privacy.de.md` exists
- **THEN** the page shows the content of `privacy.de.md` without a page reload

#### Scenario: Not configured
- **WHEN** no privacy policy is configured and a visitor opens `/privacy`
- **THEN** the page shows the not-found message

#### Scenario: External imprint
- **WHEN** the imprint is set by `BOARDLY_IMPRINT_URL` and a visitor opens `/imprint`
- **THEN** the page shows the not-found message

#### Scenario: Document title
- **WHEN** a visitor opens `/imprint` in German and the app title is "Boardly"
- **THEN** the document title is "Impressum - Boardly", whatever headings the content has
