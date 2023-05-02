-include ../../Makefile.help

VERSION = $(shell grep -Po "version = \"\K[^\"]*" Move.toml | sed "s/\./_/g")

.PHONY: clean
clean:
	rm -rf build

.PHONY: check
## Build contract
check:
	sui move build -d

.PHONY: test
## Run tests
test: check
	grep "public(friend) fun current_version(): V__${VERSION} {" sources/version_control.move
	sui move test -d -t 1
