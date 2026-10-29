# Proposal

## Why

The server logs one line for every HTTP request, and operators can't turn this off or narrow it down. The container health check alone calls `GET /api/health` every 30 seconds, which adds about 2,880 lines a day to an idle instance and buries the lines that matter.

## What Changes

- New environment variable `BOARDLY_LOG_REQUESTS` with three values:
  - `all` (default): every request is logged, as today
  - `errors`: only requests answered with status 500 or higher are logged
  - `none`: no request is logged
- Any other value stops startup with an error that names the variable.
- A successful `GET /api/health` (status 200) is never logged, in any mode. A failing health check (503) follows the rules of the mode, so it is logged in `all` and `errors`.
- Requests for static web app files are treated like any other request.
- Other log output (startup, background jobs, errors logged by handlers) is unchanged.
- The README's configuration table documents the variable.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `app-hosting`: a new requirement "Request logging" defines `BOARDLY_LOG_REQUESTS`, its modes, its default and the health check exception.

## Impact

- `backend/internal/config`: new `LogRequests` setting, read and validated in `Load`.
- `backend/internal/httpapi`: `logRequests` middleware takes the mode and filters by status and path; the router passes the mode in.
- `backend/main.go`: passes the setting to the router.
- `README.md`: new row in the configuration table.
- The only behaviour change with the default is that successful health checks are no longer logged.
