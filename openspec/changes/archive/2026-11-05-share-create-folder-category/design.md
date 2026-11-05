# Design

## Context

`SharePage` keeps its own form state (`chosenFolder`, `chosenCategory`) on top of the defaults from the profile. Folders are chosen with `FavouritePicker` (via `LinkFields`), categories with `CategoryPicker`. Both create dialogs already exist:

- `NodeDialog` takes `{ kind: "folder", parent, parentLabel }` and calls `onSaved(node)` after a save.
- `CategoryDialog` takes `{ parentId }`, shows a parent field preselected from it, and closes after a save, but doesn't report what it saved.

Both dialogs invalidate their query (`favouritesQueryKey`, `noteCategoriesQueryKey`), so the pickers pick up the new item on their own.

## Goals / Non-Goals

**Goals:**
- Reuse the existing dialogs unchanged in look and validation.
- Keep the share form's state intact across opening and closing a dialog.

**Non-Goals:**
- No inline or name-only quick create, and no "+" on picker rows.
- No changes to the pickers, which are also used in widget settings and the note editor.
- No editing or deleting of folders and categories from the share page.

## Decisions

- **Button below each picker, parent = current selection.** One button per mode, rendered by `SharePage` right after the folder field and after `CategoryPicker`. A subfolder is made by choosing its parent first. This was preferred over per-row "+" buttons (clutter on a phone, changes to shared pickers) and over a "New…" pseudo-option (unclear parent).
- **Folder button placement.** `LinkFields` renders the background hint after the folder picker, so the button can't simply follow `LinkFields`. `LinkFields`' `folder` prop gains an optional `action` node, rendered right after the picker. The favourites dialog passes none.
- **Dialogs outside the share form.** The dialogs render in a portal, but React events bubble through portals along the component tree. A dialog's submit would therefore also reach the share form's `onSubmit` and save the link or note. Both dialogs are rendered after the `<form>`, not inside it.
- **Fixed labels.** The buttons always read "New folder" and "New category", reusing `favourites.dialog.newFolder` and `notes.categories.new` where the wording fits, or new `share.*` keys if a key's namespace would be misleading. The dialog's own title and parent field make the parent clear.
- **Selecting the new item.** `NodeDialog.onSaved(node)` → `setChosenFolder(node.id)`. `CategoryDialog` gets an optional `onSaved?: (category: NoteCategory) => void`, called in the mutation's `onSuccess` with the response before `onClose()`. The share page calls `setChosenCategory({ id: category.id })`. The notes page passes nothing and behaves as before.
- **Parent label for `NodeDialog`.** `parentLabel` is `t("favourites.title")` for the root, otherwise `label(folder)`, the same as the share page's confirmation text. `chooseFolder` stays off: the parent is the current choice.
- **Defaults untouched.** Only the page-local `chosen*` state is set, never the profile, which matches the existing "applies only to that save" rule.

## Risks / Trade-offs

- [The new item isn't in the cached tree for a moment after `onSaved`] → The pickers render from the query, which is invalidated by the dialog. `chosenFolder` / `chosenCategory` hold the ID, and the radio becomes checked once the refetch lands. The save uses the ID directly, so it works even before the refetch.
- [The category dialog is large for a phone share sheet] → Accepted, for consistency. The dialog already scrolls within the viewport.
