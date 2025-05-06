SOLANA_VERSION := 2.1.11

NETWORK ?= mainnet

ifeq ($(NETWORK),mainnet)
    BUILD_FEATURES = $(SVM)
else
    BUILD_FEATURES = $(SVM),$(NETWORK)
endif

POST_MESSAGE_TEST_FIXTURES_PATH = programs/post-message/tests/fixtures

.PHONY: clean
clean:
	cargo clean
	rm -rf $(POST_MESSAGE_TEST_FIXTURES_PATH) artifacts-mainnet artifacts-testnet artifacts-localnet
	cd anchor && $(MAKE) clean

.PHONY: build
build: solana-install
	@if [ -z "$(SVM)" ]; then \
		echo "Error: SVM must be defined" >&2; \
		exit 1; \
	fi
	solana-verify build -- --features "$(BUILD_FEATURES)"
	solana-verify get-executable-hash target/deploy/wormhole_post_message_shim.so
	solana-verify get-executable-hash target/deploy/wormhole_verify_vaa_shim.so

artifacts-$(NETWORK):
	@if [ ! -d "$@" ]; then \
		$(MAKE) build && \
		mkdir -p "$@" && \
		cp target/deploy/*.so "$@/"; \
	else \
		echo "Directory $@ already exists, skipping"; \
		exit 1; \
	fi

.PHONY: build-artifacts
build-artifacts: artifacts-$(NETWORK)

.PHONY: test
test: solana-install $(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so
	cargo test --lib --features solana
	cargo test --doc --features solana
	cargo test-sbf --features solana

.PHONY: clean-test
clean-test: clean test

.PHONY: clippy
clippy:
	cargo clippy --all-targets --all-features -- -D warnings

$(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so:
	mkdir -p $(POST_MESSAGE_TEST_FIXTURES_PATH)
	solana program dump -u m worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth $(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so

.PHONY: solana-install
solana-install:
	@if ! solana --version | grep -q "$(SOLANA_VERSION)"; then \
		agave-install init $(SOLANA_VERSION); \
	else \
		echo "Solana $(SOLANA_VERSION) is already installed"; \
	fi