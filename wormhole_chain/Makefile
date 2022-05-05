PROTO_FILES=$(shell find proto -name "*.proto")
GO_FILES=$(shell find . -name "*.go")
STARPORT_EXPECTED_VERSION:=v0.19.5
STARPORT_ACTUAL_VERSION:=$(shell starport version | grep "Starport version" | cut -d'	' -f2)

ifneq ("$(STARPORT_ACTUAL_VERSION)", "$(STARPORT_EXPECTED_VERSION)")
  $(error "Expected starport version $(STARPORT_EXPECTED_VERSION) but found $(STARPORT_ACTUAL_VERSION)")
endif

.PHONY: all
all: build/wormhole-chaind vue

build: config.yml
	starport chain init --home $@
	cp client.toml $@/config/client.toml
	touch proto

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

.PHONY: clean
clean:
	rm -rf build vue
