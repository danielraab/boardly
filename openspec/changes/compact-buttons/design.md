## Context

Button classes are spread over three modules: `primaryButton`/`secondaryButton` in `features/auth/LoginPage.tsx`, `dangerButton` in `components/dialogStyles.ts`, `headerButton` in `components/menuStyles.ts`, plus two local `iconButton` constants in `DashboardsPage.tsx` and `FavouritesPage.tsx` and one inline icon button in `PasskeysTab.tsx`. The base classes use `px-6 py-3` and most call sites override the padding (`py-2.5`, `py-2`, `px-4 py-2 text-sm`), so sizes differ from screen to screen.

## Goals / Non-Goals

**Goals:**
- One size and one corner radius per button kind, defined in one place.
- Noticeably more compact buttons without hurting touch usability (icon buttons stay at least 28px).

**Non-Goals:**
- A `<Button>` React component; plain class constants keep matching the existing style.
- Changing chips, badges, spinners or menu items.

## Decisions

### 1. One `buttonStyles.ts` module with class constants

```ts
primaryButton   // bg-ink text-paper
secondaryButton // border border-line text-ink
dangerButton    // bg-danger text-on-danger
iconButton      // size-7 rounded-md, muted → ink on hover
headerButton    // size-9 rounded-lg
```

Text buttons share `inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-medium` and disabled handling. Call sites keep layout-only extras (`mt-4 w-full`, `shrink-0`, `hover:text-danger`) but no padding or font-size overrides. The icon-only cancel button in the layout editor uses `secondaryButton p-2` and stays that way, since it is a bordered icon button.

*Alternative:* keep the constants where they are and only edit them. Rejected because dialogs and pages would still need to import from the login page, and the per-call overrides would keep sizes drifting.

### 2. Corner radius

Text buttons and header buttons use `rounded-lg` (same as inputs). Small icon buttons use `rounded-md` so the corner doesn't look round at 28px.

## Risks / Trade-offs

- [Smaller hit targets on touch screens] → icon buttons stay ≥28px and keep their spacing; text buttons are still 36px high.
