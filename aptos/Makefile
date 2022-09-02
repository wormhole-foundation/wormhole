build:
	make -C deployer build
	make -C contracts build

test:
	make -C deployer test
	make -C contracts test

test-docker:
	docker build -f Dockerfile --target tests .
