PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
IGNITE_EXPECTED_VERSION:=v0.23.0
IGNITE_ACTUAL_VERSION:=$(shell ignite version | awk '/Ignite CLI version:/ { print $$4 }')
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

ifneq ("$(IGNITE_ACTUAL_VERSION)", "$(IGNITE_EXPECTED_VERSION)")
  $(error "Expected ignite version $(IGNITE_EXPECTED_VERSION) but found $(IGNITE_ACTUAL_VERSION)")
endif

.PHONY: all
all: client vue validators

.PHONY: client
client: build/wormhole-chaind

.PHONY: validators
validators:
	# These files change when the genesis file changes, so we need to make
	# sure to copy them over
	touch -m $@
	./build/wormhole-chaind --home build/ tendermint unsafe-reset-all
	./build/wormhole-chaind --home build/ gentx tiltGuardian "0uworm" --chain-id=wormholechain --min-self-delegation="0" --keyring-dir=keyring-test

	# Copy config to validators/first_validator
	cp build/config/priv_validator_key.json validators/first_validator/config/
	cp build/config/node_key.json validators/first_validator/config/
	mkdir -p validators/first_validator/keyring-test
	cp build/keyring-test/* validators/first_validator/keyring-test/

	# Copy these lines for each new validator
	# We grab the validator's address from the gentx memo that it creates.
	sed -E "s/(persistent_peers = \")[^@]*/\1$$(grep -lR MsgCreateValidator build/config/gentx | xargs grep -l $(TILT_VALADDRESS) | xargs jq '.body.memo' -r | cut -d@ -f1)/" validators/second_validator/config/config.toml -i
	mkdir -p validators/second_validator/keyring-test
	cp build/keyring-test/* validators/second_validator/keyring-test/

build/wormhole-chaind: cmd/wormhole-chaind/main.go $(GO_FILES) proto
	go build -o $@ $<

proto:  $(PROTO_FILES)
	ignite generate proto-go
	touch proto

vue: $(GO_FILES) proto
	mkdir -p $@
	touch -m $@
	NODE_OPTIONS="" ignite generate vuex --proto-all-modules

# For now this is a phony target so we just rebuild it each time instead of
# tracking dependencies
.PHONY: ts-sdk
ts-sdk: vue
	npm ci --prefix $@
	npm run build --prefix $@

.PHONY: run
run: build/wormhole-chaind
	./$< start --home build  --log_level="debug"

.PHONY: test
test:
	go test -v ./...

.PHONY: bootstrap
bootstrap:
	npm run bootstrap --prefix testing/js

.PHONY: clean
clean:
	rm -rf build/wormhole-chaind build/**/*.db build/**/*.wal vue
	echo "{\"height\":\"0\",\"round\":0,\"step\":0}" > build/data/priv_validator_state.json
