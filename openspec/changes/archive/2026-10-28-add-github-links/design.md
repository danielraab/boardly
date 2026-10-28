# Design

## Context

See proposal.md for the motivation. The header (`AppShell`) is shared by every page and has no footer. The landing page (`Landing.tsx`) is rendered inside it at `/` for signed-out visitors. `UserMenu.tsx` uses Base UI `Menu` with router `Link`s rendered through `Menu.LinkItem`. A plain `<div className="mx-1 mb-1 border-t border-line" />` acts as its divider. The project uses lucide-react 1.x, which has no brand icons, so there is no GitHub icon to import. Several widgets already open external links with `target="_blank" rel="noopener noreferrer"`.

## Goals / Non-Goals

**Goals:**
- One source for the repository URL and one GitHub icon, used in all three places.
- External links that behave and read the same everywhere.

**Non-Goals:**
- A configurable source URL (e.g. `BOARDLY_SOURCE_URL`). The URL is hardcoded on purpose.
- An app-wide footer, a copyright or license line, or a version display.
- A GitHub link in the signed-out user menu.

## Decisions

### Repository URL as a constant in `src/lib/`
`src/lib/sourceCode.ts` exports `githubRepoUrl = "https://github.com/danielraab/boardly"`. It lives next to `appTitle.ts`, which also holds app-level identity, and the landing page and the user menu both import it.
*Alternative:* inline the string in each component. That was rejected because three copies would drift apart.

### `GithubIcon` as an inline-SVG component
`src/components/GithubIcon.tsx` follows `AppIcon`. It draws the official GitHub mark path with `fill="currentColor"` so it follows the theme, uses `aria-hidden`, and takes a `className` for the size. That lets it sit in for a lucide icon (`size-4 text-muted` in the menu, `size-4` in buttons).
*Alternatives:* a third-party icon package such as simple-icons, rejected as a dependency for a single path, or a generic lucide icon such as `Code`, rejected because the user wants the recognisable mark.

### External-link presentation
Each link renders the GitHub mark, the label "GitHub" and a lucide `ArrowUpRight` (aria-hidden) as the visible marker. An `aria-label` built from the label and the translated "(opens in new tab)" gives the accessible name "GitHub (opens in new tab)". An `sr-only` span was tried first, but name computation dropped the space before it ("GitHub(opens in new tab)"). The markup is small, so each place writes it out; a shared `ExternalLink` component isn't worth it for three uses with different styling.

### Landing hero button
The hero gets a flex row that wraps (`flex flex-wrap gap-3`) and holds the existing primary `Link` to `/login` and an `<a>` styled with `secondaryButton`. Wrapping keeps the 320px requirement.

### Landing footer
A `<footer>` at the end of `Landing`, with a top border (`border-t border-line`), muted small text, "Open source" + `·` + the GitHub link. It lives in `Landing.tsx`, not `AppShell`, so dashboards stay free of it.

### User menu item
`Menu.LinkItem` with `render={<a href={githubRepoUrl} target="_blank" rel="noopener noreferrer" />}` and `closeOnClick`. The existing divider markup is reused above and below it. The item appears only in the signed-in branch.

### i18n keys
`common.opensInNewTab` (shared), `landing.github`, `landing.footer.openSource`, and `shell.github`. "GitHub" is not translated, but it is still a key so that every visible string stays translated.

## Risks / Trade-offs

- [Rebranded instances link to upstream] → Intended. A self-hoster who wants their own fork's link can change the constant.
- [GitHub mark usage] → The official mark is used unmodified and only to link to the GitHub repository, which GitHub's logo guidelines allow.
- [A divider just before "Sign out"] → The order changes slightly, but "Sign out" stays last, so muscle memory still works.
