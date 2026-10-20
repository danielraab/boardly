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

Database migrations are applied automatically on startup. There are no down migrations: to roll
back, stop the server and restore a backup of the database file.

### Translations

UI texts live in `frontend/src/locales/en.json` and `de.json`. Both files must have the same
keys; a test checks this.

## Status

Early development. The features described above are the target scope and are not implemented yet.
