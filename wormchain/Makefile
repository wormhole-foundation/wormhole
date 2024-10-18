PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo v0.0.1))
BFT_VERSION := $(shell go list -m github.com/cometbft/cometbft | sed 's:.* ::') # grab everything after the space in "github.com/cometbft/cometbft v0.34.7"
COMMIT := $(shell git log -1 --format='%h' 2> /dev/null || echo 'abc123')

build_tags = netgo gcc ledger
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
whitespace += $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=wormchain \
	-X github.com/cosmos/cosmos-sdk/version.ServerName=wormchaind \
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
	-X github.com/cometbft/cometbft/version.TMCoreSemVer=$(BFT_VERSION)
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'

.PHONY: all
all: client

.PHONY: client build/wormchaind install
client: build/wormchaind

build/wormchaind: cmd/wormchaind/main.go cmd/wormchaind/root.go $(GO_FILES)
	@echo building "wormchaind-$(VERSION)"
	go build -mod=readonly $(BUILD_FLAGS) -o $@ ./cmd/wormchaind
	cp "$@" "$@"-"$(VERSION)"

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/wormchaind

proto: $(PROTO_FILES)
	DOCKER_BUILDKIT=1 docker build --target go-export -f Dockerfile.proto -o type=local,dest=. ..

ts-client: $(GO_FILES) proto
	mkdir -p $@
	touch -m $@
	DOCKER_BUILDKIT=1 docker build --target ts-client-export -f Dockerfile.proto -o type=local,dest=. ..

# For now this is a phony target so we just rebuild it each time instead of
# tracking dependencies
.PHONY: ts-sdk
ts-sdk: ts-client
	npm ci --prefix $@
	npm run build --prefix $@

.PHONY: run
run: build/wormchaind
	./$< start --home build  --log_level="debug"

# get npm packages for contracts/tools
contracts-tools-deps: contracts/tools/package-lock.json
	npm ci --prefix=contracts/tools

# get .env and devnet-consts.json for contracts/tools
contracts-devnet-env:
	cd .. && ./scripts/guardian-set-init.sh 1
	cd .. && ./scripts/distribute-devnet-consts.sh

# get wasm artifacts for cosmwasm contracts
contracts-artifacts:
	cd ../cosmwasm && $(MAKE) artifacts
	cp -r ../cosmwasm/artifacts contracts

# get everything needed to
contracts-deploy-setup: contracts-tools-deps contracts-devnet-env contracts-artifacts

# runs the contract deployment script
contracts-deploy-local: contracts-deploy-setup
	npm run deploy-wormchain --prefix=contracts/tools

.PHONY: test
test:
	go test -v ./...

.PHONY: bootstrap
bootstrap:
	npm run bootstrap --prefix testing/js

.PHONY: clean
clean:
	rm -rf build/wormchaind build/wormchaind-* build/**/*.db build/**/*.wal vue/ ts-client/ 
	echo "{\"height\":\"0\",\"round\":0,\"step\":0}" > build/data/priv_validator_state.json

#####################
## INTERCHAINTESTS ##
#####################

# Generate Wormchain Image
local-image: build/wormchaind
	docker build -t wormchain:local -f Dockerfile.ict ..

# Individual Tests ($$ is interpreted as $)
rm-testcache:
	go clean -testcache

ictest-cancel-upgrade: rm-testcache
	cd interchaintest && go test -race -v -run ^TestCancelUpgrade$$ ./...

ictest-chain-start: rm-testcache
	cd interchaintest && go test -race -v -run ^TestChainStart$$ ./...

ictest-malformed-payload: rm-testcache
	cd interchaintest && go test -race -v -run ^TestMalformedPayload$$ ./...

ictest-guardian-set-proposal: rm-testcache
	cd interchaintest && go test -race -v -run ^TestGuardianSetUpdateProposal$$ ./... 

ictest-wormhole-message-proposal: rm-testcache
	cd interchaintest && go test -race -v -run ^TestGovernanceWormholeMessageProposal$$ ./...

ictest-upgrade-failure: rm-testcache
	cd interchaintest && go test -race -v -run ^TestUpgradeFailure$$ ./...

ictest-upgrade: rm-testcache
	cd interchaintest && go test -race -v -run ^TestUpgrade$$ ./...

ictest-wasmd-ante: rm-testcache
	cd interchaintest && go test -race -v -run ^TestWasmdAnteDecorator$$ ./...

ictest-wormchain: rm-testcache
	cd interchaintest && go test -race -v -run ^TestWormchain$$ ./...

.PHONY: run-ictests ictest-cancel-upgrade ictest-chain-start ictest-malformed-payload ictest-proposals ictest-upgrade-failure ictest-upgrade ictest-wasmd-ante ictest-wormchain