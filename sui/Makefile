TEST_CONTRACT_DIRS := wormhole token_bridge examples/coins examples/core_messages
CLEAN_CONTRACT_DIRS := wormhole token_bridge examples/coins examples/core_messages

.PHONY: clean
clean:
	$(foreach dir,$(TEST_CONTRACT_DIRS), make -C $(dir) $@ &&) true

.PHONY: test
test:
	$(foreach dir,$(TEST_CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build --progress plain  -f ../Dockerfile.cli -t cli-gen ..
	DOCKER_BUILDKIT=1 docker build --build-arg num_guardians=1 --progress plain -f ../Dockerfile.const -t const-gen ..
	DOCKER_BUILDKIT=1 docker build -f Dockerfile ..
