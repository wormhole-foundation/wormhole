PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
# Address of the main tilt validator that the others should connect to
TILT_VALADDRESS=wormholevaloper1cyyzpxplxdzkeea7kwsydadg87357qna87hzv8

VERSION := $(shell echo $(shell git describe --tags 2> /dev/null || echo v0.0.1) | sed 's/^v//')
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

.PHONY: client
client: build/wormchaind


build/wormchaind: cmd/wormchaind/main.go $(GO_FILES)
	go build -v $(BUILD_FLAGS) -tags ledger -o $@ $<

proto: $(PROTO_FILES)
	docker build --target go-export -f Dockerfile.proto -o type=local,dest=. ..

vue: $(GO_FILES) proto
	mkdir -p $@
	touch -m $@
	docker build --target vue-export -f Dockerfile.proto -o type=local,dest=. ..

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
