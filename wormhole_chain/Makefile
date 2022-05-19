PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
STARPORT_EXPECTED_VERSION:=v0.19.5
STARPORT_ACTUAL_VERSION:=$(shell starport version | grep "Starport version" | cut -d'	' -f2)
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

ifneq ("$(STARPORT_ACTUAL_VERSION)", "$(STARPORT_EXPECTED_VERSION)")
  $(error "Expected starport version $(STARPORT_EXPECTED_VERSION) but found $(STARPORT_ACTUAL_VERSION)")
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
	rm build/config/gentx/gentx-c3f474217c930af3a4e998c4e52a57cee188ff43.json
	./build/wormhole-chaind --home build/ gentx tiltGuardian "0uworm" --chain-id wormholechain --min-self-delegation "0"
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
	starport generate proto-go
	touch proto

vue: $(GO_FILES) proto
	mkdir -p $@
	touch -m $@
	starport generate vuex --proto-all-modules

# For now this is a phony target so we just rebuild it each time instead of
# tracking dependencies
.PHONY: ts-sdk
ts-sdk: vue
	npm ci --prefix $@
	npm run build --prefix $@

.PHONY: run
run: build/wormhole-chaind
	./$< start --home build

.PHONY: test
test:
	go test -v ./...

.PHONY: bootstrap
bootstrap:
	npm run bootstrap --prefix testing/js

.PHONY: clean
clean:
	rm -rf build/wormhole-chaind vue
