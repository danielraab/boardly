# Proposal

## Why

Famodash is meant to be self-hosted, and the README already describes running it as a Docker container behind a reverse proxy, but the repository has no way to build an image. Building the frontend and the Go binary by hand needs Node, pnpm and Go on the host. One `docker build` should produce a small, production-ready image that runs the whole app.

The built-in fallback base URL `http://localhost:3000` (the Vite dev port) is wrong for every deployment. If it applies silently, sign-in links break and every sign-in request fails the Origin check. The server should refuse to start without an explicit base URL.

## What Changes

- Add a multi-stage `Dockerfile` at the repository root: build the frontend with pnpm, build a static Go binary that embeds it, and copy only the binary into a minimal non-root runtime image. The image keeps its data in a `/data` volume, listens on 8080 and has a health check. It can be built for several CPU architectures.
- Add a `.dockerignore`, so local secrets (`.env`), databases, `node_modules` and build output never enter the build context.
- Add a `famodash healthcheck` subcommand. It requests the local `/api/health` and exits 0 if the server is healthy and 1 otherwise. The runtime image has no shell or curl, so the image's `HEALTHCHECK` uses this command.
- **BREAKING**: `FAMODASH_BASE_URL` becomes required. The server exits with an error naming the variable when it is unset or empty. The dev `.env.example` already sets it, so `make dev-backend` keeps working.
- Add a `make docker` target that builds the image.
- Add an app service without a profile to `compose.yaml`, so `docker compose up --build` builds and runs the image locally. `make dev-backend` keeps starting only the dev services.
- Document running with Docker in the README: required variables, a `docker run` example, a compose example with a named volume, and a hint on bind-mount ownership for the non-root user.

Out of scope: running tests in the image build, CI, publishing images to a registry, and version stamping. These come with the build pipeline in a later change.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `app-hosting`: the base URL is required configuration and no longer has a default. Adds the health check command and the container image (non-root, `/data` volume, port 8080, health check).

## Impact

- **New files**: `Dockerfile`, `.dockerignore`.
- **Backend**: `main.go` gets subcommand dispatch for `healthcheck`. `internal/config` drops the base URL default and errors when it is missing. `config_test.go` is updated accordingly.
- **Build**: new `make docker` target, and a `famodash` service in `compose.yaml` that builds the image. `scripts/dev-backend.sh` starts and removes only its dev services by name. The base images are `node` (build), `golang` (build) and `gcr.io/distroless/static` (runtime).
- **Docs**: README configuration table (base URL is required) and a new "Running with Docker" section.
- **Operators**: anyone running the binary without `FAMODASH_BASE_URL` now gets a startup error instead of a silently wrong base URL.
