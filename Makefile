.PHONY: build dev test clean web-build go-build dev-up dev-down dev-logs

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY  := alertlens
WEB_DIR := web
DIST_DIR := dist

# ─── Primary targets ────────────────────────────────────────────────────────

build: web-build go-build
	@echo "✓ Build complete: ./$(BINARY)"

web-build: $(WEB_DIR)/node_modules
	cd $(WEB_DIR) && npm run build

$(WEB_DIR)/node_modules: $(WEB_DIR)/package.json
	cd $(WEB_DIR) && npm ci

go-build:
	CGO_ENABLED=0 go build \
		-ldflags="-s -w -X main.version=$(VERSION)" \
		-o $(BINARY) .

# ─── Development ────────────────────────────────────────────────────────────

dev-up:
	docker compose -f dev/docker-compose.yml up -d --build
	@echo "✓ Dev stack running — AlertLens UI at http://localhost:9000"

dev-down:
	docker compose -f dev/docker-compose.yml down

dev-logs:
	docker compose -f dev/docker-compose.yml logs -f

dev-backend:
	go run . -config config.example.yaml

dev-frontend:
	cd $(WEB_DIR) && npm run dev

# ─── Tests ──────────────────────────────────────────────────────────────────

test:
	go test ./... -v -cover

# ─── Cleanup ────────────────────────────────────────────────────────────────

clean:
	rm -rf $(BINARY) $(DIST_DIR) $(WEB_DIR)/build $(WEB_DIR)/.svelte-kit
