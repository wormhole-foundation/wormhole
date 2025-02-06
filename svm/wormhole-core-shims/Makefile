SOLANA_VERSION := 2.1.11

POST_MESSAGE_TEST_FIXTURES_PATH = programs/post-message/tests/fixtures

.PHONY: clean
clean:
	cargo clean
	rm -rf $(POST_MESSAGE_TEST_FIXTURES_PATH)
	cd anchor && $(MAKE) clean

.PHONY: build
build: solana-install
	cargo build-sbf --features $(SVM)

.PHONY: test
test: solana-install $(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so
	cargo test --lib --features solana
	cargo test --doc --features solana
	cargo test-sbf --features solana

.PHONY: clippy
clippy:
	cargo clippy --all-targets --all-features -- -D warnings

$(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so:
	mkdir -p $(POST_MESSAGE_TEST_FIXTURES_PATH)
	solana program dump -u m worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth $(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so

.PHONY: solana-install
solana-install:
	agave-install init $(SOLANA_VERSION)