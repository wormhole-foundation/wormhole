CONTRACT_DIRS := wormhole

TARGETS := build test

.PHONY: $(TARGETS)
$(TARGETS):
	$(foreach dir,$(CONTRACT_DIRS), make -C $(dir) $@ &&) true

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --target tests .

key_export:
	mkdir -p $$HOME/.sui/sui_config
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.export -t near-export -o type=local,dest=$$HOME/.sui/sui_config .
	sed -i -e "s-/root-$$HOME-g" $$HOME/.sui/sui_config/*
	ls $$HOME/.sui/sui_config
