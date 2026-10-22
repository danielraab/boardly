# Tasks

## 1. Backend

- [x] 1.1 Register the `link` kind with `Normalize`: source, http/https URL ≤ 2048, title ≤ 200 trimmed (empty → null), `imageId` ≤ 64 or null, `icon` from the allowed set or null, target `_blank`/`_self`, size S/M/L. Verify with layout tests for each rule
- [x] 1.2 In `Normalize`, refresh `url`/`favouriteTitle` from an existing favourite of the owner, and convert a missing or foreign favourite to `source: "url"` (title fallback). Verify with tests: refresh, conversion keeping the widget's own title, conversion taking the favourite title, another user's favourite ID
- [x] 1.3 Implement `Resolve` (live `url`/`title` for favourites, copies when missing, nothing extra for visitors). Verify with lookup tests for owner and visitor, a renamed favourite and a deleted favourite
- [x] 1.4 Add a test that an uploaded image referenced by a link widget's `imageId` survives the `image-store` cleanup sweep
- [x] 1.5 Add `FitContent` to `dashboards.Kind`, set it for `link`, and store such widgets with height auto. Verify with a layout test that a link widget saved with height 3 comes back with height auto
- [x] 1.6 Add `iconSource` to the link config (favicon/image/symbol/none, keeping only the matching field, reading old configs), and resolve `data.iconImageId` for website icons from the current URL's origin, queueing a fetch when none is known. Verify with tests for each source, the rejected combinations, an old config, a favourite moved to another site, and a URL link's favicon for a visitor

## 2. Frontend

- [x] 2.1 Add `widgets/sizes.ts` if it doesn't exist yet, and `widgets/link/icons.ts` with the curated icon list. Verify that `pnpm tsc` passes
- [x] 2.2 Build `LinkView` (image → icon → globe, title/URL lines with truncation and tooltip, target, inert in edit mode), and verify with component tests for every scenario of "Link widget rendering"
- [x] 2.3 Build the settings dialog: source radio group (`FavouritePicker` for the favourite, URL with validation), title, image field (pre-fill with debounce, upload, remove, translated errors), icon select, target, size, common width/height. Verify with tests: pre-fill, upload error, URL validation, confirm disabled without a link
- [x] 2.4 Implement "New favourite…": open the favourites' `NodeDialog` (with `chooseFolder` and `onSaved`) on top of the settings, and choose the saved favourite. Verify with tests: the favourite is created in the chosen folder and chosen in the settings, a failed save keeps that dialog open, and the favourite exists after cancelling the dashboard edit
- [x] 2.5 Register the kind (icon `Link`, labels, defaults, view: `data` in view mode, favourites query in edit mode), and verify with an editor test that adds a link widget through the picker
- [x] 2.6 Add the en/de keys (kind, sources, fields, targets, errors), and verify that the i18n check passes
- [x] 2.7 Add a `fitContent` prop to `WidgetSettingsForm` (and a `height` prop to `SizeFields`) that leaves out the height choice and applies height auto, and set it in the link settings. Verify with a test that the link dialog has no height choice and applies height auto
- [x] 2.8 Replace the image and icon fields with the icon chooser (button, popover with tabs, upload, searchable symbol grid, Done), add `useFavicon`, and drop the pre-fill. Verify with tests: the default website icon in the dialog and the draft, choosing a symbol by filter and arrow keys, uploading and an upload error, None, and the saved config

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify that both pass
- [x] 3.2 Verify in the browser: add a link from a favourite, from a new favourite and from a URL; check truncation in a narrow column; delete the favourite and check that the widget still works and is stored as a URL after saving
