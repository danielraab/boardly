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
- **Link nodes** hold a URL, a title and an optional icon or thumbnail.

Dashboards (through navigation trees) and the Karakeep API both read from and write to this tree.
Neither keeps its own separate copy of the bookmarks.

### Dashboards

A user can create any number of dashboards. Each dashboard:

- uses a **12-column grid** layout,
- is edited **WYSIWYG**: you add, drag, resize and configure widgets directly on the page,
- is made of **widgets** (see below).

### Widgets

| Widget | Description |
|---|---|
| **Link** | Icon or thumbnail, title and URL. Links always open in a new tab. |
| **Section** | A container for other widgets, including nested sections. It has an optional icon, a label and a color that is used for its border. Inside, it has its own grid. |
| **Navigation tree** | Shows part of the user's favourites tree. When you add one, you choose a node of the tree as its root, and the widget shows that node and everything below it. A dashboard can have several navigation trees, each with a different root. Changes to the favourites tree show up in the widget right away. |

More widget types can be added later. The widget model is meant to be extended.

### Karakeep-compatible server

Famodash exposes the [Karakeep API](https://docs.karakeep.app/api), so Karakeep clients (browser extensions, iOS/Android apps, CLI) can connect to it as if it were a Karakeep server.

- The user creates a **Karakeep API token** in the Famodash settings.
- When creating the token, the user must pick a **Karakeep root node** in their favourites tree.
- Everything the client sees and saves goes through that node:
  - bookmarks saved from a client become link nodes below the root,
  - Karakeep lists map to folder nodes below the root,
  - the client can only see the subtree below its root and nothing outside it.
- A user can have several tokens with different roots, for example one per device or per purpose.

## Concept overview

```
User
├── Auth: magic link, passkeys
├── Favourites tree (one per user)
│   └── Folder / Link nodes …
│         ▲                    ▲
│         │ root node          │ root node
├── Dashboards                 │
│   └── Grid (12 columns)      │
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

`make dev-backend` also starts the `dev` profile of [compose.yaml](compose.yaml), and stops it
again when the server stops (Ctrl+C or a kill):

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

`./bin/famodash` is the whole app: it serves the REST API under `/api` and the web app, with
client-side routes falling back to `index.html`. `GET /api/health` reports whether the
database is reachable.

### Configuration

The server is configured with environment variables. For development, `make dev-backend` loads
them from `.env` (gitignored). If there is no `.env`, it creates one by copying
[.env.example](.env.example), which you can then adjust. Variables already set in your shell take
precedence over the file.

| Variable | Default | Meaning |
|---|---|---|
| `FAMODASH_PORT` | `8080` | Port to listen on |
| `FAMODASH_DB_PATH` | `./data/famodash.db` | SQLite database file; its directory is created if needed |
| `FAMODASH_LOG_FORMAT` | `json` | Log format, `json` or `text` |
| `FAMODASH_BASE_URL` | `http://localhost:3000` | Public URL of the web app, as users open it. Used in sign-in links, for the cross-site request check (`Origin` must match) and to set the `Secure` cookie flag when it is `https` |
| `FAMODASH_SMTP_HOST` | (empty) | SMTP server for sign-in emails. Empty disables delivery: links are written to the log instead |
| `FAMODASH_SMTP_PORT` | `587` | SMTP port |
| `FAMODASH_SMTP_USERNAME` / `FAMODASH_SMTP_PASSWORD` | (empty) | SMTP login; leave empty for servers without authentication |
| `FAMODASH_SMTP_FROM` | `Famodash <noreply@localhost>` | Sender address of sign-in emails |
| `FAMODASH_SMTP_TLS` | `starttls` | `starttls` (required STARTTLS, usually port 587), `tls` (implicit TLS, usually port 465) or `none` |
| `FAMODASH_TRUST_PROXY_HEADERS` | `false` | Take the client IP from `X-Forwarded-For` (for rate limiting). Enable it only behind a reverse proxy that sets this header |
| `FAMODASH_WEBAUTHN_RP_ID` | host name of `FAMODASH_BASE_URL` | Relying party ID that passkeys are bound to. It must be the base URL's host name or a parent domain of it (e.g. `example.org` for `dash.example.org`) |
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

**Behind a reverse proxy** (e.g. Caddy, Traefik or nginx in front of the Docker container):

- Set `FAMODASH_BASE_URL` to the public URL, for example `https://dash.example.org`. A wrong
  value breaks the links in the emails and makes every sign-in request fail with
  `forbidden_origin`. The server logs the effective base URL at startup.
- Set `FAMODASH_TRUST_PROXY_HEADERS=true` so rate limits apply per real client instead of to
  the proxy as a whole. Don't enable it when the server is reachable without the proxy, because
  clients could then fake their IP.

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

## Status

Early development. Sign-in with magic links and passkeys, the settings page and the landing page work; the other features described above are the target scope and are not implemented yet.
