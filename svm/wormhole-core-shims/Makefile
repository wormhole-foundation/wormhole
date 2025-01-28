POST_MESSAGE_TEST_FIXTURES_PATH = programs/post-message/tests/fixtures

.PHONY: clean
clean:
	cargo clean
	cd anchor && $(MAKE) clean

.PHONY: build
build:
	cargo build-sbf

.PHONY: test
test: programs/post-message/tests/fixtures/core_bridge.so
	cargo test --lib --features solana
	cargo test --doc --features solana
	cargo test-sbf --features solana

.PHONY: clippy
clippy:
	cargo clippy --all-targets --all-features -- -D warnings

$(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so:
	mkdir -p $(POST_MESSAGE_TEST_FIXTURES_PATH)
	solana program dump -u m worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth $(POST_MESSAGE_TEST_FIXTURES_PATH)/core_bridge.so