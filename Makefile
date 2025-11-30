SHELL := /bin/bash
.PHONY: help check prepare push release

help:
	@echo "Usage: make <target> [VERSION=vX.Y.Z | RELEASE=major|minor|patch]"
	@echo "Targets:"
	@echo "  check     - ensure git is available and working tree is clean"
	@echo "  prepare   - run the release script (creates changelog, commit and tag)"
	@echo "  push      - push commits and tags to origin"
	@echo "  release   - prepare and push (safe wrapper)"

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
	@if [ -z "$(VERSION)" ]; then \
	  echo "VERSION is required for push. Run prepare first and set VERSION to the tag created." >&2; exit 1; \
	fi
	@echo "Pushing branch and tags for $(VERSION)"
	@git push origin main --follow-tags

release: check prepare push
	@echo "Release $(VERSION) complete."
