# Pin by commit hash rather than tag, so a re-pointed/malleable tag can't change
# what we build. Keep the trailing comment's tag in sync with the hash and with
# the `version:` field in .custom-gcl.yml.
GOLANGCI_LINT_HASH ?= c0d3ddc9cf3faa61a4e378e879ece580256d76e5 # v2.12.2

BIN_DIR := bin

.PHONY: build build-golangci-lint install-golangci-lint test clean

# Standalone multichecker over every rules/<linter>.Analyzer.
build:
	go build -o $(BIN_DIR)/wormhole-lint ./cmd/wormhole-lint

# Pinned upstream golangci-lint, dropped into $(BIN_DIR). Used as the driver
# for `golangci-lint custom`; keep its version aligned with the `version:`
# field in .custom-gcl.yml.
install-golangci-lint:
	GOOS= GOARCH= GOBIN=$(abspath $(BIN_DIR)) go install \
	  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_HASH)

# Custom golangci-lint with our module plugins baked in. Driven by
# .custom-gcl.yml in this directory (golangci-lint custom reads it from the
# cwd); output goes to ./bin per `destination:`.
build-golangci-lint: install-golangci-lint
	$(BIN_DIR)/golangci-lint custom

test:
	go test -v ./...
	@for mod in $$(find rules -name go.mod); do \
	  dir=$$(dirname $$mod); \
	  echo "==> testing $$dir"; \
	  (cd $$dir && go test -v ./...) || exit $$?; \
	done

clean:
	rm -rf $(BIN_DIR)
