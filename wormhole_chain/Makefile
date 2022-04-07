.PHONY: all run test clean

all: build

build: config.yml
	starport chain init --home $@
	cp client.toml $@/config/client.toml

run: build
	go run cmd/wormhole-chaind/main.go start --home build

test:
	go test -v ./...

clean:
	rm -rf build
