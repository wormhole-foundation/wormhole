.PHONY: clean all help .FORCE

-include ../Makefile.help

.FORCE:

node_modules:
	yarn

artifacts:
	cd ../../solana && \
		DOCKER_BUILDKIT=1 docker build \
			-f Dockerfile \
			--build-arg BRIDGE_ADDRESS=agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC \
			-o ../testing/solana-test-validator/artifacts .

artifacts-main:
	git clone \
		--depth 1 \
		--branch main \
		--filter=blob:none \
		https://github.com/wormhole-foundation/wormhole \
		wormhole-main
	cd wormhole-main/solana && \
		DOCKER_BUILDKIT=1 docker build \
			-f Dockerfile \
			--build-arg BRIDGE_ADDRESS=agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC \
			-o ../../artifacts-main .
	rm -rf wormhole-main

.PHONY: test
test: node_modules artifacts-main artifacts
	@echo "Running integration tests"
	yarn run sdk-tests

.PHONY: clean
clean:
	rm -rf artifacts artifacts-main wormhole-main node_modules validator.log .test

