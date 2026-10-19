# Dashnoli

A self-hosted personal start page and bookmark manager — a mix of
[Homepage](https://gethomepage.dev) and [Karakeep](https://karakeep.app), with
real user accounts.

- **Dashboards** like Homepage, but built and edited visually in the browser (WYSIWYG) instead of YAML files.
- **Bookmarks** like Karakeep: every user has one favourites tree that holds all their links.
- **Karakeep-compatible API**, so the existing Karakeep browser extensions and mobile apps can save straight into Dashnoli.
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

Dashnoli exposes the [Karakeep API](https://docs.karakeep.app/api), so Karakeep clients (browser extensions, iOS/Android apps, CLI) can connect to it as if it were a Karakeep server.

- The user creates a **Karakeep API token** in the Dashnoli settings.
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
| Backend | Go ([backend/](backend/)) |
| Frontend | React 19, TanStack Router, Tailwind CSS 4, Vite, Biome ([frontend/](frontend/)) |
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

### Frontend

```bash
cd frontend
pnpm install
pnpm dev        # http://localhost:3000
pnpm check      # lint + format check (Biome)
pnpm build
```

### Backend

```bash
cd backend
go run .
```

## Status

Early development. The features described above are the target scope and are not implemented yet.
