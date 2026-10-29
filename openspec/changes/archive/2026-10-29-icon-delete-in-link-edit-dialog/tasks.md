# Tasks

## 1. Icon-only Delete button

- [x] 1.1 Add `dangerIconButton` to `frontend/src/components/buttonStyles.ts` and use it for the Delete button in `NodeDialog.tsx`, with only the `Trash2` icon and `aria-label`/`title` set to "Delete". Verify that the "delete from the edit dialog" tests still pass
- [x] 1.2 Add a test that the Delete button has the accessible name "Delete" and no visible text, then run `make check` and `make test`
