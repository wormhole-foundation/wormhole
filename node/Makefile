.PHONY: lint
lint: 
	golangci-lint run -c ../.golangci.yml ./...

.PHONY: test
test: 
	go test -v ./...
	timeout 10s go test -fuzz=FuzzMessagePublicationUnmarshalBinary -fuzztime=5s ./pkg/common || true

.PHONY: test-coverage
test-coverage:
	@echo "Note: Use 'make test-coverage' from repo root to test both node and sdk"
	@cd .. && make test-coverage

.PHONY: check-coverage
check-coverage:
	@echo "Note: Use 'make check-coverage' from repo root to check both node and sdk"
	@cd .. && make check-coverage
