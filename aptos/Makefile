CONTRACT_DIRS := deployer wormhole token_bridge nft_bridge examples coin

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build --progress plain  -f ../Dockerfile.cli -t cli-gen ..
	DOCKER_BUILDKIT=1 docker build --build-arg num_guardians=1 --progress plain  -f ../Dockerfile.const -t const-gen ..
	DOCKER_BUILDKIT=1 docker build --progress plain  -f Dockerfile --target tests .
