.PHONY: install
## Install guardiand binary
build:
	go build -o wormhole-revive .

run:
	./wormhole-revive -config revive.toml ../node/...
	./wormhole-revive -config revive.toml ../sdk/...
