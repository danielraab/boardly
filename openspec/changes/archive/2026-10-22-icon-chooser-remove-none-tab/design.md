# Design

## Context

`IconChooser.tsx` renders the chooser in a Base UI `Popover` (`w-[22rem] max-w-[calc(100vw-2rem)]`, `p-3`) with a Base UI `Tabs` list of `flex flex-wrap`. Each kind's panel has a "use" button built by `chooseButton`, which stays mounted and becomes pressed ("✓ In use") once its kind is chosen, so focus doesn't fall out of the popup. The footer holds only "Done". See proposal.md for why the "None" tab goes.

## Goals / Non-Goals

**Goals:**
- Three tabs that stay on one line down to 320px wide screens.
- "Remove icon" reachable from every tab, without layout shift or lost focus.

**Non-Goals:**
- No change to the stored icon kinds, the API or the trigger button (it still says "Icon: None" / "Icon: Keines").

## Decisions

- **Footer layout.** The footer becomes `flex justify-between`: "Remove icon" (secondary button) on the left, "Done" (primary) on the right. This keeps the action visible on every tab, unlike a link inside one panel.
- **"Remove icon" when the kind already is "None".** It stays mounted and shows the same pressed state as the panels' use buttons: `aria-pressed`, a check icon and the label "No icon" / "Kein Icon"; clicking it then does nothing. Hiding or disabling it would drop focus to the body. Alternative considered: hide it, rejected for the focus reason.
- **Initial tab.** `setTabValue(value.source === "none" ? "favicon" : value.source)` on open; the `IconSource` type keeps `"none"`, the tab list uses a separate `tabs` array without it.
- **No wrapping.** `Tabs.List` uses `flex flex-nowrap`, tabs get `whitespace-nowrap` and `px-2` instead of `px-2.5`. Three German labels need about 270px, the popup has about 264px inside on a 320px screen, so the tabs also get `min-w-0 truncate` as a last resort; from 360px up everything fits untruncated. Alternative considered: `text-xs` on small screens, rejected as inconsistent with the other tabs in the app.
- **Strings.** Remove `iconChooser.none.text` and `iconChooser.none.use`; add `iconChooser.remove` ("Remove icon" / "Icon entfernen") and `iconChooser.removed` ("No icon" / "Kein Icon"). Keep `iconChooser.tabs.none`, because the trigger's kind name still uses it.

## Risks / Trade-offs

- [The "None" panel's hint "The widget shows a globe" disappears] → The trigger button shows the globe as soon as the icon is removed, which tells the same thing.
- [Labels may truncate on 320px screens] → Only on the narrowest phones, and the tab still has its full accessible name.
