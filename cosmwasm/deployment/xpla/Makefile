-include ../Makefile.help

.PHONY: artifacts
## Build contracts and copy to xpla
artifacts:
	cd ../.. && $(MAKE) artifacts
	cp -r ../../artifacts .

tools/node_modules: tools/package-lock.json
	cd tools && npm ci

.PHONY: clean
clean:
	rm -rf artifacts
	rm -rf tools/node_modules
