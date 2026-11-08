# Boardly

A self-hosted personal start page and bookmark manager — a mix of
[Homepage](https://gethomepage.dev) and [Karakeep](https://karakeep.app), with
real user accounts.

- **Dashboards** like Homepage, but built and edited visually in the browser (WYSIWYG) instead of YAML files.
- **Bookmarks** like Karakeep: every user has one favourites tree that holds all their links.
- **Notes** with todos, written with Markdown shortcuts and sorted into nested categories.
- **Internet radio** in a player that keeps playing while you move around the app.
- **News and podcasts** from RSS, Atom and JSON feeds, on their own page and in a dashboard widget.
- **Karakeep- and Linkwarden-compatible server**, so the Karakeep and Linkwarden browser extensions and mobile apps can save straight into Boardly, and [Floccus](#floccus-bookmark-sync) can sync your browser bookmarks with it.
- **Memos-compatible server**, so the MoeMemos Android app can write, tick off, pin and archive your notes.
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

Users can download all their data on **Settings → Import / Export** (see
[Exporting your data](#exporting-your-data)), and delete their own account on **Settings → Profile**,
after a waiting period in which they can change their mind (see
[Deleting an account](#deleting-an-account)).

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
- While you enter a link's URL, when adding or editing it, on **Save link** or in a link widget,
  a hint shows how many of your favourites are already on that domain, for example "3
  favourites on github.com". The domain is the host without a leading `www.`; subdomains count
  as their own domain. The hint opens `/favourites?domain=github.com` in a new tab, which shows
  only the links on that domain, until you remove the "Domain" chip or clear the filters.

Dashboards (through navigation trees) and the Karakeep and Linkwarden APIs all read from and write to
this tree. None of them keeps its own separate copy of the bookmarks.

#### Offline copies

Like Karakeep, Boardly keeps an **offline copy** of every link you save, so you can still read it
when the page changes or goes away:

- a **readable article**, the page's main text with its headings, lists, tables, links and images,
  cleaned of scripts, styles, forms and frames. Its images stay links to the original site and
  aren't downloaded;
- a **screenshot** of the top of the page (1280×800), when a
  [screenshot browser](#screenshots-for-offline-copies) is set up.

The copy is saved in the background when a link is added (in Boardly or by a connected app) and
when its URL changes. A new URL replaces the old copy. Links from a bookmark import or a data
import get no copy automatically. On a dashboard, open a link's details with its info button and
choose **Offline copy** to see the article or the screenshot, when it was saved, and **Refresh** to
save it again (30 refreshes at once, then one every two minutes). A refresh that fails keeps the
earlier copy. Copies are only ever shown to you, never to visitors of a public dashboard. When a
page has no preview image of its own, its screenshot becomes the link's preview.

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

#### Exporting your data

**Settings → Import / Export** also has an **Export** section that downloads all data of the
signed-in user, as a backup or to take it elsewhere. It works while a deletion of the account is
scheduled, too.

- **Without files** (the default), the download is one JSON file,
  `boardly-export-<slug>-<date>.json`.
- **With "Include files (images and attachments)"**, it's a ZIP file,
  `boardly-export-<slug>-<date>.zip`, that holds the same JSON as `boardly.json`, the uploaded
  images and link preview images as `images/<id>.<ext>`, and the note attachments as
  `attachments/<id>/<file name>`.
- **The JSON** has `"format": "boardly"` and `"version": 1`, and contains the profile (email address,
  name, slug, default folder and note category), the favourites tree with tags, all tags,
  dashboards with their widgets, note categories, notes (archived ones too, with their content
  document as stored), media sources, and the records of images and attachments. IDs are the ones
  the API uses, so references between items (a widget's folder, a note's attachments, an icon's
  image) stay intact. `file` is the path of a file in the ZIP, or `null` when the export has no
  files or the file couldn't be read.
- **Not exported:** passkeys, app tokens, sessions, sign-in links, favicons and background jobs.
- **Exports are limited per user**: by default 3 within an hour and 10 within 24 hours, counted with
  and without files (see `BOARDLY_EXPORTS_PER_HOUR` and `BOARDLY_EXPORTS_PER_DAY` in
  [Configuration](#configuration)). Over the limit, the section shows when the next export is
  possible. The counts start over when the server restarts.
- Exports can be imported again, into the same or another account and on any Boardly server (see
  [Importing a Boardly export](#importing-a-boardly-export)).

#### Importing a Boardly export

The import section on **Settings → Import / Export** also takes a Boardly export, as the JSON file
or the ZIP file. Pick the format **Boardly export (JSON or ZIP)** and the parts to import:
favourites, dashboards, notes and media sources (all are checked at first).

- **It only adds.** Nothing in the account is changed or deleted, and every imported item gets a new
  ID. Into an empty account, that is a full restore.
- **Favourites and notes** follow the rules of [Importing bookmarks](#importing-bookmarks): they go
  into the folder and category you choose, folders and categories with the same name are reused,
  and links and notes that are already saved are skipped (unless you uncheck that). Tags are matched
  by name.
- **Dashboards** are always added, with a free address (`home` becomes `home-2` if `home` is taken).
  They are never public and never the default dashboard, so nothing becomes visible to others until
  you decide.
- **Media sources** whose stream address you already have are skipped.
- **References stay intact.** A widget, an icon or an attachment that points to something imported
  in the same run points to the imported item, or to the existing item it was matched with. A
  reference to something not imported is kept as it is, so a dashboards-only import into the same
  account still shows your existing folders and notes; in another account, such a widget shows its
  usual "not found" state.
- **Files** come only from a ZIP export and must meet this server's rules for images and attachments
  (types, sizes); others are skipped and counted. Files the account already has are reused. Icons
  without their image are removed, and attachments without their file are left out of the note.
- The default folder and note category are taken from the export only when you have none yet. The
  display name and the address of your account are never changed.
- The file can have at most 10 MB, like any import. A ZIP with many files can be larger; export
  without files to get a smaller JSON file.

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
| **Media** | Stations you pick from your [media library](#media), as a list or tiles, in three sizes. Clicking one plays it in the player, or pauses it. With "Show on public dashboard" on, visitors can play them too. |

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
- **Attachments**: add files to a note with **Add attachment** in the editor's toolbar, or paste
  or drop them into the content; they go where the cursor is, or at the end. Images are shown in
  the note and open large in a viewer when clicked, where the arrow keys move between the note's
  images; audio and video play in the note; other files are shown as a card with a download
  button. The paperclip button lists the note's files. Deleting a file from the content takes it
  out of the note without deleting the file. Which files are allowed and how large they may be is
  set by the server (see [Attachments](#attachments)). Attachments are **public**:
  anyone with a file's link can open it, as with images, so note widgets on public dashboards
  can show them.
- **Locations**: add a place with **Add location** in the editor's toolbar; it goes where the
  cursor is, or at the end, and a note can have several. Set the point by typing its latitude and
  longitude, with **Use my location**, or by clicking the map, and give it an optional label. A
  location is shown as a pin with its label or coordinates; click it to see it on a map, and use
  **Edit** there to change or remove it. Maps are only loaded when you open one (see
  [Maps](#maps)).
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
  the [Memos-compatible server](#memos-compatible-server)) can refer to it. In connected apps, an
  attachment is an image or a link to its address, and stays an attachment when the app saves the
  note. A location is a `geo:` link, such as `[Stephansplatz](geo:48.208176,16.373819)`, which
  most phones open in a maps app.

The **Attachments** page (`/attachments`, linked from the user menu) lists all your files, newest
first, with the notes that use them as links. Filters show the files in use, the unused ones, those
marked for deletion and those you keep. A file that no note uses is marked for deletion once it is a
day old and deleted after `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` days (30), unless a note uses it
again or you choose **Keep**. A file can only be deleted by hand when no note uses it, archived
notes included.

### Media

The **Media** page (`/media`, linked from the user menu) is your library of internet radio
stations. Play one there or from a **Media** widget on a dashboard, and it plays in the **player
bar** at the bottom of every page. It keeps playing while you move between pages, until you close
it, sign out or reload. The bar has play/pause, volume and mute (remembered in the browser), and
works with the media keys, headset buttons and the lock screen of your device. Only one station
plays at a time.

- **Adding stations**: **Add station** searches [radio-browser.info](https://www.radio-browser.info),
  a free community directory of internet radio stations. Boardly's server sends the search there
  and loads the stations' logos, so your browser contacts neither the directory nor the logos'
  hosts, and the directory never sees your IP address; the dialog says so. This needs outbound
  HTTPS from the server to `*.api.radio-browser.info` and to the logos' hosts, and each user can
  search about 30 times a minute. You can also enter a station's
  name and stream URL by hand. Playlist files (`.pls`, `.m3u`) are resolved to the stream inside
  them, and HLS streams (`.m3u8`) work as well.
- **HTTPS only**: browsers don't play unencrypted streams on an https page, and Boardly doesn't relay
  audio through its server. When a station is saved, the server checks that its URL answers with an
  audio stream over https, and tries the `https://` version of an `http://` URL once. Stations that
  only offer `http://` can't be added; the search shows them as not playable. Audio always goes
  straight from the station to your browser, so a station sees the IP address of whoever plays it.
- **Managing**: rename a station, change its URL or its icon (a symbol or an uploaded image), drag
  stations into another order, or delete them. A station found in the search gets its logo as its
  icon; the server fetches it once and stores it with your images. The URL check and the logo
  fetch follow `BOARDLY_FETCH_ALLOW_PRIVATE` like favicons (see [Favicons and images](#favicons-and-images)),
  so stations on your local network need it set to `true`.

### Feeds

The **Feeds** page (`/feeds`, linked from the user menu) follows RSS, Atom and JSON feeds of news
and podcasts. Add the address of a feed, or of a website that advertises one. A **Feed** widget
shows the same items as a short list on a dashboard.

- **Shared storage**: each feed and item exists once per server, however many users follow it, and
  the server fetches it once. What's yours is your subscription, with its own title and icon, and
  your marks on items. Other users can't see either.
- **Refresh**: the server fetches feeds in the background every `BOARDLY_FEED_REFRESH_MINUTES`
  (60), asks whether anything changed, and waits longer after failures. Opening a dashboard never
  fetches a feed. Fetches follow `BOARDLY_FETCH_ALLOW_PRIVATE` like favicons (see
  [Favicons and images](#favicons-and-images)), so feeds on your local network need it set to
  `true`.
- **Read later and Liked**: the server keeps the newest `BOARDLY_FEED_LATEST_ITEMS` (100) items of
  each feed. An item someone marked stays until the mark is removed, also after unsubscribing from
  its feed. A mark keeps the item's title, link and summary, not the article itself.
- **Podcasts**: episodes play in the media player, straight from their host, so the host sees the
  IP address of whoever plays them. Your position is remembered in your browser. Episodes whose
  audio only exists over `http://` can't be played on an https page.
- **Public dashboards**: a Feed widget with **Show on public dashboard** shows visitors the titles
  and links of its items, and lets them play episodes. They never get the address of the feed, but
  a private podcast's episode URLs do reach them.

### Karakeep-compatible server

The Karakeep browser extension and the Karakeep mobile app (iOS/Android) can connect to Boardly as
if it were a Karakeep server. These two clients talk Karakeep's internal tRPC API, and Boardly
implements only the parts of it that they call. Of Karakeep's REST API (`/api/v1`), only the
bookmark and list endpoints that [Floccus](#floccus-bookmark-sync) uses are implemented; other
clients such as the CLI aren't supported.

Apps are connected on **Settings → Connected apps** (`/settings?tab=apps`):

- **Connect app** creates an API key. It asks for a name (e.g. "Phone"), the app (Karakeep,
  [Linkwarden](#linkwarden-compatible-server) or [Memos](#memos-compatible-server)), a **folder** of the favourites tree (the top-level folder gives access to everything),
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
  - **Photos and PDFs** shared to the app become notes in that category, with the file as their
    only content and its name as their title, using the same size and type limits as attachments
    in the web app. A note whose content is only one image or PDF shows as a file bookmark in the
    app, which opens it in its image or PDF viewer; once text is added, it is a text bookmark.
    Deleting it in the app deletes the note; the file is deleted later like any unused file.
- Deleting a folder that connected apps use warns first and revokes their keys. With "keep the
  contents", only keys for the deleted folder itself are revoked. Deleting a note category that a
  key uses as its category does the same; keys for its subcategories keep working.
- A user can have several keys with different folders, for example one per device or per purpose.

In the extension you can save the current page, see whether a page is already saved, and edit its
title, tags and list. In the app you can browse, search, save, edit and delete bookmarks, and
manage lists and tags. Bookmark cards show the link's **description** and **preview image**, or
the site's favicon when there is no preview; the description can be edited in the app. Tapping a
bookmark opens the page. Karakeep features Boardly doesn't have
answer with an error in the apps: notes on link bookmarks, adding files to existing bookmarks, archiving
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
  is in a collection. The key's folder and the folders directly in it are top-level collections,
  side by side, like Linkwarden's "Unorganized" next to the other collections; deeper folders are
  nested below them. The user's top-level folder, which has no name, is called "Unorganized". A
  link is in the collection of the folder it's directly in.
- A link saved from the extension or the app goes to the **top** of its collection. The extension's
  default collection "Unorganized" is the key's folder; another collection name that doesn't exist
  yet creates that folder in the key's folder. Saving a URL that's already there saves it again, as
  in Linkwarden. The extension's "already saved" badge works.
- Moving a link to another collection moves it to the top of that folder. A collection can be
  renamed and moved into another collection. Deleting a collection deletes its links and
  subfolders too, as in Linkwarden (earlier Boardly versions moved them to the parent folder). The
  key's folder itself can't be renamed, moved or deleted.
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

### Floccus bookmark sync

[Floccus](https://floccus.org) syncs the bookmarks of your browsers (as an extension for Chrome,
Firefox, Edge and others) and phones (as an app for Android and iOS) through a server. Boardly can
be that server, so your browser bookmarks live in your favourites tree. Floccus talks to Boardly
through the [Linkwarden](#linkwarden-compatible-server) or the
[Karakeep](#karakeep-compatible-server) API:

1. On **Settings → Connected apps**, connect an app of type **Linkwarden** (recommended) or
   **Karakeep**, and choose the folder Floccus may use, for example a new folder "Browser".
2. In Floccus, add a profile of type **Linkwarden** or **Karakeep**. Enter the server address
   shown for the key (`<BOARDLY_BASE_URL>/api/linkwarden` or `<BOARDLY_BASE_URL>/api/karakeep`)
   and the key as the password or token, and test the connection. Choose a server folder name;
   Floccus creates it in the key's folder on the first sync and finds it there again later.

Folders become folders and bookmarks become links of the favourites tree, with their tags. Renaming,
moving and deleting bookmarks and folders syncs both ways.

Prefer the **Linkwarden** type: a Boardly link is in exactly one folder, and only Linkwarden keeps a
separate link for each copy. With the **Karakeep** type, a URL that is bookmarked in two browser
folders is one link in Boardly, which Floccus then moves back and forth between the two folders on
every sync. The Karakeep type also reuses a link with the same URL that is already in the key's
folder, and moves it into the synced folder.

### Memos-compatible server

The [MoeMemos](https://github.com/mudkipme/MoeMemosAndroid) Android app can connect to Boardly as if
it were a [Memos](https://usememos.com) server, and then shows your notes as memos. Boardly answers
the Memos v1 API that MoeMemos calls and reports itself as Memos 0.31.0, the newest version
MoeMemos supports.

- On **Settings → Connected apps**, choose **Connect app** with the app **Memos**. A Memos key has
  no folder, only a **category for notes**: "All notes" (the default) or a category with its
  subcategories. The category can be changed later with **Edit**.
- In MoeMemos, enter the server address `<BOARDLY_BASE_URL>/api/memos` and log in with the key as
  **access token**.
- **Memos are notes** of the key's category. A memo's text is the note as Markdown; a note's title
  becomes a leading `# Title` line. When a memo is saved, a leading `# ` heading becomes the
  note's title, so changing it renames the note and removing it clears the title. A note without
  a title whose text starts with a `# ` heading gets that heading as its title the first time the
  app saves it. Todos keep their IDs when their text stays the same.
- New memos go into the key's category, or stay uncategorized with "All notes". A memo written
  offline keeps the time it was written as its creation time.
- **Pin** and **archive** in the app pin and archive the note, and the app's archive shows the
  archived notes.
- A change from the app always wins. A note open in the web editor at the same time shows that it
  was changed elsewhere.
- Moving a note out of the key's category removes it from the app on its next sync.
- **Attachments**: photos and files added in the app are uploaded as attachments of the note, with
  the same size and type limits as in the web app, and are added at the end of the note. The files
  at the end of a note are the memo's attachments in the app; files with text after them stay in
  the memo's text as images or links. Removing a file in the app removes it from the note, and the
  file itself is deleted later like any unused file (see the Attachments page).
- **Location**: a location set in the app is added at the end of the note, before its files, and
  the location at that place is the memo's location in the app. Other locations stay in the memo's
  text as `geo:` links. Removing the location in the app removes it from the note.
- Every memo is private, so MoeMemos' explore list stays empty. Tags aren't counted.

Requests under `/api/memos/` are authenticated only by `Authorization: Bearer <key>` and allow
cross-origin calls, as for Karakeep. Errors use Memos' format (`{"code", "message", "details"}`).
The endpoints and data shapes follow MoeMemos' `MemosV1Api.kt` (October 2026).

### API documentation

Every endpoint is described in OpenAPI 3.1 documents that the server serves without sign-in:

- `/api/openapi.yaml`: Boardly's own API, which the web app uses (session cookie)
- `/api/karakeep/openapi.yaml`, `/api/linkwarden/openapi.yaml`, `/api/memos/openapi.yaml`: the
  parts of the Karakeep, Linkwarden and Memos APIs that Boardly implements (app tokens). Their
  paths are relative to the app's server address, as the clients see them.

Open them in any OpenAPI viewer; Boardly doesn't ship one. The documents live in
`backend/internal/httpapi/openapi/`, and a test fails when a route is added or removed without
updating them.

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

Released images are published to `ghcr.io/danielraab/boardly` for `linux/amd64` and
`linux/arm64`, tagged with the version (`1.1.0`, `1.1`, `1`) and `latest`.

The [Dockerfile](Dockerfile) builds the frontend and the server in separate stages and produces a
small image that contains only the `boardly` binary. To build it yourself, run this from the
repository root; nothing but Docker is needed on the host:

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
  ghcr.io/danielraab/boardly:latest
```

To try the image locally, the repository's [compose.yaml](compose.yaml) has a `boardly` service
that runs it on http://localhost:8080 with a named volume and no mail delivery. Without `--build`
it pulls the released image; with `--build` it builds the image from your checkout:

```bash
docker compose up -d            # or: docker compose up -d --build
docker compose logs boardly     # sign-in links are written here
```

It takes its base URL from `BOARDLY_COMPOSE_BASE_URL` (default `http://localhost:8080`), not
from `BOARDLY_BASE_URL`, because Compose also reads the development `.env`, where that variable
points to the Vite dev server.

For a server, use your own Compose file, for example:

```yaml
services:
  boardly:
    image: ghcr.io/danielraab/boardly:1
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
| `BOARDLY_CONSENT_ANALYTICS` | (empty) | Name of the analytics service in `BOARDLY_ANALYTICS_SNIPPET`, for example `Google Analytics`. Setting it turns on a consent banner: the snippet's scripts only run after the visitor accepts. It needs marked scripts and a privacy policy. See [Analytics and cookie consent](#analytics-and-cookie-consent) |
| `BOARDLY_CONSENT_ANALYTICS_COOKIES` | (empty) | Comma-separated cookie names the analytics service sets, `*` as a wildcard, for example `_ga,_ga_*`. The banner lists them and they're deleted when a visitor withdraws consent. Needs `BOARDLY_CONSENT_ANALYTICS` |
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
| `BOARDLY_OFFLINE_MAX_SIZE_MB` | `5` | Largest page, in megabytes, that is read for the article of an [offline copy](#offline-copies). Larger pages get no article |
| `BOARDLY_BROWSER_URL` | (empty) | DevTools address of the headless browser that takes the screenshots of offline copies, `ws://`, `wss://`, `http://` or `https://`, for example `ws://chrome:9222`. Empty means no screenshots. See [Screenshots for offline copies](#screenshots-for-offline-copies) |
| `BOARDLY_FEED_LATEST_ITEMS` | `100` | How many of the newest items are kept per feed, 1 to 1000. Items someone flagged are kept in addition. See [Feeds](#feeds) |
| `BOARDLY_FEED_REFRESH_MINUTES` | `60` | How often feeds are refreshed, in minutes, 5 to 10080 |
| `BOARDLY_ATTACHMENT_MAX_SIZE_MB` | `5` | Largest note attachment, in megabytes |
| `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` | `30` | Days an unused attachment stays marked for deletion before it is deleted |
| `BOARDLY_ATTACHMENT_TYPES` | images except SVG, and PDF | Allowed attachment types, comma-separated content types or `<group>/*` such as `image/*`; [.env.example](.env.example) lists all known types |
| `BOARDLY_ACCOUNT_DELETION_DAYS` | `7` | Days between a user scheduling the deletion of their account and the deletion. `0` deletes the account within the hour. See [Deleting an account](#deleting-an-account) |
| `BOARDLY_EXPORTS_PER_HOUR` | `3` | Most data exports per user within 60 minutes. `0` turns this limit off. See [Exporting your data](#exporting-your-data) |
| `BOARDLY_EXPORTS_PER_DAY` | `10` | Most data exports per user within 24 hours. `0` turns this limit off |
| `BOARDLY_MAP_STYLE_URL` | OpenFreeMap's `https://tiles.openfreemap.org/styles/liberty` | [MapLibre style](https://maplibre.org/maplibre-style-spec/) that maps of note locations are drawn from. See [Maps](#maps) |
| `BOARDLY_AAGUID_LIST_URL` | the [passkey-authenticator-aaguids](https://github.com/passkeydeveloper/passkey-authenticator-aaguids) `combined_aaguid.json` on GitHub | Where the passkey provider list is downloaded from. Set it to an empty value to turn provider detection off |
| `BOARDLY_LEGAL_DIR` | (empty) | Directory with the Markdown files of the imprint and the privacy policy, for example a read-only volume. See [Imprint and privacy policy](#imprint-and-privacy-policy) |
| `BOARDLY_IMPRINT_URL` | (empty) | Absolute http(s) URL of an imprint on another site, linked instead of an `imprint*.md` file |
| `BOARDLY_PRIVACY_URL` | (empty) | Absolute http(s) URL of a privacy policy on another site, linked instead of a `privacy*.md` file |

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

### Screenshots for offline copies

The articles of [offline copies](#offline-copies) need nothing extra. **Screenshots** need a
headless Chrome that runs next to Boardly, for example
[`chromedp/headless-shell`](https://hub.docker.com/r/chromedp/headless-shell). Set
`BOARDLY_BROWSER_URL` to its DevTools address; without it, copies have no screenshots.

```yaml
services:
  boardly:
    # … as above, plus:
    environment:
      BOARDLY_BROWSER_URL: ws://chrome:9222
    networks: [default, browser]
  chrome:
    image: chromedp/headless-shell:156.0.8078.36
    restart: unless-stopped
    networks: [browser]

networks:
  browser:
    internal: true # no internet access for the browser
```

- **The browser never goes online itself.** Boardly pauses every request of the page, loads it
  with the same rules as favicons, including the [private address protection](#favicons-and-images),
  and hands the answer to the browser. Only `GET` and `HEAD` requests over `http` and `https` are
  made, at most 300 per page, each at most 10 MB and 10 seconds. Cookies aren't kept.
- **Keep the browser on its own network without internet access**, as above. Boardly's checks
  don't depend on it, but it also stops WebSockets and other traffic that Boardly can't see.
  Don't publish the browser's port: anyone who can reach it can control it.
- **Cookie banners are left out.** Requests to the hosts of common consent tools (OneTrust,
  Cookiebot, Usercentrics, Didomi, Quantcast, Sourcepoint, TrustArc, consentmanager, iubenda and
  others) are refused, and the banners of these and of common self-hosted tools (Complianz,
  Borlabs, Klaro, Cookie Notice, …) are hidden before the screenshot. Nothing is clicked and no
  consent is given, so a page that shows its content only after consent may still show a banner.
  The lists are `backend/internal/offline/consent_hosts.txt` and `consent.css`.
- One screenshot is taken at a time, and one may take at most 30 seconds. A browser that can't be
  reached only means no screenshot, with a warning in the log; the article is still saved.
- Copies are stored in `BOARDLY_DATA_DIR/offline`. Files of replaced copies and of deleted links
  are removed within about two hours.

The repository's [compose.yaml](compose.yaml) has this browser in the `screenshots` profile:
`docker compose --profile screenshots up -d`, after uncommenting `BOARDLY_BROWSER_URL`.

### Attachments

- **Files are stored** in `BOARDLY_DATA_DIR/attachments`, next to the images, and served to
  anyone at `/api/attachments/<id>/<name>`. The IDs are random, so a file can only be found
  through its link.
- **Types are checked by content** for images, PDF, audio and video, so a renamed file is caught,
  and by the file name's extension for the others (text, Office documents, archives, …). Only the
  types in `BOARDLY_ATTACHMENT_TYPES` are accepted; [.env.example](.env.example) lists all of
  them. Images (except SVG), PDF, audio and video open in the browser; every other file is
  downloaded. SVG and HTML files can contain scripts: they are off by default, and if you allow
  them they are always downloaded and never run on Boardly's address.
- **Unused files are cleaned up** on start and then hourly: a file at least a day old that no note
  uses is marked for deletion, and deleted `BOARDLY_ATTACHMENT_DELETE_AFTER_DAYS` days (30) later
  if still no note uses it and it isn't kept. A marked file still opens until it is deleted.

### Maps

Maps of note locations are vector maps drawn with [MapLibre GL](https://maplibre.org/) from the
style at `BOARDLY_MAP_STYLE_URL`, by default [OpenFreeMap](https://openfreemap.org/). The map
provider sees the viewer's IP address and the area viewed, so Boardly asks first: until the viewer
chooses **Load map**, nothing is loaded from the provider, also for visitors of public dashboards.
The location form works with its latitude and longitude fields alone.

- **Load map** allows maps until the page is reloaded. With **Always load maps in this browser**,
  the choice is stored in the browser for that provider's host, so changing the provider asks
  again. A **Don't load maps automatically** link below the map withdraws it.
- **No prompt for your own host.** When the style is on the same host name as Boardly, maps load
  without asking. Host a style and its tiles yourself (for example with OpenFreeMap's
  self-hosting setup), serve them under Boardly's host name and point `BOARDLY_MAP_STYLE_URL` at
  it. Make sure the style doesn't reference tiles, fonts or sprites on other hosts, which would
  then load without asking.

### Imprint and privacy policy

Boardly can show an imprint and a privacy policy without an admin area. Every page links to them
in its footer, next to "Open source · GitHub", and the links are visible to signed-out visitors.
A page that isn't configured has no link. Each page comes either from Markdown files or from an
external URL:

- **Markdown files** in the directory `BOARDLY_LEGAL_DIR`. The page is then shown at `/imprint` or
  `/privacy`:

  | File | Shown |
  |---|---|
  | `imprint.md`, `privacy.md` | as the fallback, in every language without its own file |
  | `imprint.de.md`, `imprint.en.md`, `privacy.de.md`, `privacy.en.md` | in that language |

  A visitor gets the file in their language, then the fallback file, then another language
  (English first). So a single `imprint.de.md` is enough for a German-only imprint.
- **An external URL** in `BOARDLY_IMPRINT_URL` or `BOARDLY_PRIVACY_URL`, for example the imprint
  on your company's website. The footer link opens it in a new tab. A URL serves every language.

```yaml
services:
  boardly:
    volumes:
      - boardly-data:/data
      - ./legal:/legal:ro          # imprint.de.md, privacy.md, ...
    environment:
      BOARDLY_LEGAL_DIR: /legal
      # or link pages on another site instead of files:
      # BOARDLY_IMPRINT_URL: https://example.at/impressum
```

- **Checked at startup.** The server reads the files once when it starts, so restart it after
  changing them. It refuses to start, naming the problem, when a page has both files and a URL,
  a `.md` file has an unknown name (such as `impressum.md` or `imprint.fr.md`), a file is empty
  or larger than 1 MiB, or a URL isn't an absolute http(s) URL. Other files, dotfiles and
  subdirectories are ignored. Mount the directory, not single files, so edited files are picked up.
- **Markdown like in notes**: headings, paragraphs, bold, italic, strikethrough, code, lists, task
  lists, quotes, tables and links (`http`, `https` and `mailto`). Images are shown as links, and
  HTML is shown as text.
- **Templates** for both pages are in [docs/legal-example](docs/legal-example). The privacy
  policy lists what Boardly processes, but it is only a starting point: adapt it to your instance
  (operator, hosting, mail provider, analytics) and have it checked if you are unsure.
- The Vite dev server (`make dev-frontend`) serves the web app without the server's settings, so
  it shows no legal links. Use a build (`make build`) to try them.

### Analytics and cookie consent

`BOARDLY_ANALYTICS_SNIPPET` adds an analytics tool to every page. Whether visitors must agree first
depends on the tool. Tools that set cookies or similar identifiers, such as Google Analytics or
Matomo with cookies, need consent under the GDPR and the ePrivacy rules (in Austria § 165 (3) TKG
2021, in Germany § 25 TDDDG). For those, turn on Boardly's consent banner:

1. Name the service in `BOARDLY_CONSENT_ANALYTICS`, and optionally its cookies in
   `BOARDLY_CONSENT_ANALYTICS_COOKIES`.
2. Mark **every** `<script>` of the snippet with `type="text/plain" data-category="analytics"`. The
   browser doesn't run these scripts, and the banner runs them once the visitor accepts.
3. Configure a privacy policy (see [Imprint and privacy policy](#imprint-and-privacy-policy)). The
   banner links to it, and it must say what the tool collects.

```yaml
environment:
  BOARDLY_CONSENT_ANALYTICS: Google Analytics
  BOARDLY_CONSENT_ANALYTICS_COOKIES: _ga,_ga_*
  BOARDLY_ANALYTICS_SNIPPET: >-
    <script type="text/plain" data-category="analytics" async
      src="https://www.googletagmanager.com/gtag/js?id=G-XXXXXXX"></script>
    <script type="text/plain" data-category="analytics">
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date()); gtag('config', 'G-XXXXXXX');
    </script>
```

For Matomo, mark its tracking `<script>` the same way, set `BOARDLY_CONSENT_ANALYTICS: Matomo`
and `BOARDLY_CONSENT_ANALYTICS_COOKIES: _pk_id.*,_pk_ses.*`, and leave out the `<noscript>` image.

- **The banner** names the service, links to the privacy policy, and offers "Accept" and "Reject"
  with equal weight, plus "Settings". It doesn't block the page, follows the app's language and
  theme, and is shown to signed-in and signed-out visitors. The footer gets a "Cookie settings"
  button, where visitors change or withdraw their choice. Withdrawing deletes the listed cookies,
  also on parent domains, and reloads the page so the tool stops.
- **The choice is stored only in the visitor's browser**, in the necessary first-party cookie
  `boardly_consent`, for 182 days. The server keeps no consent records.
- **Asking again.** When the snippet, the service name or the cookie names change, the banner asks
  every visitor again after the restart.
- **Checked at startup.** With `BOARDLY_CONSENT_ANALYTICS` set, the server refuses to start, naming
  the problem, when the snippet is empty, a `<script>` isn't marked, the snippet contains other
  elements that would load without consent (such as `<img>`, `<noscript>`, `<link>` or
  `<iframe>`), or no privacy policy is configured.
- **Without consent**, every page shows a small notice that can be closed: Boardly only uses
  necessary cookies and browser storage. With a snippet, it adds that usage statistics are
  collected without cookies. So only use a snippet without `BOARDLY_CONSENT_ANALYTICS` for a
  cookieless tool such as Umami or Plausible, and the notice stays true.
- **What stays your job.** Boardly provides the consent mechanism. The privacy policy's content
  (provider, what it collects, legal basis, transfers to other countries, how long data is kept),
  the settings in the analytics tool, and whether a cookieless tool needs consent are yours to
  decide. Have them checked if you are unsure.
- The Vite dev server doesn't insert the snippet or these settings, so it shows neither banner nor
  notice. Use a build (`make build`) to try them.

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

### Deleting an account

Users delete their account in the **Delete account** section at the end of **Settings → Profile**.

- **It needs a fresh email sign-in.** The deletion can only be scheduled within **3 minutes**
  after signing in with a magic link or one-time code. Outside that window, the section asks the
  user to sign out and sign in again by email. The user also has to type their email address to
  confirm.
- **Scheduling it takes effect right away for other devices and apps.** All other sessions of the
  user are signed out, all active app tokens are revoked, and their public dashboards are hidden
  from everyone else. The account otherwise stays fully usable, and every page shows a banner
  with the deletion date and **Cancel deletion**.
- **Cancelling** keeps the account. Revoked app tokens stay revoked and have to be created again;
  public dashboards are visible again.
- **One reminder email** is sent a day before the deletion, in the language the user had when
  scheduling it. No other emails are sent, not even after the deletion.
- **The deletion** happens `BOARDLY_ACCOUNT_DELETION_DAYS` days (7) after scheduling, checked every
  hour. It removes the account with all its dashboards, favourites, tags, notes, categories,
  attachments, media sources, uploaded images and their files, passkeys, app tokens, sessions,
  pending sign-in links and background jobs. Shared favicons are kept. The email address can sign
  up again afterwards (if registration is open), and the user's address `/<user-slug>/…` can be
  taken by another user. Old links to public dashboards then show that user's dashboards of the
  same name, if any.

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

Version 1.1.0. All features described above are implemented. Changes between versions are listed
in [CHANGELOG.md](CHANGELOG.md), and released images are published to
`ghcr.io/danielraab/boardly` (tags `latest`, `1`, `1.1` and `1.1.0`).
