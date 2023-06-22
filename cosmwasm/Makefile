SOURCE_FILES=$(shell find . -name "*.rs" -or -name "*.lock" -or -name "*.toml" | grep -v target)
PACKAGES=$(shell find . -name "Cargo.toml" | grep -E 'packages|contracts' | xargs cat | grep "name *=" | cut -d' ' -f3 | sed s/\"//g | sed s/-/_/g)
WASMS=$(patsubst %, artifacts/%.wasm, $(PACKAGES))

-include ../Makefile.help

.PHONY: artifacts
## Build contracts.
artifacts: artifacts/checksums.txt

$(WASMS) artifacts/checksums.txt: $(SOURCE_FILES)
	DOCKER_BUILDKIT=1 docker build --target artifacts -o artifacts -f ./Dockerfile ../

tools/node_modules: tools/package-lock.json
	cd tools && npm ci

.PHONY: test
## Run unit tests
test:
	cargo test --workspace --locked


.PHONY: clean
clean:
	rm -f $(WASMS)
	rm -f artifacts/checksums.txt
