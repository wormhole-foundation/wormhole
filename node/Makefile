.PHONY: lint
lint: 
	golangci-lint run -c ../.golangci.yml ./...

.PHONY: test
test: 
	go test -v ./...
	timeout 10s go test -fuzz=FuzzMessagePublicationUnmarshalBinary -fuzztime=5s ./pkg/common || true

.PHONY: coverage-test
coverage-test:
	@echo "Note: Use 'make coverage-test' from repo root to test both node and sdk"
	@cd .. && $(MAKE) coverage-test

.PHONY: coverage
coverage:
	@echo "Note: Use 'make coverage' from repo root to check both node and sdk"
	@cd .. && $(MAKE) coverage

# Backward-compatible aliases
.PHONY: test-coverage check-coverage
test-coverage: coverage-test
check-coverage: coverage
