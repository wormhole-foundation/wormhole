.PHONY: test

test:
	./runPythonUnitTests.sh

.PHONY: artifacts
artifacts:
	mkdir -p $@
	@echo "Building artifacts for algorand"
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.build -t algorand-builder -o type=local,dest=$@ .
	cd $@ && ls | xargs sha256sum > checksums.txt

