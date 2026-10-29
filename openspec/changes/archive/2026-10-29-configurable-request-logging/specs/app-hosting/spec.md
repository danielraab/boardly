# Spec Delta

## ADDED Requirements

### Requirement: Request logging
The server SHALL write one log line per HTTP request that it has finished answering, with the method, path, status and duration, according to `BOARDLY_LOG_REQUESTS`:
- `all` (the default when the variable is unset or empty): every request is logged
- `errors`: only requests answered with status 500 or higher are logged
- `none`: no request is logged

Any other value SHALL stop startup with an error message that names `BOARDLY_LOG_REQUESTS`. In every mode, a `GET /api/health` request answered with status 200 SHALL NOT be logged; a health request with any other status follows the rules of the mode. Requests for the web app's files SHALL be treated like any other request. The setting SHALL NOT affect any other log output.

#### Scenario: All requests by default
- **WHEN** the server starts without `BOARDLY_LOG_REQUESTS` and a client requests `GET /api/dashboards`, `GET /` and a hashed JavaScript bundle
- **THEN** the log contains one request line for each of them

#### Scenario: Errors only
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=errors` and answers one request with status 200, one with 404 and one with 500
- **THEN** the log contains a request line only for the request answered with 500

#### Scenario: No request logging
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=none` and answers one request with status 200 and one with 500
- **THEN** the log contains no request line

#### Scenario: Successful health check is not logged
- **WHEN** the server runs with `BOARDLY_LOG_REQUESTS=all` and a client requests `GET /api/health` while the database is reachable
- **THEN** the log contains no request line for it

#### Scenario: Failing health check is logged
- **WHEN** the server runs with `BOARDLY_LOG_REQUESTS=errors` and a client requests `GET /api/health` while the database is not reachable
- **THEN** the log contains a request line with status 503

#### Scenario: Invalid value
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=verbose`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_LOG_REQUESTS`

#### Scenario: Other log output is unchanged
- **WHEN** the server starts with `BOARDLY_LOG_REQUESTS=none`
- **THEN** startup messages and errors logged while handling requests still appear in the log
