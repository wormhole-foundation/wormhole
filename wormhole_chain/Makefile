.PHONY: all run

all: build

build: config.yml
	starport chain init --home $@
	cp client.toml $@/config/client.toml

run: build
	go run cmd/wormhole-chaind/main.go start --home build

clean:
	rm -rf build
