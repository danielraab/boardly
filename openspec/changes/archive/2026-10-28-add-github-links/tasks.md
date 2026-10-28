# Tasks

## 1. Shared pieces

- [x] 1.1 Add `frontend/src/lib/sourceCode.ts` exporting `githubRepoUrl = "https://github.com/danielraab/boardly"`; verify `make check` passes
- [x] 1.2 Add `frontend/src/components/GithubIcon.tsx` (inline GitHub mark, `currentColor`, `aria-hidden`, `className` for size, like `AppIcon`); verify it renders in the theme's text colour in light and dark
- [x] 1.3 Add i18n keys `common.opensInNewTab`, `landing.github`, `landing.footer.openSource` and `shell.github` to `en.json` and `de.json`; verify the i18n tests pass and both locales have the same keys

## 2. Landing page

- [x] 2.1 In `Landing.tsx`, wrap the hero CTA in a wrapping flex row and add the secondary "GitHub" `<a>` (icon, label, `ArrowUpRight` marker, sr-only "(opens in new tab)", `target="_blank" rel="noopener noreferrer"`); verify in `Landing.test.tsx` that the link has the name "GitHub (opens in new tab)", the repository `href`, `target="_blank"` and `rel="noopener noreferrer"`
- [x] 2.2 Add the landing footer "Open source · GitHub" below the feature list; verify in `Landing.test.tsx` that a `contentinfo` landmark contains "Open source" and a GitHub link with the repository URL
- [x] 2.3 Check the landing page at 320px width; verify the hero buttons wrap and there is no horizontal scrolling

## 3. User menu

- [x] 3.1 In `UserMenu.tsx`, add the signed-in "GitHub" `Menu.LinkItem` rendering an external `<a>`, with dividers above and below, between "Settings" and "Sign out"; verify in `AppShell.test.tsx` the item order (…, "Settings", "GitHub", "Sign out"), the link attributes and the accessible name
- [x] 3.2 Verify in `AppShell.test.tsx` that the signed-out menu still contains only "Sign in" and no GitHub item

## 4. Wrap-up

- [x] 4.1 Run `make check` and `make test` and verify both pass
