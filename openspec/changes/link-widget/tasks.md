# Tasks

## 1. Backend

- [x] 1.1 Register the `link` kind with `Normalize`: source, http/https URL ≤ 2048, title ≤ 200 trimmed (empty → null), `imageId` ≤ 64 or null, `icon` from the allowed set or null, target `_blank`/`_self`, size S/M/L. Verify with layout tests for each rule
- [x] 1.2 In `Normalize`, refresh `url`/`favouriteTitle` from an existing favourite of the owner, and convert a missing or foreign favourite to `source: "url"` (title fallback). Verify with tests: refresh, conversion keeping the widget's own title, conversion taking the favourite title, another user's favourite ID
- [x] 1.3 Implement `Resolve` (live `url`/`title` for favourites, copies when missing, nothing extra for visitors). Verify with lookup tests for owner and visitor, a renamed favourite and a deleted favourite
- [x] 1.4 Add a test that an uploaded image referenced by a link widget's `imageId` survives the `image-store` cleanup sweep

## 2. Frontend

- [x] 2.1 Add `widgets/sizes.ts` if it doesn't exist yet, and `widgets/link/icons.ts` with the curated icon list. Verify that `pnpm tsc` passes
- [x] 2.2 Build `LinkView` (image → icon → globe, title/URL lines with truncation and tooltip, target, inert in edit mode), and verify with component tests for every scenario of "Link widget rendering"
- [x] 2.3 Build the settings dialog: source radio group (favourite picker, new favourite with folder, URL with validation), title, image field (pre-fill with debounce, upload, remove, translated errors), icon select, target, size, common width/height. Verify with tests: pre-fill, upload error, URL validation, confirm disabled without a link
- [x] 2.4 Implement the "New favourite" confirm flow (create first, then add the widget, invalidate favourites, keep the dialog open on error), and verify with a test that the favourite exists after cancelling the dashboard edit
- [x] 2.5 Register the kind (icon `Link`, labels, defaults, view: `data` in view mode, favourites query in edit mode), and verify with an editor test that adds a link widget through the picker
- [x] 2.6 Add the en/de keys (kind, sources, fields, targets, errors), and verify that the i18n check passes

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify that both pass
- [ ] 3.2 Verify in the browser: add a link from a favourite, from a new favourite and from a URL; check truncation in a narrow column; delete the favourite and check that the widget still works and is stored as a URL after saving
