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
all: build/wormhole-chaind vue validators

.PHONY: validators
validators: build
	# These files change when the genesis file changes, so we need to make
	# sure to copy them over
	touch -m $@
	rm validators/first_validator/genesis/*.json
	cp build/config/gentx/*.json validators/first_validator/genesis/gentx.json
	cp build/config/priv_validator_key.json validators/first_validator/config/
	cp build/config/node_key.json validators/first_validator/config/
	cp build/keyring-test/* validators/first_validator/keyring-test/
	cp build/keyring-test/* validators/guardian_validator/keyring-test/
	# Copy these lines for each new validator
	# We grab the validator's address from the gentx memo that it creates.
	sed -E "s/(persistent_peers = \")[^@]*/\1$$(grep -lR MsgCreateValidator validators/first_validator/genesis | xargs grep -l $(TILT_VALADDRESS) | xargs jq '.body.memo' -r | cut -d@ -f1)/" validators/second_validator/config/config.toml -i
	cp build/keyring-test/* validators/second_validator/keyring-test/

build: config.yml
	touch -m $@
	starport chain init --home $@
	cp client.toml $@/config/client.toml
	touch -m proto

build/wormhole-chaind: cmd/wormhole-chaind/main.go $(GO_FILES) build proto
	go build -o $@ $<

proto:  $(PROTO_FILES)
	starport generate proto-go
	touch proto

vue: $(GO_FILES) proto build
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
test: build
	go test -v ./...

.PHONY: bootstrap
bootstrap:
	npm run bootstrap --prefix testing/js

.PHONY: clean
clean:
	rm -rf build vue
