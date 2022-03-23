BRIDGE_ADDRESS=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
# These tests use a different emitter for some reason
export EMITTER_ADDRESS=CiByUvEcx7w2HA4VHcPCBUAFQ73Won9kB36zW9VjirSr

.PHONY: test .FORCE

.FORCE:

target/deploy/bridge.so: .FORCE
	cargo build-bpf

/tmp/ledger:
	mkdir -p $@

test: target/deploy/bridge.so /tmp/ledger
# Kill solana test validator if already running, as we'll need a new one
	@if pgrep solana-test; then pkill solana-test; fi
# Sleep here for a bit so that the process can close the port
	@sleep 1
# Start test validator
# We redirect the logs to test.log in case they're useful
	solana-test-validator \
		--ledger /tmp/ledger \
		--bpf-program ${BRIDGE_ADDRESS} target/deploy/bridge.so \
		--reset --log > test.log 2>&1 &
# Sleep for a few seconds so the validator initialises properly
	@sleep 5
# Run tests. If the tests fail, we kill the validator, but make sure we return
# status 1 (for CI)
	BRIDGE_PAYER=/tmp/ledger/faucet-keypair.json cargo test || (pkill solana-test && exit 1)
# If the tests succeeded, let's kill the validator
	@pkill solana-test

