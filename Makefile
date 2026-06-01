BINARY := gmailnotifier.30s
PKG    := github.com/bricklen/gmailnotifier
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# --- Code signing / notarization -------------------------------------------
# Set these in your environment (e.g. ~/.zshrc) or pass them on the command
# line: `make release SIGN_IDENTITY="Developer ID Application: ..."`
#
#   SIGN_IDENTITY    : The exact name of your Developer ID Application cert,
#                      e.g. "Developer ID Application: Jane Doe (TEAMID)".
#                      Find with: security find-identity -v -p codesigning
#   NOTARY_PROFILE   : A keychain profile created once via
#                      `xcrun notarytool store-credentials NAME ...`
#   BUNDLE_ID        : The CFBundleIdentifier baked into the signature.
SIGN_IDENTITY  ?=
NOTARY_PROFILE ?= gmailnotifier-notary
BUNDLE_ID      ?= com.bricklen.gmailnotifier
# ---------------------------------------------------------------------------

.PHONY: build test vet lint clean install dist tidy sign notarize release verify-signed

build:
	@mkdir -p bin
	go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/gmailnotifier

test:
	go test ./... -race -count=1

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf bin dist

# Detect SwiftBar's configured plugin folder. Falls back to the documented
# default if SwiftBar isn't installed or hasn't been launched. Override
# with `make install PLUGIN_DIR=...` if you need to.
SWIFTBAR_PLUGIN_DIR := $(shell /usr/bin/defaults read com.ameba.SwiftBar PluginDirectory 2>/dev/null)
PLUGIN_DIR ?= $(if $(SWIFTBAR_PLUGIN_DIR),$(SWIFTBAR_PLUGIN_DIR),$(HOME)/Library/Application Support/SwiftBar/Plugins)

install: build
	@mkdir -p "$(PLUGIN_DIR)"
	@# Remove any prior gmailnotifier* files/symlinks so SwiftBar can't
	@# end up running two copies of the plugin side by side.
	@find "$(PLUGIN_DIR)" -maxdepth 1 \( -type f -o -type l \) -name 'gmailnotifier*' -print -delete | sed 's/^/removed: /'
	cp bin/$(BINARY) "$(PLUGIN_DIR)/$(BINARY)"
	chmod +x "$(PLUGIN_DIR)/$(BINARY)"
	@echo "Installed to: $(PLUGIN_DIR)/$(BINARY)"
	@echo "Run '$(PLUGIN_DIR)/$(BINARY) setup' to configure accounts."

dist:
	@mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64 ./cmd/gmailnotifier
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64 ./cmd/gmailnotifier
	@cd dist && shasum -a 256 $(BINARY)-darwin-* > SHA256SUMS

# Sign both dist binaries with your Developer ID Application cert.
# `--options runtime` enables the hardened runtime (required for notarization).
# `--timestamp` adds a secure timestamp (also required).
sign:
	@if [ -z "$(SIGN_IDENTITY)" ]; then \
		echo "ERROR: SIGN_IDENTITY is not set."; \
		echo "Find your identity with: security find-identity -v -p codesigning"; \
		echo "Then run: make sign SIGN_IDENTITY=\"Developer ID Application: Your Name (TEAMID)\""; \
		exit 1; \
	fi
	@for arch in amd64 arm64; do \
		bin="dist/$(BINARY)-darwin-$$arch"; \
		echo "Signing $$bin"; \
		codesign --force --sign "$(SIGN_IDENTITY)" \
			--identifier "$(BUNDLE_ID)" \
			--options runtime --timestamp \
			"$$bin" || exit 1; \
	done
	@# Regenerate checksums after signing since the binaries changed.
	@cd dist && shasum -a 256 $(BINARY)-darwin-* > SHA256SUMS
	@$(MAKE) verify-signed

verify-signed:
	@for arch in amd64 arm64; do \
		bin="dist/$(BINARY)-darwin-$$arch"; \
		echo "Verifying $$bin"; \
		codesign --verify --verbose=2 "$$bin" 2>&1 | sed 's/^/  /'; \
		codesign -dvv "$$bin" 2>&1 | grep -E 'Authority|TeamIdentifier|Signature|Timestamp' | sed 's/^/  /'; \
	done

# Notarize each binary by zipping it, submitting via notarytool, and
# waiting for Apple's response. Standalone CLI binaries cannot be stapled
# (the ticket lives online), so Gatekeeper checks notarization status
# over the network on first launch.
notarize:
	@if [ -z "$(NOTARY_PROFILE)" ]; then \
		echo "ERROR: NOTARY_PROFILE is not set."; exit 1; \
	fi
	@for arch in amd64 arm64; do \
		bin="dist/$(BINARY)-darwin-$$arch"; \
		zip="$$bin.zip"; \
		echo "Zipping $$bin → $$zip"; \
		/usr/bin/ditto -c -k --keepParent "$$bin" "$$zip"; \
		echo "Submitting $$zip to Apple notary service"; \
		xcrun notarytool submit "$$zip" \
			--keychain-profile "$(NOTARY_PROFILE)" \
			--wait || exit 1; \
		echo "Notarized: $$bin"; \
	done
	@# Refresh checksums one final time so users verify the actual files
	@# that ship in the release.
	@cd dist && shasum -a 256 $(BINARY)-darwin-* > SHA256SUMS

# Full release pipeline: clean → cross-build → sign → notarize.
# Run this once your git tag is in place and HEAD == tag.
release: clean dist sign notarize
	@echo
	@echo "Release artifacts (signed + notarized):"
	@ls -lh dist/$(BINARY)-darwin-* dist/SHA256SUMS
	@echo
	@echo "Next: gh release create $(VERSION) dist/$(BINARY)-darwin-amd64 dist/$(BINARY)-darwin-arm64 dist/SHA256SUMS"
