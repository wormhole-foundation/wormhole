SOURCE_FILES:=$(shell find . -type d -name node_modules -prune -o -name "*.ts")

.PHONY: all
all: build

node_modules: package-lock.json
	@touch -m node_modules
	npm ci

.PHONY: dependencies
dependencies: node_modules

build: node_modules $(SOURCE_FILES)
	@mkdir -p build
	npm run build
	@touch -m build

install: build
	@echo Linking binaries
	chmod +x build/main.js
	npm link

.PHONY: test
test: build
# This first invocation will set up the initial config, so that the warning
# doesn't show up in the tests
	npm run check
	npm test
	node build/main.js --version > /dev/null
	./run_parse_tests

docs:
	npm run docs

clean:
	rm -rf build node_modules
