# Tasks

## 1. Required base URL

- [x] 1.1 Remove the `http://localhost:3000` fallback in `config.Load` and return `FAMODASH_BASE_URL: required` when it is unset or empty. Update `config_test.go`: the defaults test sets only the base URL, and a new case checks that a missing or empty base URL gives an error naming `FAMODASH_BASE_URL`. Verify with `go test ./internal/config/`
- [x] 1.2 Update the README configuration table (`FAMODASH_BASE_URL` is required, no default). Verify that `go run .` in `backend/` without the variable exits non-zero with that message, and that `make dev-backend` still starts with the default `.env`

## 2. Health check command

- [x] 2.1 Expose the port parsing from `internal/config` as a function that reads only `FAMODASH_PORT` (default 8080, same validation and error as `Load`). Verify table tests for default, valid and invalid values
- [x] 2.2 Implement the health check (GET `http://127.0.0.1:<port>/api/health`, 3 s timeout, nil only on 200) and dispatch `famodash healthcheck` in `main.go` before `run()`, with exit 0/1 and a one-line stderr reason. Verify tests against `httptest` servers for 200 → ok, 503 → error, an unreachable port → error and a server that never answers → timeout error, plus a test that the base URL isn't needed
- [x] 2.3 Manually verify: with the dev server running, `./bin/famodash healthcheck; echo $?` prints 0. After stopping it, the command prints a non-zero code

## 3. Container image

- [x] 3.1 Add `.dockerignore` as an allow-list (re-include `frontend/` and `backend/`, exclude `node_modules`, `dist` contents, `*.db*`, `backend/data`). Verify with a throwaway build that lists the context (a `busybox` image that runs `COPY . /c` and `find /c -maxdepth 2`) that `.env`, `data/`, `bin/` and `.git` are missing
- [x] 3.2 Add the multi-stage `Dockerfile` (web on `$BUILDPLATFORM` with pnpm `ARG PNPM_VERSION` and a store cache mount; Go build on `$BUILDPLATFORM` cross-compiling to `$TARGETOS/$TARGETARCH` with `CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"` and module/build cache mounts; runtime `distroless/static-debian12:nonroot` with `/data` owned by 65532, `ENV FAMODASH_DB_PATH=/data/famodash.db`, `VOLUME /data`, `EXPOSE 8080`, exec-form `HEALTHCHECK` and `ENTRYPOINT`). Verify that `docker build -t famodash:dev .` succeeds from a fresh `git clone` of the working tree
- [x] 3.3 Add the `make docker` target (`docker build -t famodash:dev .`, optional `DOCKER_PLATFORM`). Verify that `make docker` builds, and that `make docker DOCKER_PLATFORM=linux/arm64` builds an arm64 image (`docker image inspect -f '{{.Architecture}}'`)
- [x] 3.5 Add a `famodash` service without a profile to `compose.yaml` (build from the `Dockerfile`, port 8080, named volume at `/data`, base URL from `FAMODASH_COMPOSE_BASE_URL` with default `http://localhost:8080`), and make `scripts/dev-backend.sh` start and remove only `mailpit` and `sqlite-web` by name. Verify that `docker compose up -d --build` builds the image and the service turns healthy, and that `docker compose --profile dev config --services` still lists all three while the dev script starts only the two dev services
- [x] 3.4 Verify the runtime behavior from the spec with `make docker`: (a) `docker run` without `FAMODASH_BASE_URL` exits non-zero and logs the variable; (b) with `FAMODASH_BASE_URL=http://localhost:8080` and a named volume, `/` returns the web app, `/api/health` returns `{"status":"ok"}` and `docker ps` shows `healthy`; (c) `docker top` shows the process running as uid 65532; (d) after `docker rm -f` and a new container on the same volume, a previously signed-in account still exists (sign-in link from the log)

## 4. Documentation

- [x] 4.1 Add a "Running with Docker" section to the README: build (`make docker` / `docker build`), required `FAMODASH_BASE_URL`, a `docker run` example with a named volume, a compose example with a named volume and the SMTP variables, the bind-mount ownership hint (`--user "$(id -u):$(id -g)"` or `chown 65532:65532`), and the health status. Update the "Behind a reverse proxy" paragraph to point to it. Verify that the documented `docker run` and compose examples work as written
