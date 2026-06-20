BINARY      := kui
MODULE      := github.com/hrodrig/kui
DIST        := dist
VERSION_RAW ?= $(shell cat VERSION 2>/dev/null | tr -d '\n\r')
VERSION     := $(patsubst v%,%,$(VERSION_RAW))
TAG         := v$(VERSION)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDDATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
BRANCH      := $(shell git symbolic-ref --short HEAD 2>/dev/null || echo unknown)

COVERAGE_MIN  ?= 4
GRYPE_FAIL_ON ?= high

# Minimum golang.org/x/net (explicit go.mod pin; go mod tidy drops it → templ/afero resolve older).
X_NET_MIN_VERSION ?= v0.55.0
# Minimum golang.org/x/crypto (direct require; afero pulls v0.32.0 transitively).
X_CRYPTO_MIN_VERSION ?= v0.52.0

BOOTSTRAP_VERSION := 5.3.3
HTMX_VERSION      := 2.0.4
CHARTJS_VERSION   := 4.4.1
STATIC_DIR        := internal/server/static

LDFLAGS := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(BUILDDATE)' \
	-X '$(MODULE)/internal/version.Branch=$(BRANCH)'

KUI_ADMIN_PASSWORD ?= dev-admin
KIKO_API_KEY       ?= local-dev-key
KIKO_URL           ?= http://127.0.0.1:8080

.PHONY: help build install test cover cover-check lint lint-fix fmt-check vet gocyclo govulncheck grype security
.PHONY: check-mapstructure-pin check-x-net-pin check-x-crypto-pin vendor-static templ generate docker-build docker-scan docker-buildx
.PHONY: run stop seed-kiko seed-kiko-history release-check release snapshot semver-check tools clean

help: ## Print this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

vendor-static: ## Download vendored Bootstrap + HTMX (no npm)
	@mkdir -p $(STATIC_DIR)/css $(STATIC_DIR)/js
	curl -fsSL -o $(STATIC_DIR)/css/bootstrap.min.css \
		https://cdn.jsdelivr.net/npm/bootstrap@$(BOOTSTRAP_VERSION)/dist/css/bootstrap.min.css
	curl -fsSL -o $(STATIC_DIR)/js/bootstrap.bundle.min.js \
		https://cdn.jsdelivr.net/npm/bootstrap@$(BOOTSTRAP_VERSION)/dist/js/bootstrap.bundle.min.js
	curl -fsSL -o $(STATIC_DIR)/js/htmx.min.js \
		https://unpkg.com/htmx.org@$(HTMX_VERSION)/dist/htmx.min.js
	curl -fsSL -o $(STATIC_DIR)/js/chart.umd.min.js \
		https://cdn.jsdelivr.net/npm/chart.js@$(CHARTJS_VERSION)/dist/chart.umd.min.js

templ: ## Install templ code generator
	@which templ >/dev/null 2>&1 || go install github.com/a-h/templ/cmd/templ@v0.3.977

generate: templ ## Generate templ Go files
	templ generate

build: generate ## Build binary for current platform
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/$(BINARY)

install: generate ## Install to $$GOBIN
	go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/$(BINARY)

test: generate ## Run tests with race detector
	go test -count=1 -race ./...

cover: generate ## Run tests + coverage report
	go test -count=1 -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -func=coverage.out | tail -1

cover-check: cover ## Run tests + coverage gate
	@go tool cover -func=coverage.out | tail -1 | awk '{print $$NF}' | tr -d '%' | \
		while read pct; do \
			if [ "$$(echo "$$pct < $(COVERAGE_MIN)" | bc -l)" -eq 1 ]; then \
				echo "FAIL: coverage $$pct% < $(COVERAGE_MIN)%"; exit 1; \
			fi; \
			echo "PASS: coverage $$pct% >= $(COVERAGE_MIN)%"; \
		done

lint: check-mapstructure-pin check-x-net-pin check-x-crypto-pin fmt-check vet gocyclo ## Run all linters

lint-fix: ## Auto-fix formatting
	gofmt -s -w .

fmt-check: ## Check gofmt compliance
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "FAIL: unformatted files:"; \
		gofmt -s -l .; \
		exit 1; \
	fi
	@echo "PASS: gofmt -s"

vet: generate ## Run go vet
	go vet ./...

check-mapstructure-pin: ## Verify go-viper/mapstructure >= v2.4.0 (CVE pin)
	@if grep -q 'go-viper/mapstructure/v2 v2\.4\.' go.mod; then \
		echo "PASS: mapstructure pinned"; \
	else \
		echo "FAIL: mapstructure not pinned to v2.4.0"; \
		exit 1; \
	fi

# Ensure explicit x/net pin stays in go.mod — go mod tidy drops it; older transitive x/net downgrades x/crypto.
check-x-net-pin: ## Verify golang.org/x/net pin (see X_NET_MIN_VERSION)
	@echo "Checking golang.org/x/net pin (minimum $(X_NET_MIN_VERSION))..."
	@grep -q 'golang.org/x/net $(X_NET_MIN_VERSION)' go.mod || { \
		echo "go.mod missing pin; re-pinning golang.org/x/net@$(X_NET_MIN_VERSION)..."; \
		go get golang.org/x/net@$(X_NET_MIN_VERSION); \
	}
	@resolved=$$(go list -m -f '{{.Version}}' golang.org/x/net); \
	if [ "$$resolved" != "$(X_NET_MIN_VERSION)" ]; then \
		echo "golang.org/x/net resolved to $$resolved; re-pinning to $(X_NET_MIN_VERSION)..."; \
		go get golang.org/x/net@$(X_NET_MIN_VERSION); \
	fi
	@echo "golang.org/x/net pin OK ($(X_NET_MIN_VERSION))"

# Ensure direct x/crypto stays at $(X_CRYPTO_MIN_VERSION)+ (afero transitively wants older).
check-x-crypto-pin: ## Verify golang.org/x/crypto pin (see X_CRYPTO_MIN_VERSION)
	@echo "Checking golang.org/x/crypto pin (minimum $(X_CRYPTO_MIN_VERSION))..."
	@grep -q 'golang.org/x/crypto $(X_CRYPTO_MIN_VERSION)' go.mod || { \
		echo "go.mod missing pin; re-pinning golang.org/x/crypto@$(X_CRYPTO_MIN_VERSION)..."; \
		go get golang.org/x/crypto@$(X_CRYPTO_MIN_VERSION); \
	}
	@resolved=$$(go list -m -f '{{.Version}}' golang.org/x/crypto); \
	if [ "$$resolved" != "$(X_CRYPTO_MIN_VERSION)" ]; then \
		echo "golang.org/x/crypto resolved to $$resolved; re-pinning to $(X_CRYPTO_MIN_VERSION)..."; \
		go get golang.org/x/crypto@$(X_CRYPTO_MIN_VERSION); \
	fi
	@echo "golang.org/x/crypto pin OK ($(X_CRYPTO_MIN_VERSION))"

gocyclo: ## Check cyclomatic complexity (max 14; excludes templ-generated *_templ.go)
	@which gocyclo >/dev/null 2>&1 || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@files=$$(find . -name '*.go' ! -name '*_templ.go' -print); \
	if [ -n "$$files" ] && gocyclo -over 14 $$files | grep .; then \
		echo "FAIL: functions exceed gocyclo limit 14"; \
		exit 1; \
	fi
	@echo "PASS: gocyclo <= 14"

govulncheck: generate ## Check Go vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

grype: ## Directory CVE scan
	@if command -v grype >/dev/null 2>&1; then \
		grype --fail-on $(GRYPE_FAIL_ON) --exclude './dist/**,./$(BINARY)' .; \
	else \
		echo "grype not installed, skipping"; \
	fi

security: govulncheck gocyclo grype ## Run all security checks

tools: ## Install security tooling
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

docker-build: ## Build Docker image (local)
	@DOCKER_BUILDKIT=1 docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILDDATE=$(BUILDDATE) \
		--build-arg BRANCH=$(BRANCH) \
		-t $(BINARY):local .

docker-buildx: ## Build multi-arch Docker image
	@docker buildx build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILDDATE=$(BUILDDATE) \
		--build-arg BRANCH=$(BRANCH) \
		--platform linux/amd64,linux/arm64 \
		-t $(BINARY):local .

docker-scan: docker-build ## Build + Grype image scan
	@which grype >/dev/null 2>&1 || (echo "grype not installed, skipping scan"; exit 0)
	grype --fail-on $(GRYPE_FAIL_ON) $(BINARY):local

run: build ## Build and run locally (needs kiko on :8080 with same KIKO_API_KEY)
	@mkdir -p data
	@echo "kui dev → http://127.0.0.1:3000"
	@echo "login: admin@localhost / $(KUI_ADMIN_PASSWORD)"
	@echo "kiko:  $(KIKO_URL) (api key: $(KIKO_API_KEY))"
	KUI_ADMIN_PASSWORD=$(KUI_ADMIN_PASSWORD) \
	KIKO_API_KEY=$(KIKO_API_KEY) \
	KIKO_URL=$(KIKO_URL) \
	./$(BINARY) serve -c configs/kui.dev.yml

stop: ## Stop kui/kiko dev processes on :3000 / :8080
	@-pkill -f '[./]kui serve' 2>/dev/null || true
	@-pkill -f '[./]kiko serve' 2>/dev/null || true
	@echo "stopped kui/kiko (if running)"

seed-kiko: ## Inject sample hits into local kiko (override: KIKO_SEED_HOSTS=host1,host2)
	@chmod +x scripts/seed-kiko.sh
	KIKO_URL=$(KIKO_URL) KIKO_API_KEY=$(KIKO_API_KEY) ./scripts/seed-kiko.sh

seed-kiko-history: ## Backfill 90 days of demo hits in kiko SQLite (for README screenshots)
	@go run ./scripts/seed-kiko-history -reset -days $${SEED_DAYS:-90} -db $${KIKO_DB:-../kiko/data/kiko.db}

release-check: semver-check check-mapstructure-pin check-x-net-pin check-x-crypto-pin fmt-check vet cover-check security ## Release gate: all quality checks

release: ## Release via GoReleaser (main branch only)
	@if [ "$(BRANCH)" != "main" ]; then echo "FAIL: releases from main only"; exit 1; fi
	$(MAKE) release-check
	goreleaser release --clean

snapshot: ## GoReleaser snapshot to dist/ (no publish)
	KUI_SNAPSHOT_VERSION=$(VERSION)-next goreleaser release --snapshot --clean

semver-check: ## Validate VERSION is semver MAJOR.MINOR.PATCH
	@if ! grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' VERSION; then \
		echo "FAIL: VERSION must be MAJOR.MINOR.PATCH (got: $$(cat VERSION))"; \
		exit 1; \
	fi
	@echo "PASS: VERSION=$$(cat VERSION)"

clean: ## Remove build artifacts
	rm -rf $(BINARY) bin/ $(DIST)/ coverage.out
