# Design

## Context

See proposal.md for the motivation. `NoteEditorPage` renders the action row with `SaveIndicator`, the title and then `NoteEditor`, which renders the sticky toolbar. `NoteEditorPage` is also the content of the dashboard note dialog, where sticky positioning is relative to the scrolling dialog. `SaveIndicator` is also used by `NoteCard` for inline editing, which has no toolbar.

The toolbar wrapper uses `bg-inherit`, but its parent `div.mt-6` has no background, so it computes to transparent. The note's colour is set on the `section` around them.

`frontend/index.html` has a viewport meta tag without `interactive-widget`, so the on-screen keyboard only shrinks the visual viewport and the browser can pan content at the top out of view.

## Goals / Non-Goals

**Goals:**

- One save status region, always inside the sticky row, at every width.
- Keep autosave behaviour and `useAutosave` unchanged.

**Non-Goals:**

- A toolbar docked above the keyboard.
- Changing the status on cards.

## Decisions

**`NoteEditor` gets a `toolbarEnd` slot.** `NoteEditorPage` passes `<SaveIndicator>` as a `ReactNode`, which `NoteEditor` renders next to `Toolbar` inside the sticky wrapper. This keeps autosave state out of the editor and leaves the compact editor on cards untouched. Alternatives: making the whole action row sticky costs about three rows on a phone, and a floating pill collides with the mini player and cookie notice and needs keyboard tracking.

**`SaveIndicator` gets `compact` and `onReload` props.** With `compact`, "saving" and "saved" render an icon (`LoaderCircle`, `motion-safe:animate-spin`, and `Check`) below the `sm` breakpoint (640 px), and the text is `sr-only sm:not-sr-only`. The text stays in the DOM, inside the single `role="status"` element, so screen readers still hear it. With `onReload`, status `conflict` renders the label and a "Reload" button; without it, conflict renders nothing, as on cards today. Two responsive copies of the indicator would duplicate the live region, so there is one.

**Saved uses a new `success` token.** The theming spec requires colours to come from tokens, and no green existed. Light themes use `#166534` and dark themes `#4ade80`, which meet 4.5:1 on `paper`, `surface` and every note colour, because the status sits on the note's background. "Saving…" stays `muted`, so green only ever means that the save is done.

**The toolbar scrolls inside a flex row.** The sticky wrapper becomes `flex items-center gap-2`. The `Toolbar` root is `min-w-0 flex-1 flex-nowrap overflow-x-auto` with a hidden scrollbar, and the separator gets `shrink-0`. The slot is at most 65% of the row and its text may wrap, because "Nicht gespeichert" with "Erneut versuchen" on one line would leave a 360 px phone only 13 px of buttons. A partly cut-off button shows that the row scrolls; no extra fade is added. This applies at every width, because the buttons fit on one row from `sm` on anyway.

**The background is chained with `bg-inherit` on `div.mt-6`.** The note colours are opaque, and the inline `background` on the `section` sets its computed `background-color`, so inheriting through one more element is enough and avoids passing the colour as a prop.

**The viewport meta tag gets `interactive-widget=resizes-content`.** With it, Android browsers shrink the layout viewport for the keyboard, so `top-0` sticky content stays in view. iOS Safari ignores the key. iOS is tested by hand; a `visualViewport` offset is only added, through an update of this change, if the test shows the bar leaving the view.

## Risks / Trade-offs

- [Applies to the whole app] → On Android, fixed elements like the mini player and the `dvh` dialogs are laid out above the keyboard. This is the desired behaviour for dialogs; check the mini player during the device test.
- [iOS Safari may still pan the bar out of view while typing near the bottom of the page] → Device test in the tasks; the status is still correct and visible once the keyboard closes or the page scrolls.
- [A long failed or conflict status takes up room on narrow phones] → It wraps within 65% of the row, which makes the row taller only in that state; the buttons keep the rest and scroll.
- [Two "Reload" buttons during a conflict, in the banner and in the status] → Both do the same thing; tests that look up "Reload" have to allow for two.
