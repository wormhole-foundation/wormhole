PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
IGNITE_EXPECTED_VERSION:=v0.23.0
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo v0.0.1) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%h' 2> /dev/null || echo 'abc123')

ldflags = \
    -X github.com/cosmos/cosmos-sdk/version.Name=wormchain\
	-X github.com/cosmos/cosmos-sdk/version.ServerName=wormchaind\
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) 
BUILD_FLAGS := -ldflags '$(ldflags)'

.PHONY: all
all: client vue validators

.PHONY: client
client: build/wormchaind

.PHONY: validators
validators:
	# These files change when the genesis file changes, so we need to make
	# sure to copy them over
	touch -m $@
	./build/wormchaind --home build/ tendermint unsafe-reset-all
	./build/wormchaind --home build/ gentx tiltGuardian "0uworm" --chain-id=wormchain --min-self-delegation="0" --keyring-dir=keyring-test

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

build/wormchaind: cmd/wormchaind/main.go $(GO_FILES)
	go build -v $(BUILD_FLAGS) -o $@ $<

check-ignite:
	@ if [ "$(shell ignite version | awk '/Ignite CLI version:/ { print $$4 }')" != "$(IGNITE_EXPECTED_VERSION)" ] ; then echo "Expected ignite version $(IGNITE_EXPECTED_VERSION)" && exit 1 ; fi

proto: check-ignite $(PROTO_FILES)
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
run: build/wormchaind
	./$< start --home build  --log_level="debug"

.PHONY: test
test:
	go test -v ./...

.PHONY: bootstrap
bootstrap:
	npm run bootstrap --prefix testing/js

.PHONY: clean
clean:
	rm -rf build/wormchaind build/**/*.db build/**/*.wal vue
	echo "{\"height\":\"0\",\"round\":0,\"step\":0}" > build/data/priv_validator_state.json
