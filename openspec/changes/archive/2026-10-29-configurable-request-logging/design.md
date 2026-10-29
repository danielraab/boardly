# Design

## Context

`httpapi.New` wraps the root mux in `logRequests` (`internal/httpapi/middleware.go`). That middleware records the status with a `statusRecorder` and calls `slog.InfoContext` for every request. Configuration is read in `config.Load`, where enum-like settings such as `BOARDLY_LOG_FORMAT` and `BOARDLY_SMTP_TLS` are validated against a fixed list, and is passed to the router through `httpapi.Deps`.

## Goals / Non-Goals

**Goals:**
- Filter request lines by mode and status without changing their content or level.
- Keep the default output identical to today's, apart from successful health checks.

**Non-Goals:**
- A general log level setting (`BOARDLY_LOG_LEVEL`). It is a separate concern and can be added later without touching this one.
- Excluding static files or other paths besides the health check.
- Changing where logs go (they stay on stderr) or their format.

## Decisions

**A string-typed mode in config, validated in `Load`.** `Config.LogRequests` holds `"all"`, `"errors"` or `"none"`, defaulting to `"all"`. Validation follows `BOARDLY_LOG_FORMAT` (exact match, error message names the variable and lists the values). An empty value means the default, like the other optional variables. Alternative: a bool `BOARDLY_LOG_REQUESTS`, which was rejected because it can't keep 5xx lines while hiding the rest.

**The mode travels through `httpapi.Deps`.** A new `LogRequests string` field, set from `cfg.LogRequests` in `main.go`, and `logRequests(mode, next)` decides after `next.ServeHTTP` returns whether to log. With `none`, the middleware returns `next` unwrapped, so no recorder is allocated. Alternative: deciding in `main.go` whether to wrap at all. That would split the filter logic in two places, and `errors` needs the recorded status anyway.

**The health exception is matched on method, exact path and status.** A request is skipped when `r.Method == GET`, `r.URL.Path == "/api/health"` and the recorded status is 200. Matching the status, not just the path, keeps a failing check (503) visible, which is the one health line an operator needs. The path is matched exactly so that it can't hide other routes.

**The threshold for `errors` is `status >= 500`.** 4xx responses (expired sessions, scanners hitting unknown paths) are client errors that the server answered correctly, and would make `errors` almost as noisy as `all`.

## Risks / Trade-offs

- [A handler that writes a body without calling `WriteHeader` is recorded as 200] → This matches net/http's own behaviour, and the recorder already defaults to 200 today.
- [With `errors` or `none`, operators lose access logs they may rely on] → `all` stays the default. The README tells operators that a reverse proxy can keep access logs if they turn these off.
- [A health check that hangs and the client gives up still ends with some status] → It is logged according to that status, which is acceptable: the container health check reports the failure itself.
