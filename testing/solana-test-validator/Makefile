.PHONY: clean all help .FORCE

-include ../Makefile.help

.FORCE:

node_modules:
	yarn

artifacts: node_modules
	cd ../../solana && DOCKER_BUILDKIT=1 docker build -f Dockerfile --build-arg BRIDGE_ADDRESS=agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC -o ../testing/solana-test-validator/artifacts .

.PHONY: test
test: artifacts
	@echo "Running integration tests"
	yarn run sdk-tests

.PHONY: clean
clean:
	rm -rf artifacts node_modules validator.log .test

