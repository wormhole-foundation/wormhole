build:
	make -C deployer build
	make -C contracts build

test:
	make -C deployer test
	make -C contracts test

test-docker:
	DOCKER_BUILDKIT=1 docker build -f Dockerfile --target tests .
