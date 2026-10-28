# Proposal

## Why

Boardly is open source, but nothing in the app says so or points to the code. Visitors who find a Boardly instance can't tell where it comes from. Signed-in users have no quick way to reach the project to report issues or look for updates.

## What Changes

- The landing page gets a secondary "GitHub" button next to the sign-in call to action. It opens the Boardly repository in a new tab.
- The landing page gets a small footer below the feature list: "Open source · GitHub", where "GitHub" links to the repository.
- The signed-in user menu gets a "GitHub" item in its own section between "Settings" and "Sign out", separated by dividers, so "Sign out" stays last.
- The signed-out user menu is unchanged (only "Sign in").
- A new `GithubIcon` component draws the GitHub mark as inline SVG. lucide-react no longer ships brand icons.
- Every GitHub link points to the hardcoded URL `https://github.com/danielraab/boardly`. It is not configurable. On an instance with a custom app title, the link still points to the upstream project.
- All GitHub links are external. They open in a new tab with `rel="noopener noreferrer"`, show a visible external-link marker (↗) and tell screen readers that they open in a new tab.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `landing-page`: the hero gains a GitHub link next to the call to action, and a new footer requirement adds "Open source · GitHub".
- `app-shell`: "Signed-in menu content" gains a "GitHub" external link between "Settings" and "Sign out".

## Impact

- Frontend only, with no backend or API changes.
- `frontend/src/features/landing/Landing.tsx`: hero button and footer.
- `frontend/src/components/UserMenu.tsx`: new menu section.
- New `frontend/src/components/GithubIcon.tsx` and a shared constant for the repository URL.
- `frontend/src/locales/en.json` and `de.json`: new keys.
- Tests: `Landing.test.tsx` and the user menu tests in `AppShell.test.tsx`.
