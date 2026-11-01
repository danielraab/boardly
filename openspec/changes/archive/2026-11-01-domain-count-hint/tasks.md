# Tasks

## 1. Domain helpers

- [x] 1.1 Add `domainOf(url)` and `normaliseDomain(value)` to `frontend/src/features/favourites/tree.ts` (lower case, strip one leading `www.`, `hostname` without port, `null` for URLs `checkUrl` rejects); verify with `tree.test.ts` cases for `www.`, upper case, port, subdomain, non-http and invalid URLs
- [x] 1.2 Add `domainCount(root, domain, excludeId?)` to `tree.ts`; verify with `tree.test.ts` cases for nested folders, other domains, subdomains and an excluded node

## 2. Domain filter on `/favourites`

- [x] 2.1 Extend `TreeFilter` with `domain?`, include it in `isFiltering`, and require it in `filterTree` (also inside folders whose name matches the terms); verify with `tree.test.ts` cases for domain only, domain + terms and domain + tags
- [x] 2.2 Add `domain` to `FavouritesSearch` and to `validateSearch` in `frontend/src/routes/favourites.tsx` (normalised with `normaliseDomain`, `undefined` when empty); verify that `/favourites?domain=WWW.GitHub.com` filters by `github.com` in `Favourites.test.tsx`
- [x] 2.3 In `FavouritesPage.tsx`, pass the domain into the filter, show the removable "Domain: …" chip next to the tag chips, and reset `domain` in `clearFilters`; verify in `Favourites.test.tsx` that the chip shows, that removing it and "Clear filters" both drop it, and that moving is disabled while it is set
- [x] 2.4 Add en/de strings for the chip and its remove label; verify the i18n key test (`i18n.test.ts`) passes

## 3. Domain hint

- [x] 3.1 Create `frontend/src/features/favourites/DomainHint.tsx` (props `url`, `excludeId?`, `id`; debounced URL, cached favourites tree, plural text, `<a target="_blank" rel="noopener">` to `/favourites?domain=…`, renders nothing for invalid URL / 0 / loading) and add the en/de plural strings `favourites.domainHint.count_*` and `favourites.domainHint.others_*`; verify count, "other", 0, invalid URL and the link's `href`/`target` (covered by the page tests in `Favourites.test.tsx`)
- [x] 3.2 Render `DomainHint` under the URL input in `LinkFields.tsx`, linked with `aria-describedby`, and add the optional `excludeId` prop; pass the edited node's id from `NodeDialog.tsx`; verify in `Favourites.test.tsx` that the "New link" dialog shows "3 favourites on github.com" and "Edit link" shows "2 other favourites on github.com"
- [x] 3.3 Verify the hint on the "Save link" page in `SharePage.test.tsx` (shared `https://github.com/x` with two existing `github.com` links shows "2 favourites on github.com")
- [x] 3.4 Render `DomainHint` under `UrlField` in `frontend/src/features/dashboards/widgets/link/LinkSettings.tsx` for the "URL" source; verify in `widgets/link/Link.test.tsx` that entering a URL on a known domain shows the hint and an unknown domain shows none

## 4. Docs and checks

- [x] 4.1 Describe the domain filter and the domain hint in the Favourites section of `README.md` (and the "Save link" section); verify by reading the rendered section
- [x] 4.2 Run `make check` and `make test`; verify both pass
