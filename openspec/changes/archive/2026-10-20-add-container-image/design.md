# Design

## Context

`make build` runs `pnpm build` in `frontend/`. Vite writes into `../backend/internal/web/dist`, which Go embeds with `//go:embed all:dist`, and then `go build` produces `bin/famodash`. SQLite is `modernc.org/sqlite` (pure Go), so the binary builds with `CGO_ENABLED=0`. At runtime the binary needs outbound TLS for SMTP (`starttls`/`tls`) and for the AAGUID list download, so it needs CA certificates. It doesn't load time zones. `main.go` already shuts down cleanly on SIGINT and SIGTERM. The AAGUID cache lives next to the database (`aaguids.json`). `config.Load` currently falls back to `http://localhost:3000` for the base URL. Requirements are in `specs/app-hosting/spec.md` of this change.

## Goals / Non-Goals

**Goals:**
- One `docker build .` from a clean checkout produces a minimal, non-root runtime image.
- Rebuilds are fast when only source changes: dependency layers and caches are reused.
- Building for `linux/amd64` and `linux/arm64` works without emulating the build steps.

**Non-Goals:**
- Running tests or linters inside the image build.
- CI workflows, registry publishing, image signing, version stamping and OCI labels.
- Changing how the dev services behave (`make dev-backend` still starts only Mailpit and sqlite-web).

## Decisions

### Three stages
```
web   (node:24-slim, $BUILDPLATFORM)    pnpm install --frozen-lockfile → pnpm build
  │      COPY frontend/ to /src/frontend → writes /src/backend/internal/web/dist
  ▼
build (golang:1.26, $BUILDPLATFORM)     go mod download → COPY backend/ + dist
  │      CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w"
  │      also creates an empty /out/data
  ▼
runtime (gcr.io/distroless/static-debian12:nonroot, $TARGETPLATFORM)
         /famodash, /data (owned by 65532), ENV FAMODASH_DB_PATH=/data/famodash.db
         VOLUME /data · EXPOSE 8080 · USER nonroot · HEALTHCHECK · ENTRYPOINT ["/famodash"]
```
- The frontend output doesn't depend on the architecture, and Go cross-compiles natively, so both build stages run on `$BUILDPLATFORM`. A multi-arch build compiles twice but never runs build tools under QEMU.
- The web stage keeps the repository layout (`/src/frontend`) so the `outDir` in `vite.config.ts` works unchanged. No Docker-specific Vite config.
- Layer order: copy the lockfiles first (`package.json`, `pnpm-lock.yaml`, `pnpm-workspace.yaml`; `go.mod`, `go.sum`), install or download dependencies, then copy the sources. BuildKit cache mounts hold the pnpm store, `GOMODCACHE` and `GOCACHE`. The Dockerfile starts with `# syntax=docker/dockerfile:1` so cache mounts and `$BUILDPLATFORM` are available.
- **Why distroless/static over scratch:** it ships CA certificates, `/etc/passwd` with a `nonroot` user (65532) and tzdata, which the binary needs or could use, for about 2 MB. With `scratch` all of that would have to be copied in by hand.
- **Why not alpine:** it would add a shell and a package manager that the app doesn't need. The health check command removes the only reason to have `wget`.

### pnpm version
The web stage installs pnpm with `npm install -g pnpm@${PNPM_VERSION}`. `ARG PNPM_VERSION` defaults to the version used in development (11.x), matching the lockfile format (`lockfileVersion: '9.0'`) and the `allowBuilds` setting in `pnpm-workspace.yaml`. This uses npm instead of corepack, which is deprecated and leaves Node after v24.

### `/data` ownership
A named volume mounted on an empty image directory takes over that directory's owner. The build stage creates `/out/data`, and the runtime stage copies it with `COPY --chown=65532:65532`, so a fresh named volume is writable by `nonroot` without an entrypoint script. Bind mounts keep the host directory's owner. The README tells users to run with `--user "$(id -u):$(id -g)"` (or `user:` in compose), or to `chown 65532:65532` the host directory.

### Health check subcommand
- `main.go` checks `os.Args[1] == "healthcheck"` before `run()`. That's enough for one subcommand and needs no CLI library.
- The command reads only `FAMODASH_PORT`, validated by the same parser as `config.Load`. That parser is exposed as a small `config` function that returns the port. It must not call the full `config.Load`, because an unrelated config problem (e.g. the missing base URL) would then turn into a misleading health result.
- It sends `GET http://127.0.0.1:<port>/api/health` with a 3-second timeout. It exits 0 on 200 and 1 otherwise, and writes a one-line reason to stderr. Docker shows that line in `docker inspect`.
- Dockerfile: `HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/famodash", "healthcheck"]` (exec form, because there's no shell). `FAMODASH_PORT` comes from the container environment, so a changed port is checked correctly.
- Logic and exit code are separate: a function returns an error and `main` maps it to the exit code. Tests call the function against an `httptest` server.

### Required base URL
`config.Load` drops the `http://localhost:3000` fallback. An empty or unset `FAMODASH_BASE_URL` returns `FAMODASH_BASE_URL: required`, and `main` turns that into a non-zero exit (existing path). The image sets no default either, so `docker run` without it fails at once. `FAMODASH_WEBAUTHN_RP_ID` still defaults to the base URL's host.

### `.dockerignore`
It works as an allow-list: `*` first, then re-include `frontend/` and `backend/`. It also excludes `**/node_modules`, `backend/internal/web/dist/*` (the build stage fills it), `**/*.db*` and `backend/data`. With an allow-list, `.env`, `data/`, `bin/`, `.git`, `.claude` and anything added later stay out unless they're listed on purpose.

### `make docker`
`docker build -t famodash:dev .`, with an optional `DOCKER_PLATFORM` variable passed to `--platform`. It stays a thin wrapper, so CI can call `docker buildx` directly later.

### Compose app service
`compose.yaml` gets a `famodash` service with no profile: `build` from the `Dockerfile` (compose names the image itself; a commented-out registry `image:` line is kept for later), port 8080, a named volume at `/data`, text logs, and no SMTP, so sign-in links go to the log. `docker compose up --build` then builds and runs the image without remembering any `docker run` flags.
- **Base URL:** `${FAMODASH_COMPOSE_BASE_URL:-http://localhost:8080}`. Compose reads the dev `.env` for interpolation, and that file's `FAMODASH_BASE_URL` points to the Vite port. Reusing that name would give the container a wrong base URL.
- **Dev script:** compose always includes services without a profile, even with `--profile dev`. So `scripts/dev-backend.sh` names its services (`up -d mailpit sqlite-web`) and removes only those on exit (`rm --stop --force mailpit sqlite-web`) instead of `down`. Otherwise `make dev-backend` would start the app container on 8080, next to the Go dev server, and stop a running app container on exit.

## Risks / Trade-offs

- [Breaking: a missing base URL now stops startup] → Dev is covered by `.env.example`. The README marks it as required, and the error names the variable.
- [Base image tags drift over time] → Pin major tags (`node:24-slim`, `golang:1.26`, `distroless/static-debian12:nonroot`). Pinning digests and bumping them belongs with the pipeline change.
- [Go version in `go.mod` (1.26.x) is newer than the image's patch release] → `golang:1.26` tracks the latest 1.26 patch release. If `go.mod` ever needs a newer toolchain, the build fails clearly and the tag is bumped.
- [Bind mounts owned by the host user aren't writable for uid 65532] → Documented in the README with both fixes. Named volumes are the recommended default.
- [No shell in the runtime image makes debugging harder] → Use `docker cp` and the `:debug` distroless variant when needed. The SQLite file is on the volume and can be inspected from outside.

## Migration Plan

Nothing in the database changes. Operators who run the binary directly must set `FAMODASH_BASE_URL` before upgrading. To roll back, run the previous binary.
