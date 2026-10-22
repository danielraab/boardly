# Tasks

## 1. Icon

- [x] 1.1 Add `frontend/src/components/AppIcon.tsx` rendering the window-badge SVG with `currentColor`, a `useId()` mask id and `aria-hidden`; verify with a render test that it has no accessible name and uses `currentColor`
- [x] 1.2 Add `frontend/public/favicon.svg` with the same paths and a `prefers-color-scheme` style for the outline; verify a test asserts its path data matches `AppIcon`

## 2. Header and favicon

- [x] 2.1 Show `AppIcon` before "Famodash" inside the header link in `AppShell.tsx`; verify `AppShell.test.tsx` still finds the link by name "Famodash" and a new assertion finds the icon inside it
- [x] 2.2 Link `/favicon.svg` with `type="image/svg+xml"` in `frontend/index.html`; verify after `pnpm build` that `backend/internal/web/dist/favicon.svg` exists

## 3. Checks

- [x] 3.1 Run `make check` and `make test` and verify both pass
