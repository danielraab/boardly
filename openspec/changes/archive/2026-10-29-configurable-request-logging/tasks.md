# Tasks

## 1. Configuration

- [x] 1.1 Add `LogRequests string` to `config.Config` (default `"all"`) and read `BOARDLY_LOG_REQUESTS` in `Load`, rejecting values other than `all`, `errors` and `none` with an error naming the variable; verify with `go test ./internal/config/...`
- [x] 1.2 Add config tests for the default, each valid value and an invalid value (`verbose`); verify they pass

## 2. Middleware

- [x] 2.1 Change `logRequests` to take the mode: return `next` unwrapped for `none`, skip lines with status < 500 for `errors`, and skip `GET /api/health` answered with 200 in every mode; verify with `go build ./...`
- [x] 2.2 Add `LogRequests` to `httpapi.Deps`, pass it to `logRequests` in `New`, and set it from `cfg.LogRequests` in `main.go`; verify with `go vet ./...`
- [x] 2.3 Add middleware tests that capture slog output (a handler on a buffer) for each spec scenario: all modes against 200/404/500 responses, a static file request in `all`, healthy (200) and failing (503) health checks; verify with `go test ./internal/httpapi/...`

## 3. Docs and verification

- [x] 3.1 Add a `BOARDLY_LOG_REQUESTS` row to the README configuration table below `BOARDLY_LOG_FORMAT`, mentioning the values, that `errors` means status 500 and higher, that successful health checks are never logged, and that a reverse proxy can keep access logs; verify the table renders
- [x] 3.2 Add a CHANGELOG entry for the new variable
- [x] 3.3 Run `make check` and `make test` and verify both pass
