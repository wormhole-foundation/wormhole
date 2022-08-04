
../target/wasm32-unknown-unknown/release/near_wormhole.wasm: *.rs  ../Cargo.toml
	(cd ..; cargo build --target wasm32-unknown-unknown --release)


.PHONY: clippy
clippy:
	(cd ..; cargo clippy)

.PHONY: clippy-fix
clippy-fix:
	(cd ..; cargo clippy --fix)

