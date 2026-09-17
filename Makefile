.PHONY: dev test coverage build generate seed-perf seed-demo screenshots screenshots-headed e2e e2e-ui lint fmt clean

## Format all Go source files
fmt:
	gofmt -w .

## Run all linters: gofmt, go vet, staticcheck, deadcode
lint:
	@echo "── gofmt ──"
	@unformatted=$$(gofmt -l .); if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi
	@echo "── go vet ──"
	go vet ./...
	@echo "── go mod tidy check ──"
	go mod tidy
	@git diff --exit-code go.mod go.sum || (echo "go.mod or go.sum is not tidy. Run go mod tidy and commit."; exit 1)
	@echo "── staticcheck ──"
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
	@echo "── deadcode ──"
	go run golang.org/x/tools/cmd/deadcode@v0.50.0 ./... 2>/dev/null || true
	@echo "All checks passed."

## Regenerate OpenAPI spec from handler annotations
generate:
	go install github.com/swaggo/swag/cmd/swag@v1.16.6
	swag init -g cmd/server/main.go -o docs --parseInternal

## Seed database with synthetic data (100 tables × 20 columns × 1000 rows)
seed-perf:
	go run ./cmd/seed_perf

## Seed e2e database with 6 curated demo workspaces for marketing screenshots
seed-demo:
	go clean -cache
	go run ./cmd/seed_demo

## Regenerate demo screenshots: clean → seed → start servers → capture PNGs into screenshots/
screenshots:
	rm -rf e2e/e2e-data screenshots
	go clean -cache
	go run ./cmd/seed_demo
	cd e2e && pnpm install && pnpm exec playwright install chromium && pnpm exec playwright test --config screenshots.config.ts

## Same as screenshots but with the browser window visible
screenshots-headed:
	rm -rf e2e/e2e-data screenshots
	go clean -cache
	go run ./cmd/seed_demo
	cd e2e && pnpm install && pnpm exec playwright install chromium && pnpm exec playwright test --config screenshots.config.ts --headed

## Run Playwright e2e smoke tests (installs deps on first run)
e2e:
	rm -rf e2e/e2e-data
	cd e2e && pnpm install && pnpm exec playwright install chromium && pnpm test

## Run Playwright e2e tests with frontend coverage → coverage/lcov.info
e2e-coverage:
	rm -rf e2e/e2e-data coverage
	cd e2e && pnpm install && pnpm exec playwright install chromium && COVERAGE=1 pnpm test

## Open Playwright interactive UI (servers start automatically, reused if already running)
e2e-ui:
	cd e2e && pnpm install && pnpm exec playwright install chromium && pnpm test:ui

## Start backend (with live reload via air) and frontend in development mode
dev:
	invincible

## Run all Go integration tests
test:
	go test ./... -count=1

## Generate treemap coverage report (install: go install github.com/nikolaydubina/go-cover-treemap@v1.5.1)
coverage:
	go test ./... -count=1 -coverprofile=coverage.out -covermode=atomic -coverpkg=./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	go-cover-treemap -coverprofile coverage.out -w 1800 -h 900 -percent > coverage.svg

## Build Go binary and Vue production bundle
build: generate
	go build -o bin/degubase ./cmd/server
	cd ui && pnpm run build

## Drop and recreate the local database (next server start rebuilds from schema)
db-reset:
	rm -f degubase.db degubase.db-shm degubase.db-wal

## Remove build artifacts and local database
clean:
	rm -rf bin ui/dist degubase.db degubase.db-shm degubase.db-wal
