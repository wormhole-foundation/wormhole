
../target/wasm32-unknown-unknown/release/near_ft.wasm: *.rs  ../Cargo.toml
	(cd ..; cargo build --target wasm32-unknown-unknown --release)

.PHONY: clippy
clippy:
	(cd ..; cargo clippy)
