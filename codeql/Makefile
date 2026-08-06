# The pack lives inside the Wormhole monorepo, so the source root defaults to the parent directory.
WORMHOLE_REPO ?= ..
NODE_DB       ?= ./wormhole-node-go-db
SDK_DB        ?= ./wormhole-sdk-go-db
SUITE         ?= suites/wormhole-go.qls
NODE_OUTPUT   ?= wormhole-node-go-lints.sarif
SDK_OUTPUT    ?= wormhole-sdk-go-lints.sarif

.PHONY: all
all: analyze

.PHONY: create-db create-node-db create-sdk-db
create-db: create-node-db create-sdk-db

create-node-db:
	codeql database create $(NODE_DB) \
		--language=go \
		--source-root=$(WORMHOLE_REPO) \
		--command='/bin/sh -c "cd node && go build -a ./..."'

create-sdk-db:
	codeql database create $(SDK_DB) \
		--language=go \
		--source-root=$(WORMHOLE_REPO) \
		--command='/bin/sh -c "cd sdk && go build -a ./..."'

.PHONY: create-db-overwrite create-node-db-overwrite create-sdk-db-overwrite
create-db-overwrite: create-node-db-overwrite create-sdk-db-overwrite

create-node-db-overwrite:
	codeql database create $(NODE_DB) \
		--language=go \
		--source-root=$(WORMHOLE_REPO) \
		--command='/bin/sh -c "cd node && go build -a ./..."' \
		--overwrite

create-sdk-db-overwrite:
	codeql database create $(SDK_DB) \
		--language=go \
		--source-root=$(WORMHOLE_REPO) \
		--command='/bin/sh -c "cd sdk && go build -a ./..."' \
		--overwrite

.PHONY: analyze analyze-node analyze-sdk
analyze: analyze-node analyze-sdk

analyze-node:
	codeql database analyze $(NODE_DB) \
		$(SUITE) \
		--format=sarif-latest \
		--output=$(NODE_OUTPUT)

analyze-sdk:
	codeql database analyze $(SDK_DB) \
		$(SUITE) \
		--format=sarif-latest \
		--output=$(SDK_OUTPUT)

.PHONY: scan
scan: create-db-overwrite
	$(MAKE) analyze

.PHONY: compile
compile:
	codeql query compile src

.PHONY: test
test:
	codeql test run test

.PHONY: clean
clean:
	rm -rf $(NODE_DB) $(SDK_DB) $(NODE_OUTPUT) $(SDK_OUTPUT)
