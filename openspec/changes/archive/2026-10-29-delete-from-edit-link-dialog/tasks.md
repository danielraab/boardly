# Tasks

## 1. Delete button in the link edit dialog

- [x] 1.1 Add `favourites.dialog.delete` ("Delete" / "Löschen") to `frontend/src/locales/en.json` and `de.json`, and check that the i18n key-parity test passes
- [x] 1.2 Add an optional `onDeleted` callback to `DeleteDialog`, called on success before `onClose`, and check that the existing favourites tests still pass
- [x] 1.3 In `NodeForm`, show a "Delete" button (`Trash2` icon, left of Cancel and Save) only when editing a link. It opens a nested `DeleteDialog`, and on success it calls the new `NodeDialog` prop `onDeleted` and then `onClose`. Check it in the browser on `/favourites`
- [x] 1.4 Add tests for `NodeDialog`, or in `Favourites.test.tsx`: Delete shows only for an existing link (not for a folder or a new node); confirming calls the delete API and closes both dialogs; cancelling keeps the edit dialog open with unsaved values; a failed delete keeps the confirmation open with the error

## 2. Delete from the details on dashboards

- [x] 2.1 In `FavouriteDetailsDialog`, pass `onDeleted` to `NodeDialog` so that it calls `onSaved?.()` and `onClose()`, and update the `onSaved` doc comment to say it also fires after a delete
- [x] 2.2 Add a test in `FavouriteDetails.test.tsx`: details → Edit → Delete → confirm leaves no dialog open and calls `onSaved` and `onClose`

## 3. Verify

- [x] 3.1 Run `make check` and `make test` and confirm both pass
- [x] 3.2 Delete a link through details → Edit → Delete on a dashboard with a link list and a link widget for it. The list drops the link and the widget keeps working as a URL-only link
