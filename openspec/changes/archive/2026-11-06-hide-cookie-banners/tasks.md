# Tasks

## 1. Consent-provider hosts

- [x] 1.1 Add the embedded `backend/internal/offline/consent_hosts.txt` and a matcher that accepts a listed host or any subdomain of it, ignoring case and a trailing dot; verify with a table test (exact host, subdomain, look-alike such as `evilcdn.cookielaw.org.example`, unrelated host, comments and blank lines ignored)
- [x] 1.2 In `Browser`, add the `consentHost` matcher, set by `NewBrowser`, and make `answer` fail a matching request with `BlockedByClient` before `Fetch` makes it, except for a `Document` request of the main frame (frame id = target id); verify with `TestBlocksConsent` (provider script and iframe blocked, page on a listed host and other hosts not) and, in `TestBrowserHidesCookieBanners`, that the provider server gets no request and a page on a listed host still loads

## 2. Banner stylesheet

- [x] 2.1 Add the embedded `backend/internal/offline/consent.css` with the provider selectors (`display: none !important`) and the `html, body { overflow: auto !important; }` rule; verify with a test that the file parses into non-empty rules and contains no generic selectors (`.modal`, `.overlay`, `div`, `*`)
- [x] 2.2 In `Screenshot`, after `Navigate` and before the settle wait, enable DOM and CSS, create an inspector stylesheet for the main frame and set its text to `consent.css`; log and continue on failure. Verify with `TestBrowserHidesCookieBanners` (runs with `BOARDLY_TEST_BROWSER_URL`): a page with a red background, a full-screen blue `#onetrust-banner-sdk`, `body { overflow: hidden }`, a strict `Content-Security-Policy: style-src 'self'` header and a script from a host the test matcher blocks gives a red corner pixel, and the test server sees no request for the blocked script

## 3. Docs and checks

- [x] 3.1 Update the README section "Screenshots for offline copies" and the CHANGELOG entry to say that cookie banners of common consent tools are hidden; confirm that export/import and the OpenAPI documents need no change
- [x] 3.2 Run `go test ./internal/offline/...`, also once against a real headless-shell with `BOARDLY_TEST_BROWSER_URL`, and `make check`, and verify they pass
- [x] 3.3 Run `openspec validate hide-cookie-banners --strict` and verify it passes
