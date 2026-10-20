.PHONY: dev-backend dev-frontend build build-frontend build-backend test check

# Run the Go server on :8080 (API + last frontend build) with the dev compose
# services (mailpit, sqlite-web); they stop again when the server stops.
dev-backend:
	./scripts/dev-backend.sh

# Run the Vite dev server on :3000; it proxies /api to :8080.
dev-frontend:
	cd frontend && pnpm dev

# Build the frontend into backend/internal/web/dist and embed it into bin/famodash.
build: build-frontend build-backend

build-frontend:
	cd frontend && pnpm install --frozen-lockfile && pnpm build

build-backend:
	cd backend && go build -o ../bin/famodash .

test:
	cd backend && go test ./...
	cd frontend && pnpm test

check:
	cd backend && go vet ./... && test -z "$$(gofmt -l .)"
	cd frontend && pnpm check && pnpm exec tsc --noEmit
