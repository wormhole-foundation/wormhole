.PHONY: test

test:
	./runPythonUnitTests.sh

.PHONY: artifacts
artifacts:
	rm -rf $@
	mkdir -p $@
	@echo "Building artifacts for algorand"
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.build -t algorand-builder -o type=local,dest=$@ .
	../scripts/contract-upgrade-governance.sh  -m token_bridge -c algorand -a `cat artifacts/token_approve.teal.hash` -o artifacts
	../scripts/contract-upgrade-governance.sh  -m core -c algorand -a `cat artifacts/core_approve.teal.hash` -o artifacts > artifacts/README.md


