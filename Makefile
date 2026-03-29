BUILD_DIR := bin
ARC_BIN := $(BUILD_DIR)/arc
ARC_DESKTOP_BIN := $(BUILD_DIR)/arc-desktop

.PHONY: build build-desktop install test test-desktop smoke docs-install docs-dev docs-build desktop-preview desktop-wails-build desktop-wails-package

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(ARC_BIN) ./cmd/arc

build-desktop:
	mkdir -p $(BUILD_DIR)
	go build -o $(ARC_DESKTOP_BIN) ./cmd/arc-desktop

install:
	go install ./cmd/arc

test:
	GOCACHE=/tmp/agent-os-gocache go test ./...

test-desktop:
	GOCACHE=/tmp/agent-os-gocache go test ./...
	node --test apps/desktop/tests/*.test.mjs

smoke: build
	$(ARC_BIN) doctor || true

docs-install:
	cd apps/docs && npm install

docs-dev:
	cd apps/docs && npm run start

docs-build:
	cd apps/docs && npm run build

desktop-preview: build-desktop
	$(ARC_DESKTOP_BIN)

desktop-wails-build:
	go build -tags wails -o $(BUILD_DIR)/arc-desktop-wails ./cmd/arc-desktop-wails

desktop-wails-package:
	./scripts/build_desktop_app.sh
