# Spec Delta

## MODIFIED Requirements

### Requirement: Global footer
Every page inside the app shell SHALL end with a slim footer below the page content, in the same container as the header. It is a `contentinfo` landmark, in small muted text. On pages shorter than the viewport, it sits at the bottom of the viewport. On longer pages, it comes after the content.

- On the left, "Imprint" / "Impressum" and "Privacy" / "Datenschutz" link to the configured legal pages (see the `legal-pages` capability), separated by a dot. A page that isn't configured has no link. When analytics consent is on (see the `analytics-consent` capability), a "Cookie settings" / "Cookie-Einstellungen" button follows the legal links, separated by a dot, and opens the consent settings. When analytics consent is off, the footer has no such button. A link to a page served from files opens it in the same tab. A link to an external URL opens it in a new tab without giving the new page access to the opener, shows a visible external-link marker, and has an accessible name that says it opens in a new tab.
- On the right, "Open source" / "Open Source" is followed by a "GitHub" link to `https://github.com/danielraab/boardly`. It shows the GitHub mark, hidden from assistive technology, and a visible external-link marker. It opens in a new tab without giving the new page access to the opener. Its accessible name is "GitHub (opens in new tab)" / "GitHub (öffnet in neuem Tab)", and it points to the repository even when the app title is customised.

At narrow widths, the two groups SHALL wrap onto separate lines instead of scrolling horizontally.

#### Scenario: Footer on every page
- **WHEN** a visitor opens `/`, `/login`, a dashboard, `/settings` or an unknown path
- **THEN** the page ends with the footer, and the footer shows "Open source · GitHub"

#### Scenario: Both legal pages from files
- **WHEN** the imprint and the privacy policy are served from files and a visitor views any page in German
- **THEN** the footer shows "Impressum · Datenschutz", linking to `/imprint` and `/privacy` in the same tab

#### Scenario: External privacy policy
- **WHEN** the privacy policy is `https://example.at/datenschutz`
- **THEN** the footer's "Privacy" link points there, opens in a new tab, shows the external-link marker and is named "Privacy (opens in new tab)"

#### Scenario: Nothing configured
- **WHEN** no legal page is configured
- **THEN** the footer shows only "Open source · GitHub"

#### Scenario: Short page
- **WHEN** the login page is viewed in a viewport taller than its content
- **THEN** the footer is at the bottom of the viewport

#### Scenario: Narrow viewport
- **WHEN** a page with both legal links is shown at 320 px width
- **THEN** the legal links and "Open source · GitHub" are on separate lines and nothing scrolls horizontally

#### Scenario: Cookie settings with consent on
- **WHEN** analytics consent is on, the privacy policy is served from files, and a visitor views any page in German
- **THEN** the footer shows "Datenschutz · Cookie-Einstellungen", and activating "Cookie-Einstellungen" opens the consent settings

#### Scenario: No cookie settings with consent off
- **WHEN** analytics consent is off
- **THEN** the footer has no "Cookie settings" button
