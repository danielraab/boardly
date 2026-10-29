# Boardly

A self-hosted personal start page and bookmark manager — a mix of
[Homepage](https://gethomepage.dev) and [Karakeep](https://karakeep.app), with
real user accounts.

- **Dashboards** like Homepage, but built and edited visually in the browser (WYSIWYG) instead of YAML files.
- **Bookmarks** like Karakeep: every user has one favourites tree that holds all their links.
- **Notes** with todos, written with Markdown shortcuts and sorted into nested categories.
- **Karakeep- and Linkwarden-compatible server**, so the Karakeep and Linkwarden browser extensions and mobile apps can save straight into Boardly.
- **Passwordless login** with magic links and passkeys.

## Features

### Authentication

There are no passwords. Users sign in with:

- **Magic link**: enter an email address and click the one-time login link that is sent to it,
  or type the 6-character one-time code from the same email into the login page.
- **Passkey** (WebAuthn): register one or more passkeys after the first login and use them to sign in.

By default anyone can sign up. Set `BOARDLY_ALLOW_REGISTRATION=false` to close registration once your
users have accounts (see [Sign-in](#sign-in)).

All data (dashboards, favourites, API tokens) belongs to a single user.

### Favourites tree

Each user has exactly **one favourites tree**. It is the single source of truth for all bookmarks:

- **Folder nodes** group other nodes and can be nested to any depth.
- **Link nodes** hold an `http` or `https` URL, an optional title and any number of **tags**. They
  also have a **description** and a **preview image**, which the server takes from the linked
  page. Boardly itself doesn't show or edit these yet; the
  [Karakeep](#karakeep-compatible-server) and [Linkwarden](#linkwarden-compatible-server) apps
  show them on their bookmark cards. See [Favicons and images](#favicons-and-images).

The tree is managed on the **Favourites** page (`/favourites`, linked from the user menu):

- Add, rename and delete folders and links. Links open in a new tab.
- Each link shows the favicon of its site, or a globe until one is known. The **refresh** button
  on a link loads its site's favicon again, for example after the site changed its logo. See
  [Favicons and images](#favicons-and-images).
- Drag items by their handle to reorder them or move them into another folder. This also works with
  the keyboard: focus the handle, press Space, choose the place with the arrow keys and press Space
  again.
- Deleting a folder that isn't empty asks whether to delete its contents too or move them to the
  folder's parent.
- Tags are created as you type them on a link, and existing tags are suggested. A tag that is no
  longer on any link is removed.

Dashboards (through navigation trees) and the Karakeep and Linkwarden APIs all read from and write to
this tree. None of them keeps its own separate copy of the bookmarks.

#### Importing bookmarks

Bookmarks, and the text bookmarks of Karakeep as notes, can be imported once from an export file on
**Settings → Import / Export** (`/settings?tab=data`). Nothing is synced afterwards. The supported
files are:

- **Karakeep (JSON)**: Karakeep's JSON export. It contains your lists, so use it rather than
  Karakeep's HTML export, which has no lists.
- **Linkwarden (JSON)**: Linkwarden's data export, with collections, links and tags.
- **Bookmarks file (HTML)**: the Netscape bookmark format that Karakeep, Linkwarden and web browsers
  export.

You choose the folder to import into. The top-level folder is allowed. The import works like this:

- The file's folders (Karakeep lists, Linkwarden collections, browser folders) are created in that
  folder. A folder that already exists there with the same name, ignoring case, is used instead of a
  new one. Folders without links are created too.
- Links keep their title, tags, description and creation date, and the order of the file. They are
  added after the folder's existing content.
- A link whose URL is already in your favourites, or appears earlier in the file, is skipped, and the
  existing link is left as it is. Importing the same file twice therefore adds nothing the second
  time. Uncheck "Skip links and notes that are already saved" to import them anyway.
- A Karakeep bookmark that is in several lists goes into the first one. Karakeep notes and summaries
  are added to the description (`Note: …`, `Summary: …`), which is cut to 2048 characters.
- Karakeep **text bookmarks become notes**, in a category you choose on the import form (or the top
  level). A note keeps the bookmark's title, creation date and text, read as Markdown, with
  Karakeep's note and summary as `Note: …` and `Summary: …` paragraphs and the tags as a last line
  `#tag #other-tag` (spaces in a tag become `-`). Lists that hold text bookmarks, also in their
  sublists, are created as categories below the chosen one, reusing categories of the same name,
  ignoring case. Lists with only links create no category. A note with the same title and content as
  one of your notes, or one earlier in the file, is skipped, and a note larger than 512 KB is left
  out. An archived text bookmark becomes an archived note, shown under **Archived** on the Notes
  page.
- Not imported: Karakeep asset bookmarks, Karakeep smart lists, Linkwarden uploads without a URL
  (PDFs, images), and the archived, favourited and pinned flags of links.
- The import is all or nothing, and the file can have at most 10 MB. Afterwards, the page shows how
  many links, folders, notes and categories were added and what was skipped.
- Favicons, descriptions and preview images of the imported links are fetched in the background, as
  for any new link. A description from the file is kept.

### Dashboards

A user can create any number of dashboards on the **Dashboards** page (`/dashboards`, linked from the
user menu). Each dashboard has a title, an address, and two flags:

- **Default**: the default dashboard is shown on `/` when you're signed in. You can have at most one.
  Without one, `/` asks you to create your first dashboard or pick a default.
- **Public**: anyone with the address can see a public dashboard, also without signing in. Private
  dashboards are only visible to their owner. Everyone else gets "page not found".

Every dashboard has its own address, `/<user-slug>/<dashboard-slug>`, for example `/ann/home`:

- New accounts get a random **user slug** of 10 characters. You can change it in Settings → Profile.
  It must be unique across all users and can't be a word Boardly reserves for its own pages
  (`api`, `login`, `settings`, …).
- A **dashboard slug** must be unique among your own dashboards. When you create a dashboard, it is
  derived from the title.
- Slugs are lowercase, 1–32 characters of `a-z`, `0-9` and `-`.
- Renaming a slug breaks links that use the old address. They are not redirected, so the app asks for
  confirmation first.
- `/<user-slug>` alone redirects to `/`.

With more than one dashboard to choose from, the header shows the current dashboard's title next to
"Boardly" as a dropdown for switching. Visitors only see the owner's public dashboards there.

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
| **Navigation tree** | Shows part of the user's favourites tree. When you add one, you choose a node of the tree as its root, and the widget shows that node and everything below it. A dashboard can have several navigation trees, each with a different root. Changes to the favourites tree show up in the widget right away. On your own dashboard, the + button next to "Expand all" adds a link to the shown folder or another one. |
| **Link list** | Shows the links of one folder of the favourites tree as a flat list, optionally including the links of all its subfolders. The links are shown as a list (title and URL), a compact list (title only) or tiles, in three sizes. Links open in a new tab. Like the navigation tree, it is hidden from visitors of a public dashboard unless "Show on public dashboard" is on. |

Sections are available now. Link and navigation-tree widgets will follow. More widget types can be
added later. The widget model is meant to be extended.

### Notes

The **Notes** page (`/notes`, linked from the user menu) holds your notes as cards, last edited
first. A card shows the note's icon, title and the beginning of its content on the note's colour;
click it to open the note.

- **Writing**: a note has an optional title and rich content. Markdown shortcuts format while you
  type: `# `, `## ` and `### ` for headings, `- ` or `* ` for bullet lists, `1. ` for numbered
  lists, `- [ ] ` or `[] ` for todos (`- [x] ` for a done one), `> ` for quotes, ```` ``` ```` for
  code blocks, `---` for a divider, and `**bold**`, `*italic*`, `~~strikethrough~~`, `` `code` ``
  and `[text](https://…)` inline. A toolbar offers the same, which helps on phones. Tick a todo by
  clicking its checkbox. Links open with Ctrl or Cmd held. Changes are saved a second after you
  stop typing. A note without a title is shown by its first line.
- **Icon and colour**: a note can have a symbol or an uploaded image as its icon, and one of nine
  colours. The colours have a light and a dark shade, so they suit every theme.
- **Categories**: each note can be in one category. Categories have a title, an optional icon and
  colour, and can be nested. They are listed in the side panel of the Notes page (a drawer on
  phones), with the number of notes in each, including their subcategories. Choose one to show its
  notes; create, edit, move and delete categories from there. A category stays when it has no
  notes. Deleting one makes its notes uncategorized and moves its subcategories up a level. The
  category of a note is shown as a badge; click the badge to change it.
- **Pinning and archiving**: the menu of a note has **Pin note** and **Archive note**. Pinned
  notes come first and show a pin, on the Notes page within the chosen category and in the note
  widgets of dashboards. Archived notes leave "All notes", "Uncategorized" and the categories, and
  their counts; the **Archived** entry at the end of the side panel shows them. Note widgets still
  show archived notes, dimmed and marked as archived. Pinning, archiving and their opposites count
  as changes, so the note moves up in "last changed" lists.
- **Getting notes in**: share text into the installed app (see [Install as an app](#install-as-an-app))
  to save it as a note, save a text bookmark in a connected Karakeep app (see
  [Karakeep-compatible server](#karakeep-compatible-server)), or import the text bookmarks of a
  Karakeep export (see [Importing bookmarks](#importing-bookmarks)). Notes saved from a share start in your **default
  category for new notes**, which you choose in Settings → Profile.
- **Storage**: note content is stored as a JSON document of Boardly's own, not as Markdown. Each
  todo has a stable ID, so later features (todo widgets, link tagging, Markdown import and export,
  a Memos-compatible API) can refer to it.

### Karakeep-compatible server

The Karakeep browser extension and the Karakeep mobile app (iOS/Android) can connect to Boardly as
if it were a Karakeep server. These two clients talk Karakeep's internal tRPC API, and Boardly
implements only the parts of it that they call. Karakeep's REST API (`/api/v1`) and other clients
such as the CLI aren't supported.

Apps are connected on **Settings → Connected apps** (`/settings?tab=apps`):

- **Connect app** creates an API key. It asks for a name (e.g. "Phone"), the app (Karakeep or
  [Linkwarden](#linkwarden-compatible-server)), a **folder** of the favourites tree (the top-level folder gives access to everything),
  for Karakeep a **category for notes** ("All notes", the default, or a category with its
  subcategories) and an expiry: 30 days, 90 days (default), 1 year, never or a date.
- The key (`bdly_…`) is shown **once**, together with the server address to enter in the app:
  `<BOARDLY_BASE_URL>/api/karakeep`. Boardly stores only a hash of it. In the app, enter the
  server address and log in with **API key**; email and password login isn't available, because
  Boardly has no passwords.
- **Edit** changes the folder and the category of an active key. The app keeps its key and sees
  the new folder and category with its next request. Name, app and expiry can't be changed. An
  active key can be **revoked**, which is final. Only revoked or expired keys can be deleted from
  the list.
- Everything the client sees and saves goes through its folder, and it can't see or change anything
  outside it:
  - **Bookmarks** are all links in the folder, at any depth. A page saved from the extension or
    the app becomes a link at the **top** of the folder; saving a URL that's already there returns
    the existing bookmark.
  - **Lists** are the folders below it, nested like the folders. A bookmark is in the list whose
    folder it's directly in, so it is in at most one list: adding it to a list **moves** the link
    into that folder, and removing it moves the link back to the top of the app's folder.
    Deleting a list keeps its links and subfolders by moving them to the parent folder.
  - **Tags** are the tags on the folder's links. Deleting a tag in the app removes it only from
    links in the folder. A tag that links outside the folder also have can't be renamed from the
    app, and tags are created by adding them to a bookmark, not on their own.
  - **Text bookmarks** are the notes of the key's category and its subcategories, or all notes,
    with their content as Markdown. A text bookmark saved in the app becomes a note in that
    category (uncategorized with "All notes"). They show up with the links, newest first, and in
    searches by their title and text. Editing a note's text in the app keeps the IDs of its todos
    whose text stays the same; formatting Markdown can't express exactly, such as empty lines
    between paragraphs, may be simplified. Notes have no tags and aren't in lists. Archiving a
    text bookmark in the app archives the note, and making it a favourite pins it; the app's
    archive and favourites lists show the archived and pinned notes.
- Deleting a folder that connected apps use warns first and revokes their keys. With "keep the
  contents", only keys for the deleted folder itself are revoked. Deleting a note category that a
  key uses as its category does the same; keys for its subcategories keep working.
- A user can have several keys with different folders, for example one per device or per purpose.

In the extension you can save the current page, see whether a page is already saved, and edit its
title, tags and list. In the app you can browse, search, save, edit and delete bookmarks, and
manage lists and tags. Bookmark cards show the link's **description** and **preview image**, or
the site's favicon when there is no preview; the description can be edited in the app. Tapping a
bookmark opens the page. Karakeep features Boardly doesn't have
answer with an error in the apps: notes on link bookmarks, file bookmarks and assets, archiving
and favouriting links, highlights, reading progress, AI tagging and summaries, smart and public lists, list
icons and descriptions, and list sharing.

Requests under `/api/karakeep/` are authenticated only by `Authorization: Bearer <key>`, never by
the session cookie, and allow cross-origin calls (CORS) so browser extensions can use them. The
procedures and data shapes follow the Karakeep source at commit `85054d9` (October 2026); a later
client release that renames a procedure shows up as a "not found" error.

### Linkwarden-compatible server

The Linkwarden mobile app (iOS/Android) and the Linkwarden browser extension can connect to Boardly
as if it were a Linkwarden server. Boardly implements only the parts of Linkwarden's REST API that
these two clients call.

Keys are created on **Settings → Connected apps** as for Karakeep, with the app **Linkwarden**. The
server address is `<BOARDLY_BASE_URL>/api/linkwarden`. In the mobile app, enter it as the server and
log in with an **access token**; in the extension, choose the method **API Key**. Username and
password login answers with a message to use an access token, because Boardly has no passwords.

Everything goes through the key's folder, as for Karakeep, with these differences:

- **Collections** are the key's folder **and** the folders below it, because every Linkwarden link
  is in a collection. The key's folder is the top collection; the user's top-level folder, which has
  no name, is called "Unorganized". A link is in the collection of the folder it's directly in.
- A link saved from the extension or the app goes to the **top** of its collection. The extension's
  default collection "Unorganized" is the key's folder; another collection name that doesn't exist
  yet creates that folder in the key's folder. Saving a URL that's already there saves it again, as
  in Linkwarden. The extension's "already saved" badge works.
- Moving a link to another collection moves it to the top of that folder. Deleting a collection
  keeps its links and subfolders by moving them to the parent folder (Linkwarden would delete the
  links). The key's folder itself can't be deleted.
- **Tags** work as for Karakeep: deleting a tag in the app removes it only from links in the folder,
  and tags are created by adding them to a link, not on their own.
- A link's **description** is shown and can be edited in the app and the extension.
- Link **previews** in the app are the link's preview image, or else the favicon of its site when
  Boardly knows one in PNG, JPEG, GIF or WebP. With **upload screenshot** turned on in the
  extension, the screenshot becomes the link's preview. Boardly stores no PDFs or readable copies,
  so tapping a link opens the page.
- The app's dashboard shows statistics and the 16 newest links. Its layout can't be changed.

Linkwarden features Boardly doesn't have answer with an error or "not found" in the clients:
preserved formats other than the screenshot (PDF, readable view, single file) and re-preserving,
other file uploads, highlights, pinned links, collection sharing, public collections and RSS feeds,
link and collection icons and colors, bulk actions, tag merging, SSO and registration.

Requests under `/api/linkwarden/` are authenticated only by `Authorization: Bearer <key>` and allow
cross-origin calls, as for Karakeep. IDs are numbers, because the Linkwarden clients expect them:
every folder, link and tag has a stable integer ID next to its internal one. The endpoints and data
shapes follow Linkwarden v2.16.3 (commit `952ac45`, September 2026), and Boardly reports that
version to the clients.

### Themes

The palette button in the header switches between 7 color themes: **White**, **Black**, **Sand**,
**Sage** (light) and **Fjord**, **Dusk**, **Ember** (dark). The default, **System**, follows the
operating system's light or dark setting. The choice is remembered in the browser.

### Install as an app

Boardly can be installed as a web app (PWA). Chrome, Edge and Android browsers offer this with
their own install button once the app is served over HTTPS. On iOS, use Share → "Add to Home
Screen". The installed app opens your default dashboard in its own window, and its title bar
follows the theme. It doesn't work offline.

- **Share links and notes into Boardly** (Android, ChromeOS): Boardly appears in the system share
  sheet. Sharing a page opens **Save link** (`/share`) with its URL and title filled in. Pick a
  folder, add tags and save. The page starts in your **default folder for new links**, which you
  choose in Settings → Profile and which applies on every device. Without one, or once it's
  deleted, it starts in the root folder.
- Sharing text without a web address, such as a message or a selection, opens the page as **Save
  note**, with the title and text filled in; the switch at the top changes between link and note,
  keeping what you entered in both. The text is saved as it is, with web addresses as links, in
  your **default category for new notes** (Settings → Profile), or without a category.
- **Shortcuts** on the app icon (long-press on Android, right-click on desktop): **Add link**,
  **Favourites** and **Dashboards**.

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
├── Notes ──── Note categories (nested, one per note)
└── Connected apps (API keys) ─┘ (each bound to a folder)
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
and removes them again when the server stops (Ctrl+C or a kill). It doesn't start the `boardly`
app service, see [Running with Docker](#running-with-docker):

| Service | URL | Purpose |
|---|---|---|
| Mailpit | http://localhost:8025 (SMTP on `localhost:1025`) | Catches all mail the dev server sends |
| sqlite-web | http://localhost:8081 | Browse and edit the dev database |

In development the database lives at `data/boardly.db` in the repository root.

| Command | What it does |
|---|---|
| `make build` | Builds the frontend into `backend/internal/web/dist` and the binary `bin/boardly`, which embeds it |
| `make test` | Go tests and frontend tests (Vitest) |
| `make check` | `go vet`, `gofmt`, Biome and the TypeScript type check |
| `make docker` | Builds the container image `boardly:dev` (`DOCKER_PLATFORM=linux/arm64` for another architecture) |

`./bin/boardly` is the whole app: it serves the REST API under `/api` and the web app, with
client-side routes falling back to `index.html`. `GET /api/health` reports whether the
database is reachable, and `boardly healthcheck` checks it from the command line: it exits 0
when the server on `BOARDLY_PORT` is healthy and 1 otherwise.

### Running with Docker

The [Dockerfile](Dockerfile) builds the frontend and the server in separate stages and produces a
small image that contains only the `boardly` binary. Build it from the repository root; nothing
but Docker is needed on the host:

```bash
make docker                      # or: docker build -t boardly:dev .
```

The container:

- needs `BOARDLY_BASE_URL` (the public URL users open). Without it, the container exits right
  away with an error naming the variable,
- listens on port 8080,
- keeps the database and the passkey provider cache in `/data`,
- runs as the non-root user `nonroot` (uid and gid 65532),
- reports its state through `boardly healthcheck`, shown as `healthy` in `docker ps`.

All other settings from [Configuration](#configuration) can be passed as environment variables.

```bash
docker run -d --name boardly \
  -p 8080:8080 \
  -e BOARDLY_BASE_URL=http://localhost:8080 \
  -v boardly-data:/data \
  boardly:dev
```

To try the image locally, the repository's [compose.yaml](compose.yaml) has a `boardly` service
that builds it and runs it on http://localhost:8080 with a named volume and no mail delivery:

```bash
docker compose up -d --build
docker compose logs boardly     # sign-in links are written here
```

It takes its base URL from `BOARDLY_COMPOSE_BASE_URL` (default `http://localhost:8080`), not
from `BOARDLY_BASE_URL`, because Compose also reads the development `.env`, where that variable
points to the Vite dev server.

For a server, use your own Compose file, for example:

```yaml
services:
  boardly:
    image: boardly:dev
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      BOARDLY_BASE_URL: https://dash.example.org
      BOARDLY_SMTP_HOST: smtp.example.org
      BOARDLY_SMTP_USERNAME: boardly@example.org
      BOARDLY_SMTP_PASSWORD: change-me
      BOARDLY_SMTP_FROM: Boardly <boardly@example.org>
    volumes:
      - boardly-data:/data

volumes:
  boardly-data:
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
| `BOARDLY_PORT` | `8080` | Port to listen on |
| `BOARDLY_DB_PATH` | `./data/boardly.db` | SQLite database file; its directory is created if needed |
| `BOARDLY_DATA_DIR` | `./data` (`/data` in the container) | Directory for files the server writes itself: favicons and uploaded images go to its `images` folder. It is created if needed. Keep it on the same volume as the database |
| `BOARDLY_LOG_FORMAT` | `json` | Log format, `json` or `text` |
| `BOARDLY_LOG_REQUESTS` | `all` | Which HTTP requests get a log line: `all`, `errors` (only responses with status 500 or higher) or `none`. A successful health check (`GET /api/health` answered with 200) is never logged. Other log output is unchanged. If you turn request lines off, a reverse proxy in front of Boardly can keep access logs |
| `BOARDLY_BASE_URL` | **required** | Public URL of the web app, as users open it (in development `http://localhost:3000`, set by `.env.example`). The server doesn't start without it. Used in sign-in links, for the cross-site request check (`Origin` must match) and to set the `Secure` cookie flag when it is `https` |
| `BOARDLY_APP_TITLE` | `Boardly` | Name the web app, its installed app (manifest) and sign-in emails show, for example `Boardly Staging` to tell a staging system apart. Blank means the default. Development with the Vite dev server always shows `Boardly` |
| `BOARDLY_ANALYTICS_SNIPPET` | (empty) | HTML inserted unchanged right before `</head>` of the web app, for example the `<script>` tag of Umami, Plausible or Matomo. It is read at startup, so changing it needs a restart, not a new build. The web app loads it once per full page load, so let the tool track page changes itself (most do by default). The snippet runs on every page, signed-in ones included, with full access to them: only put code there you trust. The Vite dev server doesn't insert it |
| `BOARDLY_ALLOW_REGISTRATION` | `true` | Whether the first sign-in with a new email address creates an account. `false` closes registration: only existing users can sign in, and new addresses get an email saying that registration is closed. See [Sign-in](#sign-in) |
| `BOARDLY_SMTP_HOST` | (empty) | SMTP server for sign-in emails. Empty disables delivery: links are written to the log instead |
| `BOARDLY_SMTP_PORT` | `587` | SMTP port |
| `BOARDLY_SMTP_USERNAME` / `BOARDLY_SMTP_PASSWORD` | (empty) | SMTP login; leave empty for servers without authentication |
| `BOARDLY_SMTP_FROM` | `Boardly <noreply@localhost>` | Sender address of sign-in emails |
| `BOARDLY_SMTP_TLS` | `starttls` | `starttls` (required STARTTLS, usually port 587), `tls` (implicit TLS, usually port 465) or `none` |
| `BOARDLY_TRUST_PROXY_HEADERS` | `false` | Take the client IP from `X-Forwarded-For` (for rate limiting). Enable it only behind a reverse proxy that sets this header |
| `BOARDLY_WEBAUTHN_RP_ID` | host name of `BOARDLY_BASE_URL` | Relying party ID that passkeys are bound to. It must be the base URL's host name or a parent domain of it (e.g. `example.org` for `dash.example.org`) |
| `BOARDLY_FETCH_ALLOW_PRIVATE` | `false` | Let the server fetch favicons and link previews from private, loopback and link-local addresses (for example `http://192.168.1.10:8096`). See [Favicons and images](#favicons-and-images) |
| `BOARDLY_PREVIEW_MAX_SIZE_MB` | `10` | Largest preview image, in megabytes, that the server downloads from a page or accepts from a connected app before converting it |
| `BOARDLY_PREVIEW_MAX_PIXELS` | `25000000` | Largest preview image, as width times height, that the server decodes. Lower it on machines with little memory: decoding takes about 4 bytes per pixel |
| `BOARDLY_PREVIEW_WIDTH` | `800` | Width, in pixels, that larger preview images are scaled down to |
| `BOARDLY_AAGUID_LIST_URL` | the [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids) `combined_aaguid.json` on GitHub | Where the passkey provider list is downloaded from. Set it to an empty value to turn provider detection off |

Database migrations are applied automatically on startup. There are no down migrations: to roll
back, stop the server and restore a backup of the database file.

#### Upgrading from Famodash

Boardly was called Famodash before. The rename has no fallbacks, so an existing install needs
these steps once:

- Rename every `FAMODASH_*` variable to `BOARDLY_*`.
- Rename the database file `famodash.db` to `boardly.db` (in the container: `/data/famodash.db`),
  or set `BOARDLY_DB_PATH` to the old file.
- Sign in again: the session cookie has a new name.
- Create new app tokens for the Karakeep and Linkwarden apps: tokens now start with `bdly_`, and
  old `famo_` tokens are no longer accepted.
- Theme and language choices stored in the browser reset once.

### Sign-in

Anyone can sign up: the first sign-in with an email address creates the account.

With `BOARDLY_ALLOW_REGISTRATION=false`, registration is closed. Users with an account sign in as
before. An address without an account gets an email, in the visitor's language, saying that
registration is closed, with no link and no code. The login page responds the same either way, so it
doesn't reveal which addresses have an account. These requests count against the same rate limits.
A link or code that was sent to an address without an account before registration was closed no
longer works.

1. On `/login`, the visitor enters their email address. The server sends a one-time link that is
   valid for 15 minutes, in the visitor's language.
   The browser that asked also gets a short-lived login-request cookie (`boardly_login_request`).
2. The link opens `/login/verify#token=…`. The token is in the URL fragment, so it never reaches
   the server's or a proxy's access logs. The page removes it from the address bar.
3. If the link is opened in the browser that requested it, the visitor is signed in right away.
   Anywhere else (another device, a mail app's built-in browser, or a mail scanner that opens
   links) the page asks the visitor to click **Complete sign-in** first, so merely opening the
   link never uses it up. After sign-in the browser has an HttpOnly session cookie
   (`boardly_session`) that expires after 30 days without use.
4. Instead of the link, the visitor can type the one-time code from the email (for example
   `K7Q-4MZ`) into the login page that is still waiting for it. This helps when the link opens in
   another browser or outside the installed app. The code only works in the browser that requested
   the email and for that address. Link and code expire together, and using either one uses up
   both. After 5 wrong codes, that browser's codes for the address stop working, while the link
   still works.

Link requests are limited to one per minute and five per hour per email address, and 20 per
hour per client IP.

**Development without mail.** If `BOARDLY_SMTP_HOST` is empty, the server logs a warning at
startup and writes every sign-in email (including the link and the code) to its log. To sign in,
copy the `/login/verify#token=…` link from the log into the browser, or type the code into the
login page. The default `.env` sends mail to
Mailpit instead (http://localhost:8025).

**Behind a reverse proxy** (e.g. Caddy, Traefik or nginx in front of the
[Docker container](#running-with-docker)):

- Set `BOARDLY_BASE_URL` to the public URL, for example `https://dash.example.org`. A wrong
  value breaks the links in the emails and makes every sign-in request fail with
  `forbidden_origin`. The server logs the effective base URL at startup.
- Set `BOARDLY_TRUST_PROXY_HEADERS=true` so rate limits apply per real client instead of to
  the proxy as a whole. Don't enable it when the server is reachable without the proxy, because
  clients could then fake their IP.

### Favicons and images

- **Favicons** are fetched by the server when a link is added or its URL changes. It loads the
  page, uses the icon the page declares (or `/favicon.ico`) and keeps one copy per site for all
  users. PNG, JPEG, WebP, GIF and ICO icons up to 1 MB are used; SVG icons are not. After a failed
  fetch, the server waits a day before trying that site again.
- **Favicons aren't refreshed automatically.** A stored favicon is kept until someone presses the
  refresh button of a link to that site on the Favourites page; that also works during the day
  after a failed fetch. A user can refresh 30 favicons right away and then one every two minutes.
  Links that existed before a favicon could be fetched get one only through a refresh or a URL
  change.
- **Descriptions and preview images** of links are taken from the linked page when a link is
  added or its URL changes: the description from `og:description` or the page's meta
  description, the image from `og:image` or `twitter:image`. A description that a connected app
  set is never overwritten. A preview is only fetched for a link that has none; changing the URL
  removes the old one first. A screenshot from the Linkwarden extension always replaces it. Each
  link has its own preview, also when two users save the same page. Links added before this
  feature don't get a description or preview until their URL changes.
- **Preview images are converted** in the background into a JPEG of at most
  `BOARDLY_PREVIEW_WIDTH` pixels wide (800), upright and with transparent areas on white. PNG,
  JPEG, WebP and GIF images up to `BOARDLY_PREVIEW_MAX_SIZE_MB` (10 MB) and
  `BOARDLY_PREVIEW_MAX_PIXELS` (25 million pixels) are used; others are skipped. One image is
  converted at a time.
- **Background work survives restarts.** Favicon fetches, page fetches and conversions wait in the
  database, so a restart or a busy server doesn't lose any. A job that was interrupted three times
  is dropped with a warning in the log.
- **Private addresses are blocked by default.** The server doesn't connect to loopback, private
  (`10.x`, `172.16–31.x`, `192.168.x`), link-local, unique-local or similar addresses, also
  not after DNS resolution or a redirect. This keeps users from making the server probe your
  internal network. On a home server where the links point to local services, set
  `BOARDLY_FETCH_ALLOW_PRIVATE=true` to get their favicons and previews too; only do that if you
  trust everyone who can sign up.
- **Images are stored as files** in `BOARDLY_DATA_DIR/images`, next to the database in the
  container's `/data` volume. Uploaded images that nothing uses (no widget, note or note category)
  are deleted after 24 hours, and previews that no link uses any more after about an hour.

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
  name of `BOARDLY_BASE_URL` (or `BOARDLY_WEBAUTHN_RP_ID`). If you later move Boardly to
  another host name, existing passkeys stop working. Users can still sign in with an email link
  and add new passkeys. Browsers only allow passkeys on `https://` sites and on `localhost`.
- **Provider names and icons** ("iCloud Keychain", "Google Password Manager", …) come from the
  community [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids)
  list. The server downloads it (about 7 MB) and caches it as `aaguids.json` next to the
  database, and downloads it again once the copy is a day old. This needs outbound HTTPS to `raw.githubusercontent.com`.
  Without it, passkeys are shown as "Passkey" with a generic icon and otherwise work the same.
  For servers without internet access, download `combined_aaguid.json` yourself, serve it from
  a URL the server can reach and set `BOARDLY_AAGUID_LIST_URL` to it. The list isn't shipped
  with Boardly because upstream declares no license.

### Translations

UI texts live in `frontend/src/locales/en.json` and `de.json`. Both files must have the same
keys; a test checks this.

### Themes

Each theme is a `[data-theme="…"]` block in `frontend/src/styles.css` that sets all color tokens
(`paper`, `surface`, `ink`, `muted`, `line`, `accent`, `danger`, `on-danger`, `shadow`). Components
use only these tokens, never raw Tailwind colors. To add a theme, add its block, add it with its
`paper` colour to the list in `frontend/src/theme/index.ts` and to the inline script in
`frontend/index.html`, and add its name to both locale files. Tests check that every theme defines
all tokens, meets the WCAG contrast minimums, and that the inline script matches the theme list.

### App icons

The home-screen icons in `frontend/public/icons/` are rendered from `frontend/icons/*.svg`, which
use the same drawing as `frontend/public/favicon.svg`. After changing the drawing, run `pnpm icons`
in `frontend/` (it needs Chromium; set `CHROMIUM` to use another binary) and commit the PNGs. The
web app manifests are `frontend/public/manifest.en.webmanifest` and `manifest.de.webmanifest`.

## Status

Early development. Sign-in with magic links and passkeys, the settings page and the landing page work; the other features described above are the target scope and are not implemented yet.
