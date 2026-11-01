# Spec Delta

## ADDED Requirements

### Requirement: Domain of a link
The domain of a link SHALL be the host of its URL in lower case, with one leading `www.` removed. Port, path, query and fragment are ignored. Subdomains are different domains: `gist.github.com` is not `github.com`. Only absolute `http` and `https` URLs have a domain.

#### Scenario: www is ignored
- **WHEN** a link has the URL `https://WWW.GitHub.com/foo?x=1`
- **THEN** its domain is `github.com`

#### Scenario: Subdomain is its own domain
- **WHEN** links have the URLs `https://github.com/a` and `https://gist.github.com/b`
- **THEN** their domains are `github.com` and `gist.github.com`

### Requirement: Filter favourites by domain
`/favourites` SHALL accept a domain filter in its URL (`/favourites?domain=github.com`). The given domain is normalised like a link's domain, so `WWW.GitHub.com` filters by `github.com`. While it is set, the tree SHALL show only links whose domain equals it, together with the folders that contain them, expanded. The domain filter SHALL be shown as a removable chip "Domain: github.com" / "Domain: github.com" next to the tag chips. It applies together with the search text and the chosen tags.

#### Scenario: Open with a domain
- **WHEN** Ann has links on `github.com`, `gist.github.com` and `notgithub.com` and opens `/favourites?domain=github.com`
- **THEN** only the `github.com` links and their folders are shown, and the chip "Domain: github.com" is shown

#### Scenario: Domain and search
- **WHEN** the domain filter is `github.com` and Ann types "boardly"
- **THEN** only links on `github.com` that match "boardly" are shown

#### Scenario: Remove the chip
- **WHEN** Ann removes the "Domain: github.com" chip
- **THEN** the domain filter is gone from the URL and the tree shows what the search text and tags match

### Requirement: Domain hint when entering a link
Wherever the user enters a favourite link's URL, namely the "New link" and "Edit link" dialog (on `/favourites`, in the favourite details, in the favourite explorer widget and from "New favourite…" in the link widget settings) and the "Save link" page, the URL field SHALL show a hint with the number of the user's favourite links that have the same domain as the entered URL: "1 favourite on github.com" / "1 Favorit auf github.com", "3 favourites on github.com" / "3 Favoriten auf github.com". When an existing link is edited, the link itself SHALL NOT be counted and the hint says "2 other favourites on github.com" / "2 weitere Favoriten auf github.com". The hint SHALL NOT be shown while the URL isn't an absolute `http` or `https` URL or when the count is 0. The hint follows the entered URL while the user types. The hint SHALL be a link to `/favourites?domain=<domain>` that opens in a new tab, so the dialog or page and what was entered stay as they are.

#### Scenario: New link on a known domain
- **WHEN** Ann has three links on `github.com` and enters `https://github.com/new/repo` in the "New link" dialog
- **THEN** the dialog shows "3 favourites on github.com"

#### Scenario: Unknown domain
- **WHEN** Ann enters a URL whose domain none of her links has
- **THEN** no hint is shown

#### Scenario: Invalid URL
- **WHEN** Ann enters `github.com` without a scheme
- **THEN** no hint is shown

#### Scenario: Editing excludes the link itself
- **WHEN** Ann has three links on `github.com` and edits one of them
- **THEN** the dialog shows "2 other favourites on github.com"

#### Scenario: Only link on its domain
- **WHEN** Ann edits her only link on `example.org`
- **THEN** no hint is shown

#### Scenario: Save link page
- **WHEN** a link to `https://github.com/x` is shared to Boardly and Ann has two links on `github.com`
- **THEN** the "Save link" page shows "2 favourites on github.com"

#### Scenario: Follow the hint
- **WHEN** Ann activates the hint "3 favourites on github.com" in the "New link" dialog
- **THEN** `/favourites?domain=github.com` opens in a new tab, and the dialog stays open with what she entered

## MODIFIED Requirements

### Requirement: Filter state and clearing
The search text, the chosen tags and the domain filter SHALL be kept in the page URL, so a reload or going back restores them. While a filter is active, a "Clear filters" / "Filter zurücksetzen" action SHALL clear all of them. When nothing matches, the page SHALL show "No favourites match" / "Keine passenden Favoriten" with the same action instead of the tree.

#### Scenario: Reload
- **WHEN** the user has typed "graf" and chosen "tools", and reloads the page
- **THEN** the search field shows "graf", "tools" is chosen, and the tree is filtered

#### Scenario: Reload with a domain
- **WHEN** the domain filter is `github.com` and the user reloads the page
- **THEN** the chip "Domain: github.com" is shown and the tree is filtered by it

#### Scenario: Nothing matches
- **WHEN** the user types a text no link matches
- **THEN** the page shows "No favourites match" and "Clear filters"

#### Scenario: Clear
- **WHEN** the user activates "Clear filters"
- **THEN** the search field is empty, no tag is chosen, no domain filter is set, and the whole tree is shown with the remembered collapse state

### Requirement: No moving while filtered
While a search, tag or domain filter is active, nodes SHALL NOT be movable on `/favourites`, by pointer or keyboard, and the page SHALL show "Clear the filters to reorder." / "Setze die Filter zurück, um die Reihenfolge zu ändern." Editing, deleting and adding nodes stay possible.

#### Scenario: Drag handles off
- **WHEN** a filter is active
- **THEN** no row offers a move action, and the hint is shown

#### Scenario: Edit while filtered
- **WHEN** a filter is active and the user edits a shown link
- **THEN** the change is saved and the filter stays active
