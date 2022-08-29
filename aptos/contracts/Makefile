-include ../../Makefile.help

.PHONY: artifacts
artifacts: build

## Build contract
.PHONY: build
build:
	aptos move compile --save-metadata

.PHONY: test
test:
	aptos move test
