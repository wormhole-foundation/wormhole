PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo v0.0.1))
COMMIT := $(shell git log -1 --format='%h' 2> /dev/null || echo 'abc123')

ldflags = \
    -X github.com/cosmos/cosmos-sdk/version.Name=wormchain\
	-X github.com/cosmos/cosmos-sdk/version.ServerName=wormchaind\
	-X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
	-X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
	-X "github.com/cosmos/cosmos-sdk/version.BuildTags=ledger"
BUILD_FLAGS := -ldflags '$(ldflags)'

.PHONY: all
all: client

.PHONY: client build/wormchaind
client: build/wormchaind

build/wormchaind: cmd/wormchaind/main.go $(GO_FILES)
	@echo building "wormchaind-$(VERSION)"
	go build -v $(BUILD_FLAGS) -tags ledger -o $@ $<
	cp "$@" "$@"-"$(VERSION)"

vue: $(GO_FILES)
	mkdir -p $@
	touch -m $@
	DOCKER_BUILDKIT=1 docker build --target vue-export -f Dockerfile.proto -o type=local,dest=. ..

# For now this is a phony target so we just rebuild it each time instead of
# tracking dependencies
.PHONY: ts-sdk
ts-sdk: vue
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
	rm -rf build/wormchaind build/wormchaind-* build/**/*.db build/**/*.wal vue
	echo "{\"height\":\"0\",\"round\":0,\"step\":0}" > build/data/priv_validator_state.json

# New Proto Targets (copied from Juno Network)

DOCKER := $(shell which docker)
protoVer=0.13.1
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
protoImage=$(DOCKER) run --rm -v $(CURDIR):/workspace --workdir /workspace $(protoImageName)

.PHONY: proto-all proto-gen proto-format proto-lint
proto-all: proto-format proto-lint proto-gen

proto-gen:
	@echo "Generating Protobuf files"
	@$(protoImage) sh ./scripts/protocgen.sh

proto-format:
	@$(protoImage) find ./ -name "*.proto" -exec clang-format -i {} \;

proto-lint:
	@$(protoImage) buf lint --error-format=json
