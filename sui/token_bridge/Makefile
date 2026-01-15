-include ../../Makefile.help

VERSION = $(shell sed -n 's/^version = "\([^"]*\)".*/\1/p' Move.toml | tr '.' '_')

.PHONY: clean
clean:
	rm -rf build

.PHONY: check
## Build contract
check:
	sui move build

.PHONY: test
## Run tests
test: check
	grep "public(friend) fun current_version(): V__${VERSION} {" sources/version_control.move
	sui move test -t 1
