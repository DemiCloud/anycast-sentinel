# Project metadata
NAME := anycast-sentinel
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILT_BY ?= DemiCloud

# Base linker flags (used for dev builds)
LDFLAGS := \
	-X github.com/demicloud/anycast-sentinel/internal/version.Version=$(VERSION)

# Release linker flags (strip symbols + trim paths + metadata)
RELEASE_LDFLAGS := $(LDFLAGS) -s -w \
	-X github.com/demicloud/anycast-sentinel/internal/version.Commit=$(shell git rev-parse --short HEAD) \
	-X github.com/demicloud/anycast-sentinel/internal/version.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	-X github.com/demicloud/anycast-sentinel/internal/version.BuiltBy=$(BUILT_BY)

# Target platforms
PLATFORMS := linux/amd64 linux/arm64

# Default build (debug symbols kept)
build:
	mkdir -p build
	go mod tidy
	go build -ldflags "$(LDFLAGS)" -o build/$(NAME) ./cmd/anycast-sentinel

# Release build (static, stripped, reproducible-ish)
release: clean
	mkdir -p dist
	go mod tidy
	$(foreach platform,$(PLATFORMS), \
		OS=$(word 1,$(subst /, ,$(platform))); \
		ARCH=$(word 2,$(subst /, ,$(platform))); \
		echo "Building $$OS/$$ARCH"; \
		GOOS=$$OS GOARCH=$$ARCH CGO_ENABLED=0 go build -trimpath -ldflags "$(RELEASE_LDFLAGS)" -o dist/$(NAME) ./cmd/anycast-sentinel; \
		tar -czf dist/$(NAME)_$(VERSION)_$${OS}_$${ARCH}.tar.gz -C dist $(NAME); \
		rm dist/$(NAME); \
	)

clean:
	rm -rf build dist
