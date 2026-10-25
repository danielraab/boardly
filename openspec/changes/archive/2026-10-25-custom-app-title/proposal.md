# Proposal

## Why

The staging system looks exactly like production: same tab title, same header, same installed app name and same sign-in emails. That makes it easy to mix up the two. An operator should be able to give an instance its own name, such as "Boardly Staging", without building a different frontend.

## What Changes

- New optional setting `BOARDLY_APP_TITLE` (default `Boardly`). It is trimmed, and an empty value falls back to the default. The value replaces the product name in full; it is not a suffix.
- The server writes the app title into the web app's HTML document (document title and an `application-name` meta tag) and into the `name` and `short_name` of both web app manifests when it serves them. The manifests are served with `Cache-Control: no-cache`, so a changed title reaches browsers.
- The web app reads the app title from the `application-name` meta tag and uses it in the header, the landing page hero, the document title (`<page title> - <app title>`) and the translated texts that name the product (en + de).
- Sign-in emails (subject, heading, text, button and signature, en + de, plain text and HTML) use the app title.
- Not changed: the passkey relying party name, the default sender `Boardly <noreply@localhost>` (`BOARDLY_SMTP_FROM` already sets it), API error messages of the Linkwarden/Karakeep-compatible APIs and fetcher User-Agent strings. These name the software, not the instance.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-hosting`: new setting `BOARDLY_APP_TITLE`; the HTML document carries the app title.
- `app-shell`: the header and the document title show the configured app title instead of the fixed "Boardly"; translated texts that name the product use it.
- `landing-page`: the hero shows the configured app title.
- `pwa`: the manifests' `name` and `short_name` are the configured app title, and the manifests aren't cached without revalidation.
- `auth-magic-link`: the sign-in email names the configured app title.

## Impact

- Backend: `internal/config` (new field and variable), `internal/web` (fills in `index.html` and the manifests, needs the title), `main.go` wiring, `internal/mail` (`MagicLinkData.AppName`, 4 templates, golden files unchanged with the default), `internal/auth` (passes the title to the mail).
- Frontend: `index.html` (meta tag), a small helper that reads the app title, `usePageTitle`, `AppShell`, `Landing`, `locales/{en,de}.json` (`{{appName}}` interpolation in 3 strings).
- Docs: README configuration table and `.env.example`.
- No API, database or dependency changes. Existing deployments keep "Boardly".
