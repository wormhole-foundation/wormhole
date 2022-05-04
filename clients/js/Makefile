SOURCE_FILES:=$(shell find . -name "*.ts" -not -path "./node_modules/*")

.PHONY: all
all: build

package-lock.json: package.json
	npm install

node_modules: package-lock.json
	@touch -m node_modules
	npm ci

.PHONY: dependencies
dependencies: node_modules

build: node_modules $(SOURCE_FILES)
	@mkdir -p build
	@touch -m build
	npm run build

install: build
	@echo Linking binaries
	chmod +x build/clients/js/main.js
	npm link

clean:
	rm -rf build node_modules
