## ADDED Requirements

### Requirement: Cookie banners are left out of screenshots
When the browser takes a screenshot for an offline copy, it SHALL NOT load anything from the hosts of well-known consent-management providers. These requests fail in the browser as blocked requests and are never made by the server. The page of the link itself, including its redirects, is never blocked this way. Before the screenshot, the banners, dialogs and overlays of well-known consent tools SHALL be hidden, and a page they kept from scrolling SHALL be scrollable again, so the screenshot shows the page behind them. Nothing on the page is clicked and no consent is given. The article of the copy is not affected.

#### Scenario: Banner script from a consent provider
- **WHEN** a page loads its cookie banner script from `https://cdn.cookielaw.org/`
- **THEN** no request to `cdn.cookielaw.org` is made, and the screenshot shows the page without the banner

#### Scenario: Banner on the page itself
- **WHEN** a page contains a full-screen cookie dialog with a known banner id, and sets `overflow: hidden` on its `body`
- **THEN** the screenshot shows the page content, not the dialog

#### Scenario: The link is on a provider's host
- **WHEN** a link points to a page on a host on the consent-provider list
- **THEN** the page itself is loaded and gets a screenshot

#### Scenario: Unknown banner
- **WHEN** a page shows a cookie banner that is on neither list
- **THEN** the screenshot shows the banner, and the copy is still `ready`
