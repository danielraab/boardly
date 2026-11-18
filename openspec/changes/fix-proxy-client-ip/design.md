## Context

`ratelimit.ClientIP(r, trustProxy)` is used for the per-IP limit on sign-in link requests (`auth-magic-link`, "Rate limits on link requests"). With trust on, it returns the first entry of the first `X-Forwarded-For` header. Every proxy in the chain appends the address it received the connection from. So only entries added by proxies the operator controls can be trusted, and those are the rightmost ones.

## Goals / Non-Goals

**Goals:**
- A client can't choose the IP that its requests are rate-limited under.
- Setups with more than one proxy in front of Boardly (for example a CDN and nginx) can still limit per real client.

**Non-Goals:**
- Support for the `Forwarded` (RFC 7239) header or `X-Real-IP`.
- Trusting proxies by address range (CIDR lists). A hop count is enough for a self-hosted app and is simpler to configure.
- Any change to which limits use the client IP.

## Decisions

- **Count from the right.** With `n` = `BOARDLY_TRUSTED_PROXY_COUNT` and the combined entries `e[0..len-1]`, the client IP is `e[len-n]`. With one proxy that is the last entry, the address nginx saw.
  *Alternative:* a CIDR list of trusted proxies, walking from the right and skipping trusted addresses. It is more exact, but harder to configure and to explain, and container networks change their addresses.
- **Fewer entries than `n`**: take `e[0]`. Every entry present was added by one of the trusted proxies (the client's own entries would make the list longer), so it is still trustworthy.
- **Combine header lines.** `r.Header.Values("X-Forwarded-For")` is joined with commas before splitting, because some proxies add a second header line instead of appending.
- **Invalid entry**: when the selected entry doesn't parse with `netip.ParseAddr` (after trimming spaces and an optional port or brackets), use `RemoteAddr`. A broken header never becomes a shared bucket key.
- **New variable instead of changing `BOARDLY_TRUST_PROXY_HEADERS` to a number.** The boolean stays valid, so existing deployments keep their configuration.
- **Signature:** `ClientIP(r *http.Request, trustedProxies int)`, where 0 means "don't trust headers". `httpapi.Deps` gets `TrustedProxies int` in place of `TrustProxyHeaders bool`, set from both settings in `main.go`.

## Risks / Trade-offs

- [Operators with two or more proxies get rate limits per outer proxy after the upgrade] → The CHANGELOG and the README explain `BOARDLY_TRUSTED_PROXY_COUNT`. Until it is set, limits are stricter, not weaker.
- [Count set too high] → The leftmost entries become trusted again, and clients could fake them. The README warns to set exactly the number of proxies.
