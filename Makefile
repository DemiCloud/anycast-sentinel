# Project metadata
NAME     := anycast-sentinel
VERSION  := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILT_BY ?= DemiCloud

# Install destination
PREFIX  ?= /usr/local
BINDIR  ?= $(PREFIX)/sbin

# Base linker flags (dev builds — version only)
LDFLAGS := \
	-X github.com/demicloud/anycast-sentinel/internal/version.Version=$(VERSION)

# Release linker flags (strip + trim + full metadata)
RELEASE_LDFLAGS := $(LDFLAGS) -s -w \
	-X github.com/demicloud/anycast-sentinel/internal/version.Commit=$(shell git rev-parse --short HEAD) \
	-X github.com/demicloud/anycast-sentinel/internal/version.BuildDate=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
	-X github.com/demicloud/anycast-sentinel/internal/version.BuiltBy=$(BUILT_BY)

# Target platforms for release
PLATFORMS := linux/amd64 linux/arm64

.PHONY: build test vet install release clean

# Default: development build (debug symbols, version = "dev" if no tags)
build:
	mkdir -p build
	go mod tidy
	go build -ldflags "$(LDFLAGS)" -o build/$(NAME) ./cmd/anycast-sentinel

# Run all tests
test:
	go test ./...

# Run static analysis
vet:
	go vet ./...

# Install binary into $(BINDIR) — supports DESTDIR for packaging
install: build
	install -D -m 0755 build/$(NAME) $(DESTDIR)$(BINDIR)/$(NAME)

# Release builds: static, stripped, trimpath, multi-arch tarballs in dist/
release: clean
	mkdir -p dist
	go mod tidy
	@set -e; for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d/ -f1); \
		ARCH=$$(echo $$platform | cut -d/ -f2); \
		echo "building $$OS/$$ARCH"; \
		GOOS=$$OS GOARCH=$$ARCH CGO_ENABLED=0 go build -trimpath \
			-ldflags "$(RELEASE_LDFLAGS)" \
			-o dist/$(NAME) ./cmd/anycast-sentinel; \
		tar -czf dist/$(NAME)_$(VERSION)_$${OS}_$${ARCH}.tar.gz -C dist $(NAME); \
		rm dist/$(NAME); \
	done

clean:
	rm -rf build dist
