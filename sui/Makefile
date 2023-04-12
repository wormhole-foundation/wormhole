BUILD_CONTRACT_DIRS := wormhole token_bridge
TEST_CONTRACT_DIRS := wormhole token_bridge testing/coin_wrapped_12 examples/core_messages

.PHONY: test
test:
	$(foreach dir,$(TEST_CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile ..
