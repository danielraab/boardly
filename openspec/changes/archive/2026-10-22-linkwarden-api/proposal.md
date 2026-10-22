## Why

Famodash can already serve the Karakeep browser extension and mobile app at `/api/karakeep`. Linkwarden is the other popular self-hosted bookmark manager with an official mobile app and browser extension. Both of them can log in with an access token, so Famodash can serve them in the same way: one app token, bound to one folder of the favourites tree, without passwords.

## What Changes

- New Linkwarden-compatible REST API under the prefix `/api/linkwarden`. The clients get the server address `<FAMODASH_BASE_URL>/api/linkwarden` and append Linkwarden's own paths (`/api/v1/links`, `/api/v2/dashboard`, …). Only the endpoints that the official Linkwarden mobile app and browser extension (Linkwarden 2.16.3) call are implemented:
  - user and instance: `GET /api/v1/users/me`, `GET /api/v1/config`, `GET /api/v1/logins`, `GET /api/v2/dashboard`
  - links: `GET /api/v1/search`, `GET|PUT|DELETE /api/v1/links/{id}`, `POST /api/v1/links`
  - collections: `GET|POST /api/v1/collections`, `DELETE /api/v1/collections/{id}`
  - tags: `GET /api/v1/tags`, `GET|DELETE /api/v1/tags/{id}`, and `POST /api/v1/tags`, which is refused because Famodash tags only exist on links
  - previews: `GET /api/v1/archives/{id}?format=…&preview=true` serves the favicon of the link's site
- Mapping: a Linkwarden link is a link node in the token's subtree, a collection is a folder. Unlike Karakeep lists, the token's root folder is a collection too, because every Linkwarden link must be in a collection.
- Linkwarden clients parse IDs as numbers. Folders, links and tags get a stable integer ID next to their UUID, and the API uses only that number.
- Preserved formats (screenshot, PDF, readable view, single-file archive), uploads, highlights, pinned links, collection sharing, password and SSO login, billing and RSS are not supported: they answer 404, or 400 for unsupported fields.
- App tokens get the new app `linkwarden`. The create dialog offers it and shows its server address. Bearer authentication, CORS and the subtree scope also apply to `/api/linkwarden/`, and CORS allows `PUT` and `DELETE` too.

## Capabilities

### New Capabilities
- `linkwarden-api`: the Linkwarden REST endpoints, their wire format (`{"response": …}` and `{"data": …}` envelopes, integer IDs), and how Linkwarden's links, collections, tags, user and dashboard map onto the token's subtree.

### Modified Capabilities
- `app-tokens`: the app can also be `linkwarden`. The create dialog offers "Linkwarden" and shows the server address of the chosen app. Bearer authentication, CORS (now also `PUT` and `DELETE`) and the token subtree scope also cover `/api/linkwarden/`.
- `favourites`: every folder, link and tag has a stable integer ID, assigned when it is created and never reused.

## Impact

- Backend: new package `internal/linkwarden` (mapping and rules), new `httpapi/linkwarden.go` (routes, decoding, errors), `httpapi/apps.go` (CORS methods), `httpapi/router.go` (mount the prefix), `apptokens` (new app), `favourites` (integer IDs, lookups by number).
- Database: migration `0012` adds integer ID columns to `favourite_nodes` and `tags`, filled for existing rows.
- Frontend: the app choice in the create token dialog, the server address per app, the app name in the token list, and translated texts (en + de) that so far only mention Karakeep.
- README: the Linkwarden section (server address, supported features, the Linkwarden version the shapes are taken from).
- No new dependencies. The Karakeep API is unchanged.
