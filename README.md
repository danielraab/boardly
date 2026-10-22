# Famodash

A self-hosted personal start page and bookmark manager — a mix of
[Homepage](https://gethomepage.dev) and [Karakeep](https://karakeep.app), with
real user accounts.

- **Dashboards** like Homepage, but built and edited visually in the browser (WYSIWYG) instead of YAML files.
- **Bookmarks** like Karakeep: every user has one favourites tree that holds all their links.
- **Karakeep-compatible API**, so the existing Karakeep browser extensions and mobile apps can save straight into Famodash.
- **Passwordless login** with magic links and passkeys.

## Features

### Authentication

There are no passwords. Users sign in with:

- **Magic link**: enter an email address and click the one-time login link that is sent to it.
- **Passkey** (WebAuthn): register one or more passkeys after the first login and use them to sign in.

All data (dashboards, favourites, API tokens) belongs to a single user.

### Favourites tree

Each user has exactly **one favourites tree**. It is the single source of truth for all bookmarks:

- **Folder nodes** group other nodes and can be nested to any depth.
- **Link nodes** hold an `http` or `https` URL, an optional title and any number of **tags**. Icons
  and thumbnails will follow.

The tree is managed on the **Favourites** page (`/favourites`, linked from the user menu):

- Add, rename and delete folders and links. Links open in a new tab.
- Each link shows the favicon of its site, or a globe until one is known. See
  [Favicons and images](#favicons-and-images).
- Drag items by their handle to reorder them or move them into another folder. This also works with
  the keyboard: focus the handle, press Space, choose the place with the arrow keys and press Space
  again.
- Deleting a folder that isn't empty asks whether to delete its contents too or move them to the
  folder's parent.
- Tags are created as you type them on a link, and existing tags are suggested. A tag that is no
  longer on any link is removed.

Dashboards (through navigation trees) and the Karakeep API both read from and write to this tree.
Neither keeps its own separate copy of the bookmarks.

### Dashboards

A user can create any number of dashboards on the **Dashboards** page (`/dashboards`, linked from the
user menu). Each dashboard has a title, an address, and two flags:

- **Default**: the default dashboard is shown on `/` when you're signed in. You can have at most one.
  Without one, `/` asks you to create your first dashboard or pick a default.
- **Public**: anyone with the address can see a public dashboard, also without signing in. Private
  dashboards are only visible to their owner. Everyone else gets "page not found".

Every dashboard has its own address, `/<user-slug>/<dashboard-slug>`, for example `/ann/home`:

- New accounts get a random **user slug** of 10 characters. You can change it in Settings → Profile.
  It must be unique across all users and can't be a word Famodash reserves for its own pages
  (`api`, `login`, `settings`, …).
- A **dashboard slug** must be unique among your own dashboards. When you create a dashboard, it is
  derived from the title.
- Slugs are lowercase, 1–32 characters of `a-z`, `0-9` and `-`.
- Renaming a slug breaks links that use the old address. They are not redirected, so the app asks for
  confirmation first.
- `/<user-slug>` alone redirects to `/`.

With more than one dashboard to choose from, the header shows the current dashboard's title next to
"Famodash" as a dropdown for switching. Visitors only see the owner's public dashboards there.

#### Layout

A dashboard is made of **sections**, and everything on it lives in a section. Sections sit on a
**column grid** and fill it left to right, wrapping to the next line when a section doesn't fit.
Each dashboard has its own grid settings in its dialog on `/dashboards`:

- **Columns**: 6 by default (1–12).
- **Columns on small screens**: 2 by default (1 up to the columns above).
- **Small screens are narrower than** 640, 768 (default) or 1024 px. Below that width the dashboard
  uses the small-screen column count.

A section's width always counts in page columns, also inside another section. A section is never
shown wider than its parent, so a width-6 section inside a width-3 section is 3 columns wide, and on
a 2-column phone screen every wider section takes the full width. The stored width is kept, so
raising the column count again brings the original layout back.

#### Editing

The owner of a dashboard sees an **Edit** button next to its title. It opens edit mode (the address
gets `?edit=true`), where every section shows a drag handle and a settings button (borderless
sections also get a dashed outline, so you can see them), and **Add section** buttons appear at the
top level and inside sections:

- Changes are a draft until you click **Save and exit**, which saves the whole layout at once. The
  **✕** button leaves without saving. Leaving with unsaved changes (✕, another page, reload or
  closing the tab) asks first.
- Drag sections by their handle to reorder them or to move them into another section, with
  everything inside them. With the keyboard: focus the handle, press Space, choose the place with
  the arrow keys and press Space again.
- Sections can be nested at most **5 levels** deep. A move that would go deeper is refused with a
  message.
- Deleting a section asks for confirmation and deletes everything inside it.
- If the dashboard was saved elsewhere (another tab or device) since you started editing, saving is
  refused instead of overwriting it. **Reload** then loads the current version.

A new dashboard is empty. Sections can only be added in edit mode.

### Widgets

| Widget | Description |
|---|---|
| **Link** | Icon or thumbnail, title and URL. Links always open in a new tab. |
| **Section** | A container for other widgets, including nested sections (at most 5 levels). It has a width in columns, an optional title, a border that can be turned off, and a height: fitting its content, or a fixed number of rows (4rem each) with a scrollbar when the content is taller. Inside, it has a grid as many columns wide as the section. Widgets other than sections take the full width of their section; use a borderless section to place them side by side. |
| **Navigation tree** | Shows part of the user's favourites tree. When you add one, you choose a node of the tree as its root, and the widget shows that node and everything below it. A dashboard can have several navigation trees, each with a different root. Changes to the favourites tree show up in the widget right away. |

Sections are available now. Link and navigation-tree widgets will follow. More widget types can be
added later. The widget model is meant to be extended.

### Karakeep-compatible server

Famodash exposes the [Karakeep API](https://docs.karakeep.app/api), so Karakeep clients (browser extensions, iOS/Android apps, CLI) can connect to it as if it were a Karakeep server.

- The user creates a **Karakeep API token** in the Famodash settings.
- When creating the token, the user must pick a **Karakeep root node** in their favourites tree.
- Everything the client sees and saves goes through that node:
  - bookmarks saved from a client become link nodes below the root,
  - Karakeep lists map to folder nodes below the root,
  - the client can only see the subtree below its root and nothing outside it.
- A user can have several tokens with different roots, for example one per device or per purpose.

### Themes

The palette button in the header switches between 7 color themes: **White**, **Black**, **Sand**,
**Sage** (light) and **Fjord**, **Dusk**, **Ember** (dark). The default, **System**, follows the
operating system's light or dark setting. The choice is remembered in the browser.

## Concept overview

```
User
├── Auth: magic link, passkeys
├── Favourites tree (one per user)
│   └── Folder / Link nodes …
│         ▲                    ▲
│         │ root node          │ root node
├── Dashboards                 │
│   └── Grid (6 columns)       │
│       ├── Link widget        │
│       ├── Section widget (nests widgets)
│       └── Navigation tree widget ──┘ (bound to a node)
└── Karakeep API tokens ───────┘ (each bound to a node)
```

## Tech stack

| Part | Technology |
|---|---|
| Backend | Go, standard library `net/http`, SQLite (`modernc.org/sqlite`) ([backend/](backend/)) |
| Frontend | React 19, TanStack Router, Tailwind CSS 4, Base UI, lucide-react, i18next, Vite, Biome, Vitest ([frontend/](frontend/)) |
| Specs | [OpenSpec](https://github.com/Fission-AI/OpenSpec) ([openspec/](openspec/)) |

## Development

### Workflow

All code changes go through **OpenSpec**. Every change starts as a proposal in
`openspec/changes/` (proposal, design, specs, tasks). It is then implemented and
finally archived into `openspec/specs/`.

```
/opsx:explore  →  /opsx:propose  →  /opsx:apply  →  /opsx:archive
```

Typo fixes, formatting, dependency bumps and documentation-only edits don't need a change. See
[CLAUDE.md](CLAUDE.md) for the full rules.

### Running locally

You need Go (see `backend/go.mod`), Node.js, pnpm and Docker with Compose.

```bash
cd frontend && pnpm install && cd ..
make dev-backend    # Go server on http://localhost:8080 plus the dev services below
make dev-frontend   # Vite on http://localhost:3000, proxies /api to :8080
```

Open http://localhost:3000 during development. The Go server on :8080 serves the API and the
last frontend build.

`make dev-backend` also starts the services of the `dev` profile in [compose.yaml](compose.yaml),
and removes them again when the server stops (Ctrl+C or a kill). It doesn't start the `famodash`
app service, see [Running with Docker](#running-with-docker):

| Service | URL | Purpose |
|---|---|---|
| Mailpit | http://localhost:8025 (SMTP on `localhost:1025`) | Catches all mail the dev server sends |
| sqlite-web | http://localhost:8081 | Browse and edit the dev database |

In development the database lives at `data/famodash.db` in the repository root.

| Command | What it does |
|---|---|
| `make build` | Builds the frontend into `backend/internal/web/dist` and the binary `bin/famodash`, which embeds it |
| `make test` | Go tests and frontend tests (Vitest) |
| `make check` | `go vet`, `gofmt`, Biome and the TypeScript type check |
| `make docker` | Builds the container image `famodash:dev` (`DOCKER_PLATFORM=linux/arm64` for another architecture) |

`./bin/famodash` is the whole app: it serves the REST API under `/api` and the web app, with
client-side routes falling back to `index.html`. `GET /api/health` reports whether the
database is reachable, and `famodash healthcheck` checks it from the command line: it exits 0
when the server on `FAMODASH_PORT` is healthy and 1 otherwise.

### Running with Docker

The [Dockerfile](Dockerfile) builds the frontend and the server in separate stages and produces a
small image that contains only the `famodash` binary. Build it from the repository root; nothing
but Docker is needed on the host:

```bash
make docker                      # or: docker build -t famodash:dev .
```

The container:

- needs `FAMODASH_BASE_URL` (the public URL users open). Without it, the container exits right
  away with an error naming the variable,
- listens on port 8080,
- keeps the database and the passkey provider cache in `/data`,
- runs as the non-root user `nonroot` (uid and gid 65532),
- reports its state through `famodash healthcheck`, shown as `healthy` in `docker ps`.

All other settings from [Configuration](#configuration) can be passed as environment variables.

```bash
docker run -d --name famodash \
  -p 8080:8080 \
  -e FAMODASH_BASE_URL=http://localhost:8080 \
  -v famodash-data:/data \
  famodash:dev
```

To try the image locally, the repository's [compose.yaml](compose.yaml) has a `famodash` service
that builds it and runs it on http://localhost:8080 with a named volume and no mail delivery:

```bash
docker compose up -d --build
docker compose logs famodash     # sign-in links are written here
```

It takes its base URL from `FAMODASH_COMPOSE_BASE_URL` (default `http://localhost:8080`), not
from `FAMODASH_BASE_URL`, because Compose also reads the development `.env`, where that variable
points to the Vite dev server.

For a server, use your own Compose file, for example:

```yaml
services:
  famodash:
    image: famodash:dev
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      FAMODASH_BASE_URL: https://dash.example.org
      FAMODASH_SMTP_HOST: smtp.example.org
      FAMODASH_SMTP_USERNAME: famodash@example.org
      FAMODASH_SMTP_PASSWORD: change-me
      FAMODASH_SMTP_FROM: Famodash <famodash@example.org>
    volumes:
      - famodash-data:/data

volumes:
  famodash-data:
```

**Use a named volume for `/data`** as in both examples. Docker gives a new named volume the
owner of the image's `/data` directory, so `nonroot` can write to it. A bind mount
(e.g. `./data:/data`) keeps the owner of the host directory instead, and the server then fails to
create its database. In that case either run the container as the directory's owner
(`--user "$(id -u):$(id -g)"`, or `user: "1000:1000"` in Compose), or hand the directory to the
image's user once with `sudo chown -R 65532:65532 ./data`.

### Configuration

The server is configured with environment variables. For development, `make dev-backend` loads
them from `.env` (gitignored). If there is no `.env`, it creates one by copying
[.env.example](.env.example), which you can then adjust. Variables already set in your shell take
precedence over the file.

| Variable | Default | Meaning |
|---|---|---|
| `FAMODASH_PORT` | `8080` | Port to listen on |
| `FAMODASH_DB_PATH` | `./data/famodash.db` | SQLite database file; its directory is created if needed |
| `FAMODASH_DATA_DIR` | `./data` (`/data` in the container) | Directory for files the server writes itself: favicons and uploaded images go to its `images` folder. It is created if needed. Keep it on the same volume as the database |
| `FAMODASH_LOG_FORMAT` | `json` | Log format, `json` or `text` |
| `FAMODASH_BASE_URL` | **required** | Public URL of the web app, as users open it (in development `http://localhost:3000`, set by `.env.example`). The server doesn't start without it. Used in sign-in links, for the cross-site request check (`Origin` must match) and to set the `Secure` cookie flag when it is `https` |
| `FAMODASH_SMTP_HOST` | (empty) | SMTP server for sign-in emails. Empty disables delivery: links are written to the log instead |
| `FAMODASH_SMTP_PORT` | `587` | SMTP port |
| `FAMODASH_SMTP_USERNAME` / `FAMODASH_SMTP_PASSWORD` | (empty) | SMTP login; leave empty for servers without authentication |
| `FAMODASH_SMTP_FROM` | `Famodash <noreply@localhost>` | Sender address of sign-in emails |
| `FAMODASH_SMTP_TLS` | `starttls` | `starttls` (required STARTTLS, usually port 587), `tls` (implicit TLS, usually port 465) or `none` |
| `FAMODASH_TRUST_PROXY_HEADERS` | `false` | Take the client IP from `X-Forwarded-For` (for rate limiting). Enable it only behind a reverse proxy that sets this header |
| `FAMODASH_WEBAUTHN_RP_ID` | host name of `FAMODASH_BASE_URL` | Relying party ID that passkeys are bound to. It must be the base URL's host name or a parent domain of it (e.g. `example.org` for `dash.example.org`) |
| `FAMODASH_FETCH_ALLOW_PRIVATE` | `false` | Let the server fetch favicons from private, loopback and link-local addresses (for example `http://192.168.1.10:8096`). See [Favicons and images](#favicons-and-images) |
| `FAMODASH_AAGUID_LIST_URL` | the [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids) `combined_aaguid.json` on GitHub | Where the passkey provider list is downloaded from. Set it to an empty value to turn provider detection off |

Database migrations are applied automatically on startup. There are no down migrations: to roll
back, stop the server and restore a backup of the database file.

### Sign-in

Anyone can sign up: the first sign-in with an email address creates the account.

1. On `/login`, the visitor enters their email address. The server sends a one-time link that is
   valid for 15 minutes, in the visitor's language.
   The browser that asked also gets a short-lived login-request cookie (`famodash_login_request`).
2. The link opens `/login/verify#token=…`. The token is in the URL fragment, so it never reaches
   the server's or a proxy's access logs. The page removes it from the address bar.
3. If the link is opened in the browser that requested it, the visitor is signed in right away.
   Anywhere else (another device, a mail app's built-in browser, or a mail scanner that opens
   links) the page asks the visitor to click **Complete sign-in** first, so merely opening the
   link never uses it up. After sign-in the browser has an HttpOnly session cookie
   (`famodash_session`) that expires after 30 days without use.

Link requests are limited to one per minute and five per hour per email address, and 20 per
hour per client IP.

**Development without mail.** If `FAMODASH_SMTP_HOST` is empty, the server logs a warning at
startup and writes every sign-in email (including the link) to its log. To sign in, copy the
`/login/verify#token=…` link from the log into the browser. The default `.env` sends mail to
Mailpit instead (http://localhost:8025).

**Behind a reverse proxy** (e.g. Caddy, Traefik or nginx in front of the
[Docker container](#running-with-docker)):

- Set `FAMODASH_BASE_URL` to the public URL, for example `https://dash.example.org`. A wrong
  value breaks the links in the emails and makes every sign-in request fail with
  `forbidden_origin`. The server logs the effective base URL at startup.
- Set `FAMODASH_TRUST_PROXY_HEADERS=true` so rate limits apply per real client instead of to
  the proxy as a whole. Don't enable it when the server is reachable without the proxy, because
  clients could then fake their IP.

### Favicons and images

- **Favicons** are fetched by the server when a link is added or its URL changes, and when the
  server starts for links that have none yet. It loads the page, uses the icon the page declares
  (or `/favicon.ico`) and keeps one copy per site for all users. PNG, JPEG, WebP, GIF and ICO
  icons up to 1 MB are used; SVG icons are not. A favicon is fetched again after 30 days. After
  a failed fetch, the server waits a day before trying that site again. This needs outbound
  HTTP(S) to the linked sites.
- **Private addresses are blocked by default.** The server doesn't connect to loopback, private
  (`10.x`, `172.16–31.x`, `192.168.x`), link-local, unique-local or similar addresses, also
  not after DNS resolution or a redirect. This keeps users from making the server probe your
  internal network. On a home server where the links point to local services, set
  `FAMODASH_FETCH_ALLOW_PRIVATE=true` to get their favicons too; only do that if you trust
  everyone who can sign up.
- **Images are stored as files** in `FAMODASH_DATA_DIR/images`, next to the database in the
  container's `/data` volume. Uploaded images that nothing uses are deleted after 24 hours.

### Passkeys

Signed-in users can add passkeys on **Settings → Passkeys** and then sign in on `/login` with
**Sign in with passkey** or by picking the passkey in the email field's autofill. No email
address needs to be typed.

- **Adding needs a fresh email sign-in.** A passkey can only be added within **5 minutes** after
  signing in with a magic link, in the browser session that opened the link. Signing in with a
  passkey never allows adding one, so a stolen or forgotten-open session can't add its own
  passkey. When the window is closed, the Passkeys tab offers **Send sign-in link** to get a new
  link. Deleting a passkey is allowed at any time.
- **Passkeys are bound to the host name.** They belong to the relying party ID, which is the host
  name of `FAMODASH_BASE_URL` (or `FAMODASH_WEBAUTHN_RP_ID`). If you later move Famodash to
  another host name, existing passkeys stop working. Users can still sign in with an email link
  and add new passkeys. Browsers only allow passkeys on `https://` sites and on `localhost`.
- **Provider names and icons** ("iCloud Keychain", "Google Password Manager", …) come from the
  community [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids)
  list. The server downloads it (about 7 MB) and caches it as `aaguids.json` next to the
  database, and downloads it again once the copy is a day old. This needs outbound HTTPS to `raw.githubusercontent.com`.
  Without it, passkeys are shown as "Passkey" with a generic icon and otherwise work the same.
  For servers without internet access, download `combined_aaguid.json` yourself, serve it from
  a URL the server can reach and set `FAMODASH_AAGUID_LIST_URL` to it. The list isn't shipped
  with Famodash because upstream declares no license.

### Translations

UI texts live in `frontend/src/locales/en.json` and `de.json`. Both files must have the same
keys; a test checks this.

### Themes

Each theme is a `[data-theme="…"]` block in `frontend/src/styles.css` that sets all color tokens
(`paper`, `surface`, `ink`, `muted`, `line`, `accent`, `danger`, `on-danger`, `shadow`). Components
use only these tokens, never raw Tailwind colors. To add a theme, add its block, add it to the list
in `frontend/src/theme/index.ts` and to the inline script in `frontend/index.html`, and add its name
to both locale files. Tests check that every theme defines all tokens, meets the WCAG contrast
minimums, and that the inline script matches the theme list.

## Status

Early development. Sign-in with magic links and passkeys, the settings page and the landing page work; the other features described above are the target scope and are not implemented yet.
