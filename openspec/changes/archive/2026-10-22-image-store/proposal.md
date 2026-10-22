# Proposal

## Why

The favourite explorer and the link widget should show each link's favicon, and the link widget should also take an uploaded image. Famodash has no place to keep images yet, and favourites carry no icon. Both widgets need a shared image store and a favicon fetcher first. The favourites page benefits from it too.

## What Changes

- New setting `FAMODASH_DATA_DIR` (default `./data`, `/data` in the container). Images are stored as files under `<data dir>/images/`, so they live in the Docker volume next to the database.
- New image store: an image has an unguessable ID, a content type and a source (`favicon` or `upload`). `GET /api/images/{id}` serves it to anyone, without a session, so visitors of public dashboards see the images too.
- Uploads: `POST /api/images` takes one PNG, JPEG, WebP, GIF or ICO file of at most 1 MB from a signed-in user. SVG is rejected.
- Favicons: the server fetches a site's favicon (the icon declared in the page, or `/favicon.ico`) and keeps one shared copy per origin for all users. It refreshes the copy after 30 days, and after a failed fetch it waits a day before trying again.
- `POST /api/favicons` with a URL returns the favicon's image ID for that URL's origin, fetching it if needed. This is what lets the link widget pre-fill its image.
- The server doesn't fetch from private, loopback, link-local or other non-public addresses unless `FAMODASH_FETCH_ALLOW_PRIVATE=true`. Every redirect is checked again.
- Favourites: a link node shows the favicon of its URL's origin when one is known. Creating a link or changing its URL starts a background fetch. The favourites page shows the favicon, or a globe icon when there is none.
- Uploaded images that nothing refers to are deleted after 24 hours.

## Capabilities

### New Capabilities

- `images`: storage and serving of images, uploads, the favicon fetcher with its cache and address restrictions, and the cleanup of unused uploads.

### Modified Capabilities

- `favourites`: link nodes expose and show their favicon.
- `app-hosting`: the server keeps image files in a data directory next to the database, and it gets the data directory and private-address settings.

## Impact

- Backend: new package `backend/internal/images` (store, fetcher, address guard, cleanup), migration `0009_images.sql`, new routes in `httpapi/router.go`, `config.go` (`DataDir`, `FetchAllowPrivate`), `favourites` (favicon in the tree response, fetch on create/update), `main.go` (cleanup loop).
- Frontend: `favourites/api.ts` (`iconImageId`), the favourites page (icon per link), a shared `ImageIcon` component with a globe fallback.
- Docs: README and `.env.example` for the new settings.
- New Go dependency: `golang.org/x/net/html` for parsing `<link rel="icon">`.
