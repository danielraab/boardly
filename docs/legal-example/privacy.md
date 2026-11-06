# Privacy policy

> Template: replace everything in [brackets], remove what doesn't apply to your instance and add
> what does (hosting, reverse proxy, backups, analytics). It describes what Boardly itself does;
> it is not legal advice. Add `privacy.de.md` next to this file for a German version.

## Who is responsible

[Name and address of the operator, as in the imprint]
Email: [privacy@example.org](mailto:privacy@example.org)

## What this service stores

**Your account.** Your email address, the name and address (slug) you choose, and the time your
account was created. They are needed to sign you in and to show your public dashboards.

**Signing in.** Boardly has no passwords. A sign-in link or code is sent to your email address
and is valid for a short time. If you add passkeys, Boardly stores their public keys and the name
of their provider, never a secret. The emails are sent through [mail provider].

**Your content.** The dashboards, favourites, tags, notes, attachments, images, radio stations and
app tokens you create are stored on the server so that you can use them. Public dashboards are
visible to anyone who has their address.

**Cookies.** Boardly sets only cookies that are needed: `boardly_session` keeps you signed in and
expires 30 days after you last used it, and `boardly_login_request` ties a sign-in link to the
browser that asked for it. [If you use analytics with consent: Cookies for analytics are only set
after you agree, see "Analytics" below. Otherwise:] There are no tracking cookies. Your browser also stores settings such as
the language, the theme, collapsed sections and whether maps may load, in its local storage. They
never leave your device.

**Server logs.** The server logs requests with method, path, status and duration, without your IP
address. To limit abuse, it keeps IP addresses in memory for a short time (rate limiting); they are
not stored. [Add the logs of your reverse proxy or hosting provider, and how long they are kept.]

## Connections to other services

- **Favicons and link previews.** When you save a link, the server fetches the linked page's icon,
  title and preview image. The linked site sees the server's address, not yours.
- **Radio search.** The server sends station searches to radio-browser.info and loads the logos;
  your browser doesn't contact them.
- **Radio streams.** Audio goes directly from the station to your browser, so the station sees your
  IP address while you listen.
- **Maps.** Maps of note locations are loaded from [OpenFreeMap / your map provider] only after you
  choose "Load map". The map provider then sees your IP address and the area you view.
- [**Analytics.** If you set `BOARDLY_ANALYTICS_SNIPPET`, keep one of these and remove the other.
  Otherwise remove this point.]
  - [With consent (`BOARDLY_CONSENT_ANALYTICS`):] Only if you agree in the cookie banner, we use
    [Google Analytics, provided by …] to see how this site is used. It collects [which pages you
    view, your device and browser, an approximate location …] and sets the cookies [`_ga`,
    `_ga_*`], which are kept for [… months]. [Data is transferred to … ; basis for the transfer: …]
    The legal basis is your consent (Art. 6(1)(a) GDPR). You can withdraw it at any time under
    "Cookie settings" at the bottom of every page; the cookies are then deleted. Your choice is
    stored for 182 days in the necessary cookie `boardly_consent`.
  - [Without cookies:] We use [Umami / Plausible, hosted by …] to count page views without
    cookies or similar identifiers. It collects [the page, the referring site, the browser type,
    the country …] and doesn't identify you. The legal basis is [our legitimate interest in knowing
    how this site is used (Art. 6(1)(f) GDPR)].

## How long data is kept

Your data is kept as long as you have an account. You can delete your account under
**Settings → Profile**; after a waiting period of [7] days, it is deleted with all its content.
[Backups are kept for … days.]

## Your rights

You can download all your data under **Settings → Import / Export**. You have the right to
access, rectification, erasure, restriction of processing, data portability and objection, and you
can lodge a complaint with a supervisory authority, [for example the Austrian Data Protection
Authority (dsb.gv.at)]. Write to [privacy@example.org](mailto:privacy@example.org).

Legal basis: [Art. 6 (1) (b) GDPR (providing the service you signed up for) and Art. 6 (1) (f)
GDPR (security of the service)].
