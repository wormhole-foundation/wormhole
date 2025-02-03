INTERFACES := wormhole_post_message_shim wormhole_verify_vaa_shim
EXAMPLES := wormhole_integrator_example wormhole_vaa_verification_comparison

.PHONY: clean
clean:
	cargo clean

.PHONY: test
test: build-examples
# Force rebuild of the programs.
	cd .. && $(MAKE) build
	cp -R ../target/deploy/*.so target/deploy
# Copy IDLs and Typescript types to target directory.
	@for interface in $(INTERFACES); do \
		cp idls/$$interface.json target/idl; \
		cp idls/$$interface.ts target/types; \
	done
# Finally run Typescript tests.
	anchor test --skip-build

.PHONY: build-examples
build-examples: build-interfaces
	@for example in $(EXAMPLES); do \
		anchor build -p $$example; \
	done

.PHONY: build-interfaces
build-interfaces:
	@for interface in $(INTERFACES); do \
		anchor idl build -p $$interface -o idls/$$interface.json; \
		anchor idl type idls/$$interface.json > idls/$$interface.ts; \
	done

.PHONY: clean-test
clean-test: clean test

.PHONY: check-idl
check-idl: build-interfaces
	git diff --exit-code idls/