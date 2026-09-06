BINARY_NAME := scli
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-X github.com/nlink-jp/scli/cmd.version=$(VERSION)"

# macOS Developer ID signing / notarization (see nlink-jp/.github
# CONVENTIONS.md §Code Signing). Defaults match any Developer ID
# Application cert in the keychain and the org-standard notary
# profile. Builds without these fall back to ad-hoc / un-notarized
# with a one-line warning — see scripts/codesign-darwin.sh.
CODESIGN_IDENTITY ?= Developer ID Application
NOTARY_PROFILE    ?= nlink-jp-notary

# darwin ships arm64 only (no amd64, no universal). linux/windows keep their matrix.
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build build-all test lint check setup verify-release clean

## build: Build for the current platform
build:
	@mkdir -p dist
	go build $(LDFLAGS) -o dist/$(BINARY_NAME) .
	@scripts/codesign-darwin.sh dist/$(BINARY_NAME) "$(CODESIGN_IDENTITY)"

## build-all: Cross-compile for all target platforms
# Darwin: CGO_ENABLED=1 required for OS Keychain (Security.framework)
# Linux:  CGO_ENABLED=0 — keychain uses secret-tool (no CGO needed)
# Windows: CGO_ENABLED=0 for cross-compilation; keychain unavailable in cross-compiled binaries
build-all:
	@mkdir -p dist
	CGO_ENABLED=1 GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	@scripts/codesign-darwin.sh dist/$(BINARY_NAME)-darwin-arm64 "$(CODESIGN_IDENTITY)" "$(BINARY_NAME)"

## test: Run the full test suite
test:
	go test -race -cover ./...

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## check: Run lint + test + build-all (used by Git hooks)
check: lint test build-all

## setup: Install Git hooks
setup:
	@cp scripts/hooks/pre-commit .git/hooks/pre-commit
	@cp scripts/hooks/pre-push   .git/hooks/pre-push
	@chmod +x .git/hooks/pre-commit .git/hooks/pre-push
	@echo "Git hooks installed."

## package: Build all platforms, archive with version suffix (zip for
## darwin/windows, tar.gz for linux), bundle the canonical binary +
## README.md + LICENSE, and notarize the darwin build. Asset naming
## follows the org Release Archive Standard
## (scli-vX.Y.Z-<os>-<arch>.<ext>).
package: build-all
	@cd dist && for p in $(PLATFORMS); do os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = windows ] && ext=".exe"; \
		stage=_pkg; rm -rf $$stage; mkdir -p $$stage; \
		cp "$(BINARY_NAME)-$$os-$$arch$$ext" "$$stage/$(BINARY_NAME)$$ext"; \
		cp ../README.md ../LICENSE $$stage/; \
		base="$(BINARY_NAME)-$(VERSION)-$$os-$$arch"; \
		if [ "$$os" = linux ]; then ( cd $$stage && tar -czf "../$$base.tar.gz" * ); \
		else ( cd $$stage && zip -q "../$$base.zip" * ); fi; \
		rm -rf $$stage; \
	done
	@scripts/notarize-darwin.sh dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"

## verify-release: refuse to release an un-notarized zip (marker gate)
verify-release:
	@test -f "dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip.notarized" || { \
		echo "verify-release: FAIL — $(BINARY_NAME)-$(VERSION)-darwin-arm64.zip has no notarization marker."; \
		echo "  make package must end with '[notarize] ...: Accepted'. Do not upload this zip."; \
		exit 1; }
	@test "dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip.notarized" -nt "dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip" || { \
		echo "verify-release: FAIL — the zip was rebuilt after its marker (re-run make package)."; \
		exit 1; }
	@tmp=$$(mktemp -d) && \
		unzip -oq "dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip" -d "$$tmp" && \
		"$$tmp/$(BINARY_NAME)" --version && \
		spctl -a -vv -t install "$$tmp/$(BINARY_NAME)" 2>&1 | head -2 || true; \
		rm -rf "$$tmp"
	@echo "verify-release: OK ($(VERSION), notarization marker present)"

## clean: Remove build artifacts
clean:
	rm -rf dist/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

# Homebrew tap generation (see scripts/release-brew.mk). After `make package`,
# `make brew` generates this formula from the built darwin-arm64 zip into the
# local nlink-jp/homebrew-tap checkout. The package target is unchanged.
BREW_KIND := formula
BREW_DESC := Terminal Slack client for channels, DMs, search, and unread
BREW_NAME := $(BINARY_NAME)
include scripts/release-brew.mk

## test-linux: run the test suite inside a Linux container (podman/docker)
.PHONY: test-linux
test-linux:
	@scripts/test-linux.sh
