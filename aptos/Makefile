CONTRACT_DIRS := deployer wormhole token_bridge examples coin

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --target tests .
