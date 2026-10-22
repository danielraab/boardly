# Proposal

## Why

In the "Connected apps" tab, the app a token belongs to is only shown as text in the second line ("Karakeep · Favourites / Reading"). With several tokens for Karakeep and Linkwarden, it's hard to tell at a glance which row belongs to which app.

## What Changes

- Each token in the connected apps list shows the official logo of its app in front of the token's details: the Karakeep logo for Karakeep tokens and the Linkwarden logo for Linkwarden tokens.
- The logos are shown in their original colors. Where the app ships a light and a dark variant (Karakeep), the variant follows the active theme mode, like passkey provider icons.
- The logos are bundled with the web app. Nothing is loaded from the apps' websites at runtime.
- The logo is decorative. The app name stays in the text line, so screen readers and the text layout don't change.
- The app picker in the create dialog stays a plain select without logos.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `app-tokens`: the "Connected apps tab" requirement adds the app logo for each token.

## Impact

- Frontend: new logo assets (Karakeep light and dark SVG, Linkwarden PNG) and a small `AppLogo` component under `frontend/src/features/settings/appTokens/`, `ConnectedAppsTab.tsx` (`AppTokenItem`), and `ConnectedApps.test.tsx`.
- Backend: none.
- No new dependencies and no new translation keys.
