# Tasks

## 1. Configuration

- [x] 1.1 Add `AnalyticsSnippet` to `config.Config`, read from `BOARDLY_ANALYTICS_SNIPPET` (trimmed, empty when unset or blank), and verify with new cases in `config_test.go` (default empty, value kept as written, blank)
- [x] 1.2 Document `BOARDLY_ANALYTICS_SNIPPET` in the README configuration table (raw HTML before `</head>`, runs on every page with full access, Vite dev server doesn't insert it) and as a commented-out entry in `.env.example`, and verify both are present

## 2. Serving the HTML document

- [x] 2.1 Make `web.Handler`/`NewHandler` take the snippet and insert it unescaped before the first `</head>` of `index.html` (direct and fallback), and verify with `web_test.go` cases for no snippet (document unchanged), `GET /`, a deep link, a snippet with `&&` and `<` kept as written, and a manifest/asset without the snippet
- [x] 2.2 Pass `cfg.AnalyticsSnippet` to `web.Handler` in `main.go`, and verify `go build ./...` and `main_test.go` pass

## 3. Verification

- [x] 3.1 Run `make check` and `make test` and verify both pass
