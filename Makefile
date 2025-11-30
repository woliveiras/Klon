SHELL := /bin/bash
.PHONY: help test test-short test-file test-watch lint vet build build-linux install check prepare push release
.PHONY: cover cover-html

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

help:
	@echo "Usage: make <target> [VERSION=vX.Y.Z | RELEASE=major|minor|patch]"
	@echo "Targets:"
	@echo "  test          - run all unit tests (go test ./...)"
	@echo "  test-short    - run short tests only (go test -run=^$ ./...)"
	@echo "  test-file     - run tests for a specific package/file (FILE=pkg/cli)"
	@echo "  test-watch    - watch Go files and re-run tests (requires entr)"
	@echo "  lint          - run linters (golangci-lint if available, fallback to go vet)"
	@echo "  vet           - run go vet ./..."
	@echo "  build         - build klon for the current platform"
	@echo "  build-linux   - build klon for linux/amd64 (override GOOS/GOARCH as needed)"
	@echo "  install       - build and install klon to $(BINDIR) (override PREFIX/BINDIR)"
	@echo "  cover         - run tests with coverage and write coverage.out"
	@echo "  cover-html    - generate coverage.html from coverage.out"
	@echo "  check         - ensure git is available and working tree is clean"
	@echo "  prepare       - run the release script (creates changelog, commit and tag)"
	@echo "  push          - push commits and tags to origin"
	@echo "  release       - prepare and push (safe wrapper)"
	@echo "Note: Do NOT run make with sudo. Run as your normal user."

test:
	@echo "Running unit tests..."
	go test ./...

test-short:
	@echo "Running short tests..."
	go test -run=^$ ./...

test-file:
	@if [ -z "$(FILE)" ]; then \
	  echo "Usage: make test-file FILE=./pkg/cli" >&2; exit 1; \
	fi
	go test $(FILE)

test-watch:
	@command -v entr >/dev/null || { echo "entr is required for test-watch (see https://eradman.com/entrproject/)"; exit 1; }
	@echo "Watching Go files and running tests on change..."
	@find . -name '*.go' | sort | entr -c go test ./...

lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
	  golangci-lint run ./...; \
	else \
	  echo "golangci-lint not found, running go vet ./... instead."; \
	  go vet ./...; \
	fi

vet:
	@echo "Running go vet..."
	go vet ./...

build:
	@echo "Building klon for current platform..."
	go build -o klon .

build-linux:
	@echo "Building klon for linux/amd64 (override GOOS/GOARCH to change)..."
	@GOOS=${GOOS:-linux} GOARCH=${GOARCH:-amd64} go build -o klon .

install: build
	@echo "Installing klon to $(BINDIR)..."
	@install -d $(BINDIR)
	@install -m 0755 klon $(BINDIR)/klon

cover:
	@echo "Running coverage (coverage.out)..."
	go test -coverprofile=coverage.out ./...

cover-html: cover
	@echo "Generating coverage.html..."
	go tool cover -html=coverage.out -o coverage.html

check:
	@command -v git >/dev/null || { echo "git is required" >&2; exit 1; }
	@if [ -n "$(shell git status --porcelain)" ]; then \
	  echo "Working tree is not clean. Commit or stash changes first." >&2; exit 1; \
	fi

prepare:
	@if [ -z "$(VERSION)" ] && [ -z "$(RELEASE)" ]; then \
	  echo "Either VERSION or RELEASE is required. Usage: make prepare VERSION=vX.Y.Z or RELEASE=major" >&2; exit 1; \
	fi
	@if [ -n "$(RELEASE)" ]; then \
	  echo "Preparing release type $(RELEASE)"; \
	  bash ./scripts/release.sh $(RELEASE); \
	else \
	  echo "Preparing release $(VERSION)"; \
	  bash ./scripts/release.sh $(VERSION); \
	fi

push:
	@# If VERSION is not provided, take the most recent semver tag
	@if [ -z "$(VERSION)" ]; then \
	  VERSION="$$(git --no-pager tag --list --sort=-v:refname | head -n1)"; \
	  if [ -z "$${VERSION}" ]; then \
	    echo "No VERSION supplied and no tags found. Run make prepare first or pass VERSION." >&2; exit 1; \
	  fi; \
	  echo "Detected VERSION=$${VERSION}"; \
	else \
	  VERSION="$(VERSION)"; \
	fi
	@echo "Pushing branch and tags for $${VERSION}"
	@git push origin main --follow-tags

release: check prepare push
	@echo "Release $(VERSION) complete."
