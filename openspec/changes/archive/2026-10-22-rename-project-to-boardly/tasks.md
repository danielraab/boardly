# Tasks

## 1. Backend

- [x] 1.1 Rename the Go module to `boardly` and update all imports
- [x] 1.2 Rename the config prefix to `BOARDLY_` and the default DB file to `boardly.db`; update config tests
- [x] 1.3 Rename cookies (`boardly_session`, `boardly_login_request`, `boardly_webauthn`) and migration comments
- [x] 1.4 Change the app token secret prefix to `bdly_`; update tests
- [x] 1.5 Rename the product name in mail templates, golden files and API responses (Karakeep, Linkwarden, WebAuthn); rename the `famomail` alias to `boardlymail`

## 2. Frontend

- [x] 2.1 Rename the npm package, page title, manifests and translations (en + de)
- [x] 2.2 Rename browser storage keys to `boardly.*`
- [x] 2.3 Update tests and the icon render script

## 3. Ops and docs

- [x] 3.1 Update Dockerfile, `compose.yaml`, Makefile, `.env.example`, `scripts/` and CI workflow (image `ghcr.io/danielraab/boardly`)
- [x] 3.2 Update README, including a migration note for existing installs
- [x] 3.3 Update `openspec/config.yaml` and the Purpose sections of live specs

## 4. Verify

- [x] 4.1 No case-insensitive match for `famodash` or `famo_` remains outside `openspec/changes/archive/`
- [x] 4.2 `make check` and `make test` pass
