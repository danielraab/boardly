## 1. Configuration

- [x] 1.1 In `backend/internal/config/config.go`, add `TrustedProxyCount int` (default 1), parsed from `BOARDLY_TRUSTED_PROXY_COUNT` as an integer 1–10. Invalid values are a startup error naming the variable. Add config tests for the default, a valid value and invalid values (`0`, `11`, `abc`)
- [x] 1.2 Replace `httpapi.Deps.TrustProxyHeaders` with `TrustedProxies int` (0 = don't trust headers). Set it in `backend/main.go` from `TrustProxyHeaders` and `TrustedProxyCount`, and store it on `api` in place of `trustProxy`

## 2. Client IP

- [x] 2.1 Change `ratelimit.ClientIP` to `ClientIP(r, trustedProxies int)`: join all `X-Forwarded-For` values, split on commas and trim. Take the entry `trustedProxies` from the right (the leftmost if there are fewer). Strip an optional port or brackets and validate with `netip.ParseAddr`. Fall back to `RemoteAddr` when there is no header or the entry is invalid
- [x] 2.2 Update the caller in `backend/internal/httpapi/auth.go` (`requestLink`)
- [x] 2.3 Table tests in `backend/internal/ratelimit` for every scenario of the spec: spoofed leftmost entry, two proxies, fewer entries than the count, several header lines, invalid entry, entry with a port, no header, headers not trusted
- [x] 2.4 An `httpapi` test: with one trusted proxy, link requests for different addresses with a varying fake leftmost entry count against one per-IP bucket (429 once the burst of 5 is used up), and the same requests without trusted proxies but from different connection addresses are not limited by IP

## 3. Docs

- [x] 3.1 README: add `BOARDLY_TRUSTED_PROXY_COUNT` to the configuration table. In the reverse-proxy notes, say that the client IP is the address the outermost trusted proxy saw, and that the count must equal the number of proxies
- [x] 3.2 CHANGELOG `[Unreleased]` → `### Security`: the per-IP sign-in limit can no longer be bypassed with a faked `X-Forwarded-For`, and setups with several proxies should set `BOARDLY_TRUSTED_PROXY_COUNT`

## 4. Verify

- [x] 4.1 Run the backend checks and tests (`cd backend && go vet ./... && test -z "$(gofmt -l .)" && go test ./...`)

OpenAPI documents, export and import are unaffected; no tasks needed for them.
