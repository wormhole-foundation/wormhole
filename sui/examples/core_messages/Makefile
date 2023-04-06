-include ../../../Makefile.help

.PHONY: artifacts
artifacts: clean

.PHONY: clean
# Clean build artifacts
clean:
	rm -rf build

.PHONY: build
# Build contract
build:
	sui move build

.PHONY: test
# Run tests
test:
	sui move build -d || exit $?
	sui move test -t 1
