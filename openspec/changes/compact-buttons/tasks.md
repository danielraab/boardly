# Tasks

## 1. Shared button styles

- [x] 1.1 Create `frontend/src/components/buttonStyles.ts` with `primaryButton`, `secondaryButton`, `dangerButton` (`rounded-lg px-4 py-2 text-sm`), `iconButton` (`size-7 rounded-md`) and `headerButton` (`size-9 rounded-lg`)
- [x] 1.2 Remove the button constants from `LoginPage.tsx`, `dialogStyles.ts` and `menuStyles.ts`, and the local `iconButton` constants from `DashboardsPage.tsx` and `FavouritesPage.tsx`; import from `buttonStyles.ts` everywhere

## 2. Call sites

- [x] 2.1 Drop per-call padding and font-size overrides (`py-2.5`, `py-2`, `px-4 py-2 text-sm`, …) on text buttons
- [x] 2.2 Use `iconButton` (with `hover:text-danger`) for the passkey delete button and the shared primary style for the landing CTA
- [x] 2.3 Verify `grep -rn "rounded-full" frontend/src` only hits chips, badges, spinners and swatches

## 3. Check

- [x] 3.1 `make check` and `make test` pass
