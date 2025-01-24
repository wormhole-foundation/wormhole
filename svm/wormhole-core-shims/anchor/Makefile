.PHONY: clean
clean:
	cargo clean

.PHONY: test
test:
# Build example programs and IDLs.
	anchor build
# Force rebuild of the programs.
	cd .. && $(MAKE) build
	cp -R ../target/deploy/*.so target/deploy
# Finally run Typescript tests.
	anchor test --skip-build