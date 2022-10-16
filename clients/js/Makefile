SOURCE_FILES:=$(shell find . -name "*.ts" -not -path "./node_modules/*")

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
	node build/main.js --version > /dev/null
	./run_parse_tests

clean:
	rm -rf build node_modules
