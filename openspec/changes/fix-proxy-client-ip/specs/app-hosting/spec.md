## ADDED Requirements

### Requirement: Client IP behind reverse proxies
The server SHALL use the connection's remote address as the client IP for rate limits, unless `BOARDLY_TRUST_PROXY_HEADERS` is `true`. In that case it SHALL read `BOARDLY_TRUSTED_PROXY_COUNT`, an integer from 1 to 10 (default `1`) that gives the number of reverse proxies in front of the server. Other values stop startup with an error that names the variable. The server SHALL join all `X-Forwarded-For` header lines into one comma-separated list. It SHALL take the entry at position `count` from the right as the client IP, or the leftmost entry when the list has fewer entries. When there is no `X-Forwarded-For` header, or the chosen entry is not a valid IP address, it SHALL use the connection's remote address. Entries that a client adds itself, to the left of the proxies' entries, SHALL never be used.

#### Scenario: Spoofed entry behind one proxy
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true`, `BOARDLY_TRUSTED_PROXY_COUNT` is unset and a request arrives with `X-Forwarded-For: 1.2.3.4, 203.0.113.7`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Two proxies
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true`, `BOARDLY_TRUSTED_PROXY_COUNT=2` and a request arrives with `X-Forwarded-For: 1.2.3.4, 203.0.113.7, 198.51.100.2`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Several header lines
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true` and a request has two `X-Forwarded-For` lines, `1.2.3.4` and `203.0.113.7`
- **THEN** the client IP is `203.0.113.7`

#### Scenario: Invalid entry
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS=true` and the chosen entry is `unknown`
- **THEN** the client IP is the connection's remote address

#### Scenario: Headers not trusted
- **WHEN** `BOARDLY_TRUST_PROXY_HEADERS` is unset and a request arrives with `X-Forwarded-For: 203.0.113.7`
- **THEN** the client IP is the connection's remote address

#### Scenario: Invalid count
- **WHEN** the server starts with `BOARDLY_TRUSTED_PROXY_COUNT=0`
- **THEN** it exits with a non-zero status and an error message that names `BOARDLY_TRUSTED_PROXY_COUNT`

#### Scenario: Faked addresses don't escape the link limit
- **WHEN** one client behind a single trusted proxy requests sign-in links for different addresses in quick succession, each request with a different made-up leftmost `X-Forwarded-For` entry
- **THEN** the requests count against that client's one per-IP limit, and the first request over it gets 429 `rate_limited`
