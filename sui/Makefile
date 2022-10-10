
key_export:
	mkdir -p $$HOME/.sui/sui_config
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.export -t near-export -o type=local,dest=$$HOME/.sui/sui_config .
	ls $$HOME/.sui/sui_config
