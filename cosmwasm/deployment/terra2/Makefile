bridge_SOURCE=wormhole
token_bridge_SOURCE=token_bridge_terra


-include ../Makefile.help

.PHONY: artifacts
## Build contracts and copy to terra2
artifacts:
	cd ../.. && $(MAKE) artifacts
	cp -r ../../artifacts .


payer-$(NETWORK).json:
	$(error Missing private key in payer-$(NETWORK).json)

.PHONY: deploy/bridge
## Deploy core bridge
deploy/bridge: bridge-code-id-$(NETWORK).txt

.PHONY: deploy/token_bridge
## Deploy token bridge
deploy/token_bridge: token_bridge-code-id-$(NETWORK).txt

%-code-id-$(NETWORK).txt: check-network tools/node_modules payer-$(NETWORK).json artifacts
	@echo "Deploying artifacts/$($*_SOURCE).wasm on $(NETWORK)"
	@node tools/deploy_single.js \
		--network $(NETWORK) \
		--artifact artifacts/$($*_SOURCE).wasm \
		--mnemonic "$$(cat payer-$(NETWORK).json)" \
		| grep -i "code id" | sed s/[^0-9]//g \
		> $@
	@echo "Deployed at code id $$(cat $@) (stored in $@)"

tools/node_modules: tools/package-lock.json
	cd tools && npm ci

test/node_modules: test/package-lock.json
	cd test && npm ci

.PHONY: test
## Run unit and integration tests
test: artifacts test/node_modules
	@if pgrep terrad; then echo "Error: terrad already running. Stop it before running tests"; exit 1; fi
	cd devnet && DOCKER_BUILDKIT=1 docker-compose up --detach
	sleep 10
	cd test && npm run test || (cd ../devnet && docker-compose down && exit 1)
	cd devnet && docker-compose down

.PHONY: clean
clean:
	rm -rf artifacts
	rm -rf tools/node_modules
	rm -rf test/node_modules
