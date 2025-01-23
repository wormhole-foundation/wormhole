.PHONY: clean
clean:
	cargo clean

.PHONY: test
test: ../target/deploy/*.so
	anchor build
	cp -R ../target/deploy/*.so target/deploy
	anchor test --skip-build

../target/deploy/*.so:
	cd .. && $(MAKE) build