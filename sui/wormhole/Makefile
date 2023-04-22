-include ../../Makefile.help

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
	sui move test -d -t 1
