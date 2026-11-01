# Design

## Context

- The whole favourites tree is already cached on the client (`favouritesQueryOptions`) and is used by `NodeDialog`, `SharePage` and `LinkSettings`. Counting doesn't need a server call.
- `LinkFields` holds the title/URL/tags fields and is shared by `NodeDialog` (used on `/favourites`, in `FavouriteDetails`, in the favourite explorer widget and in `LinkSettings` → "New favourite…") and `SharePage`. The link widget's plain URL field (`UrlField` in `LinkSettings.tsx`) is a separate input.
- `/favourites` filtering lives in `tree.ts` (`TreeFilter`, `filterTree`) and the route's `validateSearch` (`q`, `tags`, `folder`). The free-text search is a substring match, so it can't be used for an exact domain count. That's why a separate filter is used (see proposal).
- `checkUrl` in `validate.ts` already decides what a valid `http`/`https` URL is.

## Goals / Non-Goals

**Goals:**
- One definition of "domain", used by both the hint count and the `/favourites` filter, so the number in the hint is the number of links the filter shows.
- One hint component, reused in every place.

**Non-Goals:**
- Registrable-domain grouping (public suffix list), e.g. counting `gist.github.com` under `github.com`.
- Telling the user that the exact URL is already saved.
- Hints in API-driven paths (Karakeep, Linkwarden, Floccus, import) or for media stations and note links.
- A domain picker on `/favourites`. The filter is set only from the URL (the hint link) and removed with the chip or "Clear filters".

## Decisions

### `domainOf(url)` in `tree.ts`
Returns `new URL(url).hostname` in lower case with one leading `www.` removed, or `null` when `checkUrl` fails. `hostname` (not `host`) drops the port. IDN hosts come back punycoded from `URL`, the same way for stored and entered URLs, so they compare correctly. The `domain` search param goes through a matching `normaliseDomain` (lower case, strip `www.`), so `?domain=WWW.GitHub.com` works.

*Alternative:* compare `host` with the port. Rejected: `example.org:8080` and `example.org` are the same site for this purpose.

### `domainCount(root, domain, excludeId?)`
A tree walk that counts link nodes whose `domainOf(url)` equals `domain`, skipping `excludeId`. It's cheap enough to run on every debounced change. Trees are at most a few thousand nodes.

### Filter: `domain` added to `TreeFilter`
`TreeFilter` becomes `{ terms, tags, domain? }`. `isFiltering` includes it, and `filterTree` requires `domainOf(link.url) === domain` in addition to the existing tag and term checks. A folder whose name matches the terms still only shows links that pass the domain check, the same as with tags. Because the domain is part of the filter, the existing behaviour (folder counts while filtered, expand-all, no moving, the "nothing matches" state, selection of shown nodes) applies without extra work.

The route's `validateSearch` adds `domain: normaliseDomain(text(search.domain))`. `clearFilters` also resets `domain`. The chip is drawn in the same row as the tag chips, with an ✕ that sets `domain: undefined`.

### `DomainHint` component (`features/favourites/DomainHint.tsx`)
Props: `url`, `excludeId?`. Internally it uses `useQuery(favouritesQueryOptions)` (already in the cache), `useDebounced(url)`, `domainOf`, and `domainCount`. It renders nothing for a null domain, a count of 0 or while the tree is loading. Otherwise it renders a `<p>` with an `<a href="/favourites?domain=…" target="_blank" rel="noopener">`. A plain anchor (not the router `Link`) is used so the new tab works the same on `/share`, in dialogs and in dashboards. The text uses i18next plurals, `favourites.domainHint.count_one/_other` and `favourites.domainHint.others_one/_other` (the latter when `excludeId` is set). It is linked to the URL input with `aria-describedby`, so screen readers announce it.

- `LinkFields` gets an optional `excludeId` prop and renders `<DomainHint>` under the URL input. `NodeDialog` passes the edited node's id; `SharePage` passes nothing.
- `LinkSettings` renders `<DomainHint url={url}>` under `UrlField` for the "URL" source.

*Alternative:* router `<Link target="_blank">`. Possible, but it adds nothing for a new-tab link and needs the route's search typing in every caller.

### Debounce
The hint uses the existing `useDebounced` helper (short delay), so it doesn't flicker while the user types.

## Risks / Trade-offs

- [Subdomain links aren't counted, e.g. `docs.github.com` for `github.com`] → this is the agreed definition and is stated in the spec. Grouping by registrable domain can be added later without changing the URL format.
- [`www.` is stripped, but other host aliases (e.g. `m.` sites) are not] → acceptable. It keeps the rule simple and predictable.
- [Tree not loaded yet on `/share`] → the hint appears once the query resolves. Nothing blocks saving.
- [A `?domain=` value that's not a valid host] → it simply matches nothing and shows "No favourites match" with "Clear filters".
