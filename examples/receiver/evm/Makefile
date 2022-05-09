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

.env: .env.devnet
	cp $< $@

test: build .env dependencies
	@if pgrep ganache-cli; then echo "Error: ganache-cli already running. Stop it before running tests"; exit 1; fi
	npx ganache-cli -e 10000 --deterministic --time="1970-01-01T00:00:00+00:00" > ganache.log &
	sleep 5
	npm test || (pkill ganache-cli && exit 1)
	pkill ganache-cli || true

clean:
	rm -rf ganache.log .env node_modules build
