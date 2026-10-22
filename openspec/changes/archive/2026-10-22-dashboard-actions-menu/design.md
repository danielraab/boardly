# Design

## Context

- `DashboardView` (`frontend/src/features/dashboards/DashboardView.tsx`) renders a dashboard in view mode with the owner's "Edit" button. With `?edit=true` it renders `LayoutEditor`, which has its own toolbar in the same `DashboardPage` actions slot.
- It is rendered by the `/$userSlug/$dashboardSlug` route and by `Home` (`DefaultDashboard`) at `/`. `Home` finds the default in the `["dashboards"]` list and then loads the view by its slug.
- `DashboardDialog` takes a full `Dashboard` (`id`, `slug`, `title`, `isDefault`, `isPublic`, columns). On success it invalidates the dashboards list and all view queries, then closes.
- `DashboardViewData` has `id` (owner only), `dashboard: { slug, title }` and `columns`, but neither `isDefault` nor `isPublic`.
- Header menus (`UserMenu`, `DashboardSwitcher`, …) use Base UI `Menu` with `menuPopup` and `menuItem` from `components/menuStyles.ts`.

## Goals / Non-Goals

**Goals:**
- Reuse `DashboardDialog` as it is, without a second settings form.
- Keep the page on a valid address after a save that changes the slug or the default flag.

**Non-Goals:**
- Settings in edit mode. They would mix the column settings with an unsaved layout draft that depends on them.
- Deleting the dashboard from the menu.
- Redirecting old slugs on the server.

## Decisions

### Owner-only fields in the view API
`View` gets `IsDefault *bool` and `IsPublic *bool` with `json:",omitempty"`, set only when `v.IsOwner`, like `ID`. `Lookup` already reads `is_public`, and it also selects `is_default`. Pointers keep `false` in the response for the owner while leaving the fields out for visitors.

*Alternative:* read the flags from the `["dashboards"]` list query on the client. That would avoid the backend change, but it needs a second request on `/<user>/<slug>` and a loading state for "Settings". The view already carries `id` for the owner, so the flags fit there.

On the client, `DashboardViewData` gets `isDefault?: boolean` and `isPublic?: boolean`. `DashboardView` builds the dialog's `Dashboard` from `id`, `dashboard`, the flags and `columns`.

### The actions menu
A `DashboardActions` component in `DashboardView.tsx` takes the `actions` slot in view mode: a `Menu.Trigger` styled with the new `secondaryIconButton` (square, as tall as the text buttons; the edit toolbar's ✕ uses it too) (`Ellipsis` icon, `aria-label` and `title` "Dashboard actions"), and a popup with two `Menu.Item`s, "Edit layout" (`LayoutGrid` icon) and "Settings" (`Settings` icon), with `align="end"`. "Settings" sets local state that opens `DashboardDialog`. The dialog is rendered next to the menu, outside it, so it stays open when the menu closes.

### Following the dashboard after save
`DashboardDialog` gets an optional `onSaved?: (saved: Dashboard) => void`. `DashboardForm` calls it in `onSuccess` with the server's response before it invalidates the queries and closes. `/dashboards` doesn't pass it and keeps its behaviour.

`DashboardView` gets an optional `atHome?: boolean` prop, which `Home`'s `DefaultDashboard` sets. The `onSaved` handler in `DashboardView` then decides:

```
atHome && !saved.isDefault        → navigate /$userSlug/$dashboardSlug (saved.slug), replace
!atHome && saved.slug !== slug    → navigate /$userSlug/$dashboardSlug (saved.slug), replace
otherwise                         → nothing; the invalidated view refetches
```

`replace` is used because the old address no longer works and Back shouldn't lead to a not-found page.

*Why navigate before invalidating:* the invalidation refetches the old slug's view, which now 404s, and on `/` it refetches the list, which no longer has a default. Navigating first means the page never renders the not-found page or the "Select your default dashboard" prompt in between.

*Alternative:* derive "at home" from the router location. A prop is more explicit and keeps `DashboardView` free of path checks.

### Translations
- `dashboards.editor.edit` is renamed to `dashboards.editor.editLayout`: "Edit layout" / "Layout bearbeiten".
- New `dashboards.view.actions` ("Dashboard actions" / "Dashboard-Aktionen") and `dashboards.view.settings` ("Settings" / "Einstellungen").
- `dashboards.view.emptyOwnerText` now points to "Edit layout": "Open the ⋯ menu and choose “Edit layout” to add the first section." / "Öffne das ⋯-Menü und wähle „Layout bearbeiten“, um den ersten Abschnitt hinzuzufügen."

## Risks / Trade-offs

- [The layout editor needs one more click than before] → Accepted. The user chose a quieter header, and edit mode itself is unchanged.
- [`onSaved` runs before the invalidation, so a navigation to the new slug can briefly hit a cached view of the old one] → The new slug has a different query key, so it loads fresh. The old key is invalidated right after.
- [Tests that click "Edit" break] → Update them to open the menu and pick "Edit layout", in the same change.
