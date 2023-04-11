-include ../../Makefile.help

.PHONY: artifacts
artifacts: build

.PHONY: build
## Build contract
build:
	sui move build

.PHONY: test
## Run tests
test:
	sui move build -d || exit $?
	sui move test -d -t 1
