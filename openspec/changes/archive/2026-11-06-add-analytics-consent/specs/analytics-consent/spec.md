# Spec Delta

## Purpose

Lets the operator ask visitors for consent before the analytics snippet runs, with a banner that names the analytics service, follows Boardly's look and language, and can be changed or withdrawn at any time. Without consent, tells visitors that only necessary cookies are used.

## ADDED Requirements

### Requirement: Consent configuration
The server SHALL read two optional settings when it starts:

- `BOARDLY_CONSENT_ANALYTICS` is the name of the analytics service, for example `Google Analytics`. A non-empty value turns analytics consent on. Unset or empty leaves it off.
- `BOARDLY_CONSENT_ANALYTICS_COOKIES` is a comma-separated list of the cookie names the service sets. Each name consists of letters, digits, `_`, `-` and `.`, and may contain `*`, which matches any run of characters (for example `_ga,_ga_*`). Leading and trailing whitespace around the value and around each name is removed, and empty names are ignored.

Leading and trailing whitespace is removed from `BOARDLY_CONSENT_ANALYTICS`. Changing either setting SHALL need only a restart of the server, not a new build of the web app.

#### Scenario: Consent off by default
- **WHEN** the server starts without `BOARDLY_CONSENT_ANALYTICS`
- **THEN** analytics consent is off and the analytics snippet, if any, is served as before

#### Scenario: Consent on
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS="Google Analytics"` and `BOARDLY_CONSENT_ANALYTICS_COOKIES=" _ga, _ga_* ,"`
- **THEN** analytics consent is on for the service "Google Analytics" with the cookies `_ga` and `_ga_*`

#### Scenario: Blank service name
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS="   "`
- **THEN** analytics consent is off

### Requirement: Consent configuration is checked at startup
The server SHALL refuse to start, with an error message that names the variable at fault, when:

- `BOARDLY_CONSENT_ANALYTICS_COOKIES` is set but `BOARDLY_CONSENT_ANALYTICS` isn't,
- a cookie name in `BOARDLY_CONSENT_ANALYTICS_COOKIES` contains a character other than letters, digits, `_`, `-`, `.` and `*`, or consists only of `*`,
- analytics consent is on and `BOARDLY_ANALYTICS_SNIPPET` is empty,
- analytics consent is on and the snippet contains a `<script>` element without both `type="text/plain"` and `data-category="analytics"`,
- analytics consent is on and the snippet contains an element other than `<script>`, for example `<img>`, `<iframe>`, `<link>` or `<noscript>`, because the browser would load it without consent. Comments and whitespace between the elements are allowed.
- analytics consent is on and no privacy policy is configured (see the `legal-pages` capability).

#### Scenario: Script not marked
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS="Google Analytics"` and `BOARDLY_ANALYTICS_SNIPPET='<script async src="https://www.googletagmanager.com/gtag/js?id=G-1"></script>'`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_ANALYTICS_SNIPPET` and says the script needs `type="text/plain"` and `data-category="analytics"`

#### Scenario: Tracking pixel
- **WHEN** consent is on and the snippet contains `<noscript><img src="https://stats.example.org/matomo.php?idsite=1"></noscript>`
- **THEN** the server exits with a non-zero status and an error message that names `BOARDLY_ANALYTICS_SNIPPET`

#### Scenario: No privacy policy
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS="Matomo"`, a marked snippet, and neither a `privacy*.md` file nor `BOARDLY_PRIVACY_URL`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_CONSENT_ANALYTICS` and says a privacy policy is required

#### Scenario: No snippet
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS="Matomo"` and without `BOARDLY_ANALYTICS_SNIPPET`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_ANALYTICS_SNIPPET`

#### Scenario: Cookies without a service
- **WHEN** the server starts with `BOARDLY_CONSENT_ANALYTICS_COOKIES=_ga` and without `BOARDLY_CONSENT_ANALYTICS`
- **THEN** it exits with a non-zero status and an error message that names both variables

#### Scenario: Invalid cookie name
- **WHEN** the server starts with consent on and `BOARDLY_CONSENT_ANALYTICS_COOKIES="_ga;path=/"`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_CONSENT_ANALYTICS_COOKIES` and `_ga;path=/`

#### Scenario: Valid configuration
- **WHEN** consent is on, a privacy policy is configured, and the snippet contains only `<script type="text/plain" data-category="analytics">` elements and comments
- **THEN** the server starts

### Requirement: Consent settings in the HTML document
When analytics consent is on, the web app's HTML document SHALL carry, on every response that serves it, including the fallback for client-side routes:

- `<meta name="boardly-consent-analytics">` with the service name,
- `<meta name="boardly-consent-cookies">` with the cookie names, comma-separated, when any are set,
- `<meta name="boardly-consent-revision">` with the consent revision, a positive whole number.

When analytics consent is off, the document SHALL instead carry `<meta name="boardly-cookie-notice">` with the content `necessary` when no analytics snippet is set, and `cookieless-analytics` when one is set (see "Cookie notice without consent").

All contents SHALL be escaped for HTML. The revision SHALL be derived from the analytics snippet, the service name and the cookie names: the same settings give the same revision after every restart, and changing any of them gives a different revision. When analytics consent is off, the document SHALL have none of the `boardly-consent-*` tags, and when it is on, it SHALL have no `boardly-cookie-notice` tag.

#### Scenario: Tags present
- **WHEN** consent is on for "Google Analytics" with the cookies `_ga,_ga_*` and a client requests `GET /settings`
- **THEN** the HTML document contains `<meta name="boardly-consent-analytics" content="Google Analytics" />`, `<meta name="boardly-consent-cookies" content="_ga,_ga_*" />` and a `boardly-consent-revision` tag with a positive whole number

#### Scenario: Stable revision
- **WHEN** the server restarts with unchanged settings
- **THEN** the `boardly-consent-revision` tag has the same value as before

#### Scenario: Changed snippet
- **WHEN** the operator changes the snippet's measurement ID and restarts the server
- **THEN** the `boardly-consent-revision` tag has a different value

#### Scenario: Consent off, no snippet
- **WHEN** consent is off, no analytics snippet is set, and a client requests `GET /`
- **THEN** the HTML document has no `boardly-consent-*` meta tag and contains `<meta name="boardly-cookie-notice" content="necessary" />`

#### Scenario: Consent off, with snippet
- **WHEN** consent is off, an analytics snippet is set, and a client requests `GET /settings`
- **THEN** the HTML document contains `<meta name="boardly-cookie-notice" content="cookieless-analytics" />`

#### Scenario: No notice tag with consent on
- **WHEN** consent is on and a client requests `GET /`
- **THEN** the HTML document has no `boardly-cookie-notice` meta tag

### Requirement: Analytics runs only after consent
When analytics consent is on, the analytics snippet's scripts SHALL NOT run, and no request to the analytics service SHALL be made, until the visitor has accepted the "Analytics" category in the current revision. After the visitor accepts, the scripts SHALL run in the order they appear in the snippet, without a page reload. On later page loads with a stored acceptance for the current revision, they SHALL run without asking. When the visitor has rejected analytics, they SHALL NOT run. When analytics consent is off, the web app SHALL NOT load the consent manager, and the snippet behaves as described in the `app-hosting` capability.

#### Scenario: First visit
- **WHEN** a visitor opens any page for the first time with consent on
- **THEN** the banner is shown and the analytics scripts haven't run

#### Scenario: Accept
- **WHEN** the visitor chooses "Accept" in the banner
- **THEN** the banner closes and the analytics scripts run without a page reload

#### Scenario: Reject
- **WHEN** the visitor chooses "Reject" and then opens other pages or reloads
- **THEN** the analytics scripts never run and the banner isn't shown again

#### Scenario: Accepted earlier
- **WHEN** a visitor who accepted analytics in the current revision reloads the page
- **THEN** the analytics scripts run and the banner isn't shown

#### Scenario: Consent off
- **WHEN** consent is off and a visitor opens any page
- **THEN** no consent banner is shown and the consent manager's code isn't downloaded

### Requirement: Consent banner
When analytics consent is on and the visitor hasn't made a choice in the current revision, every page, signed in or not, SHALL show a consent banner. The banner SHALL NOT block the page: the visitor can keep reading and use links, including the link to the privacy policy. It contains:

- a title, "Cookies and analytics" / "Cookies und Statistik",
- a text that says Boardly uses the named service to see how the app is used only with the visitor's consent, and that the choice can be changed at any time in the footer's cookie settings,
- a link to the privacy policy, which opens a page served from files in the same tab and an external URL in a new tab,
- the buttons "Accept" / "Akzeptieren" and "Reject" / "Ablehnen", which SHALL look alike and have equal weight, and "Settings" / "Einstellungen", which opens the consent settings.

The banner and the consent settings SHALL be shown in the app's current language and SHALL switch language without a page reload when the visitor switches it. They SHALL use the current theme's colours, light or dark, and follow theme changes without a page reload. They SHALL be usable with the keyboard alone.

#### Scenario: German banner
- **WHEN** a visitor opens the app in German with consent on for "Google Analytics"
- **THEN** the banner's title is "Cookies und Statistik", its text names "Google Analytics", and it shows "Akzeptieren", "Ablehnen" and "Einstellungen"

#### Scenario: Equal buttons
- **WHEN** the banner is shown
- **THEN** "Accept" and "Reject" have the same size, colour and position weight

#### Scenario: Reading the privacy policy
- **WHEN** the banner is shown and the visitor follows its privacy policy link to `/privacy`
- **THEN** the privacy page opens with the banner still shown and usable

#### Scenario: Language switch
- **WHEN** the banner is shown in English and the visitor switches the app to German
- **THEN** the banner shows its German texts without a page reload

#### Scenario: Dark theme
- **WHEN** the visitor uses a dark theme
- **THEN** the banner uses the theme's dark colours

### Requirement: Consent settings dialog
The consent settings SHALL be a modal dialog that lists two categories:

- "Necessary" / "Notwendig", always on and not changeable, which says that Boardly uses cookies only to keep the visitor signed in and to remember this choice,
- "Analytics" / "Statistik", off unless the visitor accepted it, which names the service and lists the configured cookie names.

It offers "Accept all" / "Alle akzeptieren", "Reject all" / "Alle ablehnen" and "Save settings" / "Einstellungen speichern", where "Accept all" and "Reject all" SHALL have equal weight. The dialog links to the privacy policy like the banner.

#### Scenario: Cookies listed
- **WHEN** consent is on with the cookies `_ga,_ga_*` and the visitor opens the settings
- **THEN** the "Analytics" category names the service and lists `_ga` and `_ga_*`

#### Scenario: Save a choice
- **WHEN** the visitor turns on "Analytics" in the settings and chooses "Save settings"
- **THEN** the dialog closes and the analytics scripts run

### Requirement: Stored consent choice
The visitor's choice SHALL be stored only in their browser, in a first-party cookie named `boardly_consent` with the path `/`, `SameSite=Lax`, the `Secure` attribute when the app is served over HTTPS, and a lifetime of 182 days. It records the accepted categories, the revision and when the choice was made. The server SHALL NOT read, store or log consent choices. The banner SHALL be shown again when the stored choice is missing, has expired, or belongs to another revision. When the browser doesn't store the cookie, the banner SHALL be shown again on each page load, and the analytics scripts SHALL NOT run before the visitor accepts on that page load.

#### Scenario: Revision changed
- **WHEN** a visitor accepted analytics, and the operator changes the service name and restarts the server
- **THEN** the visitor sees the banner again on the next page load and the analytics scripts don't run before they choose

#### Scenario: Choice expired
- **WHEN** a visitor's choice is older than 182 days
- **THEN** the banner is shown again

#### Scenario: No server record
- **WHEN** a visitor accepts analytics
- **THEN** no request to Boardly's server records the choice

### Requirement: Withdrawing consent
When analytics consent is on, the visitor SHALL be able to change their choice at any time through the "Cookie settings" button in the global footer (see the `app-shell` capability), which opens the consent settings. When the visitor withdraws a previously given acceptance of analytics, the web app SHALL delete the configured analytics cookies that match a configured name, on the page's host and on its parent domains, and SHALL reload the page so that analytics scripts that already ran stop.

#### Scenario: Withdraw
- **WHEN** a visitor who accepted analytics opens "Cookie settings", chooses "Reject all", and the cookies `_ga` and `_ga_ABC123` exist with `_ga,_ga_*` configured
- **THEN** both cookies are deleted, the page reloads, and the analytics scripts don't run after the reload

#### Scenario: Change from rejected to accepted
- **WHEN** a visitor who rejected analytics opens "Cookie settings" and chooses "Accept all"
- **THEN** the analytics scripts run without a page reload

### Requirement: Cookie notice without consent
When analytics consent is off, every page, signed in or not, SHALL show a small cookie notice fixed at the bottom of the viewport until the visitor closes it. It is a labelled region ("Cookie notice" / "Cookie-Hinweis"), SHALL NOT block the page, SHALL NOT cover the media mini player bar, and SHALL NOT make the page scroll horizontally at 320 px width. Its text depends on the `boardly-cookie-notice` variant:

- `necessary`: "Boardly only uses necessary cookies and browser storage, to keep you signed in and remember your settings." / "Boardly verwendet nur notwendige Cookies und Browser-Speicher, um dich angemeldet zu halten und deine Einstellungen zu merken."
- `cookieless-analytics`: the same text, followed by "Usage statistics are collected without cookies." / "Nutzungsstatistiken werden ohne Cookies erhoben."

When a privacy policy is configured, the notice SHALL end with a "Privacy policy" / "Datenschutzerklärung" link to it, which opens a page served from files in the same tab and an external URL in a new tab. A close button named "Close cookie notice" / "Cookie-Hinweis schließen" hides the notice. The web app SHALL remember in the browser's storage that the notice was closed, together with its variant, and SHALL NOT show it again unless the variant changes. When the browser can't store this, the notice SHALL stay hidden until the page is reloaded. The notice SHALL be shown in the app's current language, switch language without a page reload, and use the current theme's colours. When analytics consent is on, the notice SHALL NOT be shown.

#### Scenario: No analytics
- **WHEN** a visitor opens the app in German, consent is off, no snippet is set, and a privacy policy is served from files
- **THEN** a notice at the bottom says "Boardly verwendet nur notwendige Cookies und Browser-Speicher, um dich angemeldet zu halten und deine Einstellungen zu merken." and links "Datenschutzerklärung" to `/privacy`

#### Scenario: Cookieless analytics
- **WHEN** consent is off and an analytics snippet is set
- **THEN** the notice also says "Usage statistics are collected without cookies."

#### Scenario: No privacy policy
- **WHEN** consent is off and no privacy policy is configured
- **THEN** the notice has no privacy link

#### Scenario: Close
- **WHEN** the visitor closes the notice and reloads the page
- **THEN** the notice isn't shown

#### Scenario: Variant changed
- **WHEN** the visitor closed the `necessary` notice, and the operator adds an analytics snippet and restarts the server
- **THEN** the notice with the cookieless-analytics text is shown again

#### Scenario: Storage blocked
- **WHEN** the browser blocks storage and the visitor closes the notice
- **THEN** the notice stays hidden while they navigate in the app and is shown again after a reload

#### Scenario: Consent on
- **WHEN** analytics consent is on
- **THEN** no cookie notice is shown, only the consent banner when a choice is needed

#### Scenario: Mini player
- **WHEN** the notice is shown and the media mini player is open
- **THEN** the notice sits above the mini player bar and doesn't cover it
