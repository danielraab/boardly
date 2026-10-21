# syntax=docker/dockerfile:1

# Famodash container image: one static binary that serves the API and the
# embedded web app. Build with `make docker` or `docker build -t famodash .`.
# Both build stages run on the build machine's platform; Go cross-compiles to
# the target platform, so multi-arch builds don't run build tools under emulation.

# --- 1. Web app: Vite writes into ../backend/internal/web/dist ----------------
FROM --platform=$BUILDPLATFORM node:26-slim AS web
ARG PNPM_VERSION=11.1.3
RUN npm install --global pnpm@${PNPM_VERSION}

WORKDIR /src/frontend
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/pnpm-workspace.yaml ./
RUN --mount=type=cache,id=famodash-pnpm-store,target=/pnpm/store \
    pnpm install --frozen-lockfile --store-dir /pnpm/store

COPY frontend/ ./
RUN pnpm build

# --- 2. Server binary with the web app embedded ------------------------------
FROM --platform=$BUILDPLATFORM golang:1.26 AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,id=famodash-gomod,target=/go/pkg/mod \
    go mod download

COPY backend/ ./
COPY --from=web /src/backend/internal/web/dist ./internal/web/dist

ARG TARGETOS TARGETARCH
RUN --mount=type=cache,id=famodash-gomod,target=/go/pkg/mod \
    --mount=type=cache,id=famodash-gobuild,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/famodash . \
 && mkdir /out/data

# --- 3. Runtime: CA certificates and a nonroot user, nothing else -------------
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/famodash /famodash
# A named volume takes over this directory's owner, so it is writable for nonroot.
COPY --from=build --chown=65532:65532 /out/data /data

# FAMODASH_BASE_URL is required and deliberately has no default here.
ENV FAMODASH_DB_PATH=/data/famodash.db
VOLUME /data
EXPOSE 8080
USER 65532:65532

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/famodash", "healthcheck"]
ENTRYPOINT ["/famodash"]
