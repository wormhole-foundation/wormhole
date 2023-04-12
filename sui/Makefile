BUILD_CONTRACT_DIRS := wormhole token_bridge
TEST_CONTRACT_DIRS := wormhole token_bridge examples/coins examples/core_messages

.PHONY: test
test:
	$(foreach dir,$(TEST_CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile ..
