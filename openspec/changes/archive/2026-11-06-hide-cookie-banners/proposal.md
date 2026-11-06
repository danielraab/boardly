# Proposal

## Why

Many pages cover most of the viewport with a cookie consent banner or dialog until a visitor answers it. Boardly's browser starts every capture with no cookies, so it never has an answer, and the offline-copy screenshot (and the preview made from it) often shows the banner instead of the page.

## What Changes

- While taking a screenshot, the browser no longer loads anything from the hosts of well-known consent-management providers (for example OneTrust, Cookiebot, Usercentrics, Didomi, Quantcast Choice, Sourcepoint, TrustArc, consentmanager.net, iubenda). Boardly fails these requests itself, like requests to blocked addresses, so the banner script never runs. The page itself is never blocked, even if it is on one of these hosts.
- Before the screenshot, Boardly adds a stylesheet to the page that hides the banners, dialogs and dimmed overlays of these providers and of common self-hosted consent tools (for example Complianz, Borlabs, Klaro, Cookie Notice). It also undoes the scroll lock such banners put on the page.
- Nothing is clicked and no consent is given or stored. Pages that only show their content after consent may still show a banner or an empty page.
- The article and the plain page fetch are unchanged. Only the browser screenshot is affected.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `offline-copies`: new requirement "Cookie banners are left out of screenshots".

## Impact

- Backend only:
  - `backend/internal/offline/browser.go`: blocks consent-provider hosts in `Fetch` and adds the stylesheet in `Screenshot`;
  - new embedded lists of provider hosts and banner selectors in `backend/internal/offline`, with their tests.
- No new dependencies, configuration, routes or OpenAPI changes.
- Export and import are unaffected: no user data changes.
- README: the screenshot section mentions that cookie banners are hidden.
