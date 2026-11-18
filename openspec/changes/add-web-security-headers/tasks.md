## 1. Configuration

- [ ] 1.1 In `backend/internal/config/config.go`, add `CSPScriptSrc []string` from `BOARDLY_CSP_SCRIPT_SRC` (split on spaces). Reject values containing `;` or `,` with an error naming the variable. Add config tests
- [ ] 1.2 Add it to `web.Options` and pass it from `backend/main.go`

## 2. Web app headers

- [ ] 2.1 In `backend/internal/web`, after `titledFiles` has built the final `index.html`, parse it with `golang.org/x/net/html`. Collect the `'sha256-<base64>'` of the text of every `<script>` without `src`, and the origin of every absolute `<script src>`. Build the policy string once (see design.md)
- [ ] 2.2 Set `Content-Security-Policy`, `X-Frame-Options: SAMEORIGIN`, `X-Content-Type-Options: nosniff` and `Referrer-Policy: strict-origin-when-cross-origin` on every response of `NewHandler`, including the fallback and the "frontend not built" error
- [ ] 2.3 Tests in `backend/internal/web`: headers on `/`, a deep link, an asset and a manifest; the theme script's hash matches the inline script; a snippet with an external and an inline script adds its origin and hash; `CSPScriptSrc` entries appear; a document without inline scripts has no stray hash

## 3. API headers

- [ ] 3.1 In `backend/internal/httpapi`, set `X-Content-Type-Options: nosniff` and, when no `Content-Security-Policy` is set yet, `default-src 'none'; frame-ancestors 'none'` in `WriteJSON` and `WriteError`, so `muxErrorWriter` and the connected-app APIs get them too
- [ ] 3.2 Tests: a JSON success and a JSON error carry both headers; `GET /api/attachments/{id}`, `/api/images/{id}` and the offline article keep their own policies

## 4. Check the real app

- [ ] 4.1 Build the production bundle (`make build`) and run it. With the browser console open, check a note with a map, radio playback, a Spotify embed, an offline copy, theme and language switching, passkey sign-in and an analytics snippet. Fix the policy (and update design.md and the spec) if anything reports a violation

## 5. Docs and verify

- [ ] 5.1 README: add `BOARDLY_CSP_SCRIPT_SRC` to the configuration table. Note under the analytics snippet that its scripts are allowed automatically and that tools loading more scripts need this setting. Recommend HSTS at the reverse proxy
- [ ] 5.2 CHANGELOG `[Unreleased]` → `### Security`: Content-Security-Policy and framing protection for the web app, `nosniff` on API responses
- [ ] 5.3 Run the backend checks and tests (`cd backend && go vet ./... && test -z "$(gofmt -l .)" && go test ./...`). If 4.1 needed frontend changes, run the frontend checks and tests too

OpenAPI documents, export and import are unaffected; no tasks needed for them.
