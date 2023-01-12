CONTRACT_DIRS := wormhole token_bridge

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --target tests .

sui_export:
	DOCKER_BUILDKIT=1 docker build  --progress plain -f Dockerfile.export -t near-export -o type=local,dest=$$HOME/.cargo/bin .
