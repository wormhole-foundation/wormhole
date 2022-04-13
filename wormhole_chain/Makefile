PROTO_FILES:=$(shell find proto -name "*.proto")
GO_FILES:=$(shell find . -name "*.go")

.PHONY: all
all: build/wormhole-chaind

build: config.yml
	starport chain init --home $@
	cp client.toml $@/config/client.toml
	touch proto

build/wormhole-chaind: cmd/wormhole-chaind/main.go $(GO_FILES) build proto
	go build -o $@ $<

proto:  $(PROTO_FILES)
	starport generate proto-go
	touch proto

.PHONY: run
run: build/wormhole-chaind
	./$< start --home build

.PHONY: test
test: build
	go test -v ./...

.PHONY: clean
clean:
	rm -rf build
