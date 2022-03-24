SOURCE_FILES:=$(shell find contracts -name "*.sol")

.PHONY: dependencies test clean all

all: build

node_modules: package-lock.json
	touch -m node_modules
	npm ci

dependencies: node_modules

build: node_modules ${SOURCE_FILES}
	mkdir -p build
	touch -m build
	npm run build

.env: .env.test
	cp $< $@

test: build .env dependencies
	if pgrep ganache-cli; then pkill ganache-cli; fi
	npx ganache-cli > ganache.log &
	sleep 5
	npx truffle test || (pkill ganache-cli && exit 1)
	pkill ganache-cli

clean:
	rm -f ganache.log
	rm -f .env
	rm -rf node_modules
	rm -rf build
