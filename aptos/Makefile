CONTRACT_DIRS := deployer wormhole token_bridge nft_bridge examples coin

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build --progress plain  -f Dockerfile --target tests .
