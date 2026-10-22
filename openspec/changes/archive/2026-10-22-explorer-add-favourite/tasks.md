## 1. Explorer

- [x] 1.1 Add an optional `onAdd` to `ExplorerTree` that renders an "Add favourite" icon button left of the expand-all button, same size and style
- [x] 1.2 In the explorer `View`, for the owner in view mode with an existing root folder, open `NodeDialog` (new link, `chooseFolder`, root folder preselected) and invalidate the dashboard lookup on save
- [x] 1.3 Add the `addFavourite` translation (en, de)

## 2. Tests and docs

- [x] 2.1 Tests: button shown to the owner and opens the dialog with the root folder; saving posts the link; not shown to visitors, in edit mode, or for a missing folder
- [x] 2.2 Mention the button in README's dashboard widget docs
- [x] 2.3 `make check` and `make test` pass
