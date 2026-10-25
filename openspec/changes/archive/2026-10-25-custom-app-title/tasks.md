# Tasks

## 1. Configuration

- [x] 1.1 Add `AppTitle` to `config.Config`, read from `BOARDLY_APP_TITLE` (trimmed, default "Boardly" when unset or blank), and verify with new cases in `config_test.go` (default, custom, blank)
- [x] 1.2 Document `BOARDLY_APP_TITLE` in the README configuration table and as a commented-out entry in `.env.example`, and verify both mention the default "Boardly"

## 2. Serving the HTML document and manifests

- [x] 2.1 Add `<meta name="application-name" content="Boardly" />` to `frontend/index.html` next to `<title>Boardly</title>`, and verify `pnpm build` writes both markers unchanged into `dist/index.html`
- [x] 2.2 Make `web.Handler`/`NewHandler` take the app title and serve `index.html` (direct and fallback) with the escaped title in `<title>` and the meta tag, and verify with `web_test.go` cases for the default, a custom title, a deep-link fallback and a title with `&` and `<`
- [x] 2.3 Serve `*.webmanifest` with `name` and `short_name` set to the app title, other fields unchanged, content type `application/manifest+json` and `Cache-Control: no-cache`, and verify with `web_test.go` cases for both languages
- [x] 2.4 Pass `cfg.AppTitle` to `web.Handler` in `main.go`, and verify `go build ./...` and `main_test.go` pass

## 3. Sign-in email

- [x] 3.1 Add `AppName` to `MagicLinkData` and replace "Boardly" with `{{.AppName}}` in the en/de txt and html templates, and verify the golden tests pass unchanged with "Boardly" and a new test renders "Boardly Staging" in subject, heading, text, button and signature
- [x] 3.2 Give `auth.Service` the app title (set from config in `main.go`) and pass it when rendering the magic link, and verify an auth test checks that a sent email's subject contains the configured title

## 4. Web app

- [x] 4.1 Add a frontend module that exports the app title read from the `application-name` meta tag (trimmed, fallback "Boardly"), and verify with a Vitest test for present, missing and blank tags
- [x] 4.2 Use the app title in `usePageTitle`, `AppShell` and `Landing`, and verify the existing tests still pass and new tests show "Boardly Staging" in the header link (with matching accessible name), the landing hero and "Settings - Boardly Staging" as document title
- [x] 4.3 Replace "Boardly" with `{{appName}}` in `landing.featuresTitle`, `verify.text` and the reserved-slug message in `en.json` and `de.json`, provide `appName` (preferably as an i18next default variable), and verify a test renders the German reserved-slug message with a custom title
- [x] 4.4 Make sure a long app title truncates in the header instead of breaking the layout, and verify visually at phone width with a 40-character title

## 5. Verification

- [x] 5.1 Run `make check` and `make test` and verify both pass
- [x] 5.2 Build the Docker image, run it with `BOARDLY_APP_TITLE="Boardly Staging"`, and verify the tab title, the header, `/manifest.en.webmanifest` and a sign-in email in the log show "Boardly Staging"
