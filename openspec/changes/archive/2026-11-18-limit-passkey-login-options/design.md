## Context

`passkeyLoginOptions` calls `auth.PasskeyLoginOptions`, which inserts a `webauthn_challenges` row with a 5-minute expiry. The janitor deletes expired rows. Sign-in link requests already have an in-memory per-IP `ratelimit.Limiter` (20/hour, burst 5). On the login page, passkey autofill requests options when the page opens. It requests them again after an `ApiError`, so that autofill keeps working after a failed passkey.

## Goals / Non-Goals

**Goals:**
- Bound the challenge rows one client can create.
- No visible change for normal use, including shared networks such as an office behind one IP.

**Non-Goals:**
- Stateless challenges (keeping the challenge in a signed cookie instead of the database). That would remove the writes entirely, but it changes the single-use guarantee of the ceremony. It is a larger change than this fix needs.
- A global cap on live challenges. Hitting it would lock everyone out of passkey sign-in, which trades a storage problem for an availability one.
- Grouping IPv6 clients by /64. This would apply to all per-IP limits and belongs in its own change.

## Decisions

- **Limit 120/hour, burst 30, at most 10,000 tracked IPs.** This uses the same `ratelimit.New` as the other limiters. The burst covers many quick retries. The hourly rate covers a busy shared IP. Per client, this caps live rows at about 30 + 10 in any 5-minute window.
- **Check before creating the challenge.** A rejected request writes nothing and sets no challenge cookie.
- **Separate limiter from the link limiter**, so passkey attempts don't use up the email budget and the other way round.
- **Frontend: no autofill restart on 429.** `startAutofill` restarts only for API errors other than `rate_limited`. The button path maps `rate_limited` to the existing `rateLimited` problem with the `Retry-After` seconds, as the email form does.

## Risks / Trade-offs

- [An attacker with many IPs still creates rows] → The cost grows with the number of addresses, and rows expire after 5 minutes. A stateless design remains the follow-up if needed.
- [The limit is in memory and resets on restart] → The same as the existing limiters. That is acceptable for an abuse limit.
