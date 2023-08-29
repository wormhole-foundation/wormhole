-include ../../Makefile.help

.PHONY: artifacts
artifacts: build

.PHONY: build
## Build contract
build:
	aptos move compile --save-metadata

.PHONY: test
## Run tests
test:
	aptos move test --dev
