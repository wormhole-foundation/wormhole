.PHONY: all clean test check

all: check

.PHONY: clean
clean:
	rm -rf build

.PHONY: check
check:
	sui move build -d

.PHONY: test
test: check
	sui move test -d
