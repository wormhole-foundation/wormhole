.PHONY: lint
lint: 
	golangci-lint run -c ../.golangci.yml ./...

.PHONY: test
test: 
	go test -v ./...
	timeout 10s go test -fuzz=FuzzMessagePublicationUnmarshalBinary -fuzztime=5s ./pkg/common || true
