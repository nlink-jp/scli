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

PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build build-all test lint check setup clean

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
	CGO_ENABLED=1 GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=1 GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	@scripts/codesign-darwin.sh dist/$(BINARY_NAME)-darwin-amd64 "$(CODESIGN_IDENTITY)"
	@scripts/codesign-darwin.sh dist/$(BINARY_NAME)-darwin-arm64 "$(CODESIGN_IDENTITY)"

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

## package: Build and package binaries as .zip archives for all platforms, notarize darwin
package: build-all
	$(foreach platform,$(PLATFORMS), \
		$(eval GOOS=$(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH=$(word 2,$(subst /, ,$(platform)))) \
		$(eval EXT=$(if $(filter windows,$(GOOS)),.exe,)) \
		$(eval ARCHIVE=dist/$(BINARY_NAME)-$(VERSION)-$(GOOS)-$(GOARCH).zip) \
		zip -j $(ARCHIVE) dist/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(EXT) LICENSE README.md ; \
	)
	@scripts/notarize-darwin.sh dist/$(BINARY_NAME)-$(VERSION)-darwin-amd64.zip "$(NOTARY_PROFILE)"
	@scripts/notarize-darwin.sh dist/$(BINARY_NAME)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"

## clean: Remove build artifacts
clean:
	rm -rf dist/

## help: Show this help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
