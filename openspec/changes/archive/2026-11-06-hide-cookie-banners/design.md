# Design

## Context

`Browser.Screenshot` (`backend/internal/offline/browser.go`) intercepts every request of the page with the CDP Fetch domain and answers it through `Browser.Fetch`, which already fails requests with `BlockedByClient`. After `Navigate`, it waits `settleTime` (1 s) and captures a 1280×800 JPEG. Every screenshot starts in a fresh browser context with no cookies. See proposal.md for why banners are a problem.

## Goals / Non-Goals

**Goals:** most banners from the common consent tools are missing from screenshots; no new attack surface; no configuration.

**Non-Goals:**
- clicking "reject" or giving consent (autoconsent);
- unlocking cookie walls;
- hiding banners in the article;
- generic heuristics such as "hide every fixed element with a high z-index";
- a user-editable list.

## Decisions

### Host blocking in `answer`, before `Fetch` makes the request
- `answer` checks the paused request's host against an embedded list of consent-provider hosts (`consent_hosts.txt`, one host per line, `#` comments) before handing it to `Fetch`. A host matches when it equals a listed host or is a subdomain of it. A match returns `BlockedByClient` without counting as a made request.
- The list names the providers' script and CDN hosts, not their marketing sites. Examples: `cdn.cookielaw.org`, `consent.cookiebot.com`, `consentcdn.cookiebot.com`, `app.usercentrics.eu`, `sdk.privacy-center.org`, `cmp.quantcast.com`, `mgr.consensu.org`, `cdn.privacy-mgmt.com`, `consent.trustarc.com`, `cdn.consentmanager.net`, `cdn.iubenda.com`, `app.termly.io`, `cmp.osano.com`, `cdn-cookieyes.com`, `fundingchoicesmessages.google.com`.
- **The page itself is exempt.** When the paused request is a `Document` of the main frame, it isn't checked against the list. In Chrome, the main frame's id equals the page target's id, so this covers the link's URL and its redirects. Iframes from providers, such as Sourcepoint's message iframe, are still blocked.
- The matcher is a field on `Browser` (`consentHost func(host string) bool`), set by `NewBrowser`, so tests can point it at a test server's host.
- **Alternative considered:** bundling EasyList's or Fanboy's cookie list. It was rejected: it is large (thousands of cosmetic and network rules), needs a rule parser, needs regular updates, and is licensed under GPLv3 / CC BY-SA. A short curated list of the providers that cover most sites is enough for a best-effort screenshot.

### Hiding banners with a DevTools stylesheet, not a `<style>` in the page
- After `Navigate` and before the settle wait, Boardly enables the DOM and CSS domains and calls `CSS.createStyleSheet` for the main frame. It then sets the sheet's text with `CSS.setStyleSheetText`, using an embedded `consent.css`.
- The rules hide known banner, dialog and overlay elements with `display: none !important`, for example `#onetrust-consent-sdk`, `#CybotCookiebotDialog`, `#usercentrics-root`, `#didomi-host`, `.qc-cmp2-container`, `[id^="sp_message_container"]`, `.fc-consent-root`, `#cmpbox`, `#iubenda-cs-banner`, `.cky-consent-container`, `#cmplz-cookiebanner-container`, `#BorlabsCookieBox` and `#cookie-notice`. One rule sets `html, body { overflow: auto !important; }` to undo the scroll lock.
- A DevTools-created sheet is an inspector stylesheet. The page's Content Security Policy doesn't apply to it, while a `<style>` element added by script would be blocked by a strict `style-src`. The sheet stays in place, so banners added during the settle time are hidden too.
- **Alternatives considered:**
  - injecting `<style>` into the fulfilled HTML: blocked by CSP, and it means rewriting the response bodies;
  - `Page.addScriptToEvaluateOnNewDocument`: it runs before the document exists, and would still need a `<style>` element or adopted stylesheets.

### Best effort
A failure to create the stylesheet doesn't fail the screenshot: it's logged at debug level and the screenshot is taken anyway. Host blocking can't fail on its own.

## Risks / Trade-offs

- **[A site needs a provider's script to show its content]** → The screenshot may stay blank or locked. This is accepted for a best-effort screenshot; the article doesn't depend on it.
- **[Selectors or hosts go stale]** → The lists are small, plain text files and easy to update. A stale entry only means a banner stays visible.
- **[A selector hides real content]** → Only provider-specific ids and classes are listed, no generic names such as `.modal` or `.overlay`.
- **[`overflow: auto` on `html, body` changes a page's layout]** → It is rare for a page to need `overflow: hidden` on its root without a dialog open. The viewport-only screenshot is barely affected.

## Migration Plan

None. The change takes effect for new captures and refreshes; existing screenshots stay as they are.
