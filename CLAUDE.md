# CLAUDE.md

See [README.md](README.md) for what the project is and how to run it.

## Workflow: always use OpenSpec

Every code change goes through [OpenSpec](openspec/). Do not edit code under `backend/` or `frontend/` without an OpenSpec change.

1. **Explore** (optional): `/opsx:explore` to think through an idea or clarify requirements.
2. **Propose**: `/opsx:propose` to create a change in `openspec/changes/<name>/` with proposal, design, specs and tasks.
3. **Apply**: `/opsx:apply` to implement the change's tasks and tick them off as you go.
4. **Archive**: `/opsx:archive` when the change is done, which syncs its delta specs into `openspec/specs/`.

If a request asks for a code change and no matching OpenSpec change exists, propose one first.
If the work turns out to differ from the change, update the change's artifacts before changing the code.

Exceptions that don't need a change: typo fixes, formatting or lint-only fixes, dependency bumps, and documentation-only edits.

## Test only what changed

Run the tests of the side whose code changed, and only that side:

- Code under `frontend/` changed: run the frontend checks and tests (`cd frontend && pnpm check && pnpm exec tsc --noEmit && pnpm test`).
- Code under `backend/` changed: run the backend checks and tests (`cd backend && go vet ./... && test -z "$(gofmt -l .)" && go test ./...`).
- Both changed: run both (`make check` and `make test`).
- Neither changed (for example documentation, `CHANGELOG.md`, OpenSpec artifacts, or resolving a conflict in such files during a rebase): run no tests.

Check with `git diff --name-only` (against the base of the step) which paths a step changed before testing.

## Keep export and import in step

Every change must check whether the data export (`data-export` spec, `backend/internal/export`) and the import of Boardly exports (`data-import` spec) have to be adapted. This applies whenever a change adds, removes, renames or reshapes user data: tables, columns, fields, widget kinds or their configs, files, or references between items. The proposal states the result, even when it is "export and import are unaffected", and the tasks include the adaptation and its tests. Raise the export format version when older imports couldn't read the new document.

## Keep the OpenAPI documents in step

The API is described in `backend/internal/httpapi/openapi/`: `boardly.yaml` for `/api`, and `karakeep.yaml`, `linkwarden.yaml` and `memos.yaml` for the connected apps (`api-docs` spec). Every change that adds, removes or renames a route, or changes a request or response shape, updates the matching document in the same change, and its tasks say so. `TestOpenAPIDocumentsMatchRoutes` fails for missing or stale routes, but it doesn't check schemas, so keep those in step by hand.
