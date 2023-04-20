SOURCE_FILES:=$(shell find . -type d -name node_modules -prune -o -name "*.ts")

# Since the introduction of npm workspaces (#2752), the npm commands need to run
# from the root of the repo.
REPO_ROOT=../..

.PHONY: all
all: build

$(REPO_ROOT)/sdk/js/lib/cjs:
	cd $(REPO_ROOT) && npm -w sdk/js run build

$(REPO_ROOT)/node_modules: $(REPO_ROOT)/package-lock.json
	@touch -m $@
	cd $(REPO_ROOT) && npm ci

build: $(REPO_ROOT)/node_modules $(REPO_ROOT)/sdk/js/lib/cjs $(SOURCE_FILES)
	@mkdir -p build
	cd $(REPO_ROOT)/ && npm -w clients/js run build
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
	rm -rf build
