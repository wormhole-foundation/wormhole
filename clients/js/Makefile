SOURCE_FILES:=$(shell find . -name "*.ts" -not -path "./node_modules/*")

.PHONY: all
all: build

node_modules: package-lock.json
	touch -m node_modules
	npm ci

.PHONY: dependencies
dependencies: node_modules

.PHONY: build
build: build/main.js

build/%.js: node_modules %.ts
	npm run build

install: build/main.js
	chmod +x $<
	npm link

clean:
	rm -rf build node_modules
