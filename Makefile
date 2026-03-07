## Luminarr Makefile

MODULE  := github.com/luminarr/luminarr
BINARY  := luminarr
BIN_DIR := ./bin
CMD     := ./cmd/luminarr

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS   := -ldflags "\
  -X $(MODULE)/internal/version.Version=$(VERSION) \
  -X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME) \
  -X $(MODULE)/internal/config.DefaultTMDBKey=$(LUMINARR_TMDB_API_KEY)"

IMAGE ?= ghcr.io/luminarr/luminarr

.PHONY: build run dev test test/unit test/integration test/cover test/race \
        lint check install-hooks generate migrate clean docker docker/push help

## build: Compile the binary into ./bin/luminarr
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) $(CMD)

## run: Build and run the server (no hot reload)
run: build
	$(BIN_DIR)/$(BINARY)

## dev: Run with hot reload via air (install: go install github.com/air-verse/air@latest)
dev:
	air

## test: Run all tests
test:
	go test ./...

## test/unit: Run only unit tests (skips integration tests)
test/unit:
	go test -short ./...

## test/integration: Run integration tests (requires no external services)
test/integration:
	go test -run Integration ./...

## test/cover: Run tests and open coverage report
test/cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test/race: Run tests with the race detector (used in CI)
test/race:
	go test -race ./...

## lint: Run golangci-lint
lint:
	golangci-lint run

## check: Run all pre-push checks (Go lint + TypeScript type-check)
check: lint
	cd web/ui && npx tsc --noEmit

## install-hooks: Install git hooks from the hooks/ directory
install-hooks:
	cp hooks/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push
	@echo "pre-push hook installed"

## generate: Regenerate sqlc query code
generate:
	sqlc generate

## migrate: Run database migrations (uses LUMINARR_DATABASE_PATH or default)
migrate:
	go run $(CMD) -migrate-only

## clean: Remove build artifacts
clean:
	rm -rf $(BIN_DIR) tmp coverage.out coverage.html

## docker: Build the Docker image locally (IMAGE=ghcr.io/luminarr/luminarr)
docker:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg BUILD_TIME=$(BUILD_TIME) \
	  -t $(IMAGE):$(VERSION) \
	  -t $(IMAGE):latest \
	  -f docker/Dockerfile .

## docker/run: Build and run locally with Docker Compose (data persists in named volume)
docker/run:
	docker compose -f docker/docker-compose.yml up --build

## docker/push: Build and push the image to the registry
docker/push: docker
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

## help: Print this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'
