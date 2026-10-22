## Why

Buttons take more room than they need: text buttons use `px-6 py-2.5`/`py-3` at base font size, icon buttons are 32–40px, and all of them are fully rounded pills. On dense screens (dashboard editor, favourites tree, dialogs) this wastes space and the pill shape looks heavy next to the `rounded-lg` inputs and cards.

## What Changes

- Text buttons (primary, secondary, danger) become smaller: `px-4 py-2 text-sm`, with `rounded-lg` corners instead of `rounded-full`.
- Icon buttons shrink one step (header 40→36px, row/action buttons 32→28px, passkey delete 36→32px) and use `rounded-md`/`rounded-lg` corners.
- The button class constants move into one shared module, `frontend/src/components/buttonStyles.ts`, instead of being exported from `LoginPage.tsx`, `dialogStyles.ts` and `menuStyles.ts`. Call sites drop their per-call padding overrides (`py-2.5`, `px-4 py-2 text-sm`, …) so every button gets the same size.
- The landing page CTA uses the shared primary button style.
- The avatar is no longer described as a round button; it gets the same corners as the other header buttons.

Chips and badges (tag chips, the "default" badge, counts) are not buttons and keep their pill shape.

## Capabilities

### New Capabilities

None.

### Modified Capabilities
- `theming`: a requirement for a consistent, compact button shape across the app.
- `app-shell`: the avatar is an icon button with rounded corners instead of a round button.

## Impact

- Frontend only; no backend or API changes.
- New `frontend/src/components/buttonStyles.ts`; `LoginPage.tsx`, `dialogStyles.ts`, `menuStyles.ts` no longer export button classes.
- Call sites in `features/auth`, `features/dashboards`, `features/favourites`, `features/settings`, `features/home`, `features/landing` and the header switchers.
