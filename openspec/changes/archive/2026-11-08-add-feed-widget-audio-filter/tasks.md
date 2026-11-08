# Tasks

## 1. Backend

- [x] 1.1 Add `AudioOnly` to `feeds.ItemQuery` and filter on it before the limit; verify with a store test that a limit of 2 returns the two newest items with audio from a mixed feed
- [x] 1.2 Add `audioOnly` to the `feed` kind's config and pass it to the query; verify in `feed_test.go` that a widget with it on shows only episodes for the owner and for visitors, and that a config without it shows everything

## 2. Export and import

- [x] 2.1 Verify in `TestImportWidgetFollowsItsFeed` that `audioOnly` is kept when a widget is imported; the format version stays at 2

## 3. Frontend

- [x] 3.1 Add the setting to the feed widget's settings form and the filter to its edit-mode preview, with translations in `en` and `de`; verify in `Feed.test.tsx` the default, that saving sends `audioOnly`, and that the preview shows only items with audio

## 4. Checks

- [x] 4.1 Run `make check` and the tests of the touched packages and files, and verify they pass
