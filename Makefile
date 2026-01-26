.PHONY: build install clean test generate

BINARIES := ratelord ratelord-d ratelord-tui ratelord-sim
BUILD_DIR := bin

# Get version info for ldflags
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILD_TIME)

generate:
	go generate ./...

web-build:
	cd web && npm install && npm run build

build: generate $(BINARIES)

ratelord: cmd/ratelord/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$@ ./cmd/ratelord
ifeq ($(shell uname),Darwin)
	@codesign -s - -f $(BUILD_DIR)/$@ 2>/dev/null || true
	@echo "Signed $@ for macOS"
endif

ratelord-d: web-build cmd/ratelord-d/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$@ ./cmd/ratelord-d
ifeq ($(shell uname),Darwin)
	@codesign -s - -f $(BUILD_DIR)/$@ 2>/dev/null || true
	@echo "Signed $@ for macOS"
endif

ratelord-tui: cmd/ratelord-tui/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$@ ./cmd/ratelord-tui
ifeq ($(shell uname),Darwin)
	@codesign -s - -f $(BUILD_DIR)/$@ 2>/dev/null || true
	@echo "Signed $@ for macOS"
endif

ratelord-sim: cmd/ratelord-sim/main.go
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$@ ./cmd/ratelord-sim
ifeq ($(shell uname),Darwin)
	@codesign -s - -f $(BUILD_DIR)/$@ 2>/dev/null || true
	@echo "Signed $@ for macOS"
endif

install: generate
	go install -ldflags "$(LDFLAGS)" ./cmd/ratelord
	go install -ldflags "$(LDFLAGS)" ./cmd/ratelord-d
	go install -ldflags "$(LDFLAGS)" ./cmd/ratelord-tui
	go install -ldflags "$(LDFLAGS)" ./cmd/ratelord-sim

clean:
	rm -f $(BUILD_DIR)/*
	rm -rf web/dist web/node_modules

test:
	go test ./...