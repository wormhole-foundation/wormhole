.PHONY: test
test: build node_modules
	npx ts-node test/test.ts

all: node_modules build nearcore

.PHONY: artifacts
artifacts:
	rm -rf $@
	mkdir -p $@
	@echo "Building artifacts for near"
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.build -t near-builder -o type=local,dest=$@ .
	cd $@ && ls | xargs sha256sum > checksums.txt

artifacts/GOVERNANCE.md: artifacts
	../scripts/contract-upgrade-governance.sh --force -m token_bridge -c near -a `sha256sum artifacts/near_token_bridge.wasm | cut -d' ' -f1` -o artifacts
	../scripts/contract-upgrade-governance.sh --force -m core         -c near -a `sha256sum artifacts/near_wormhole.wasm | cut -d' ' -f1` -o artifacts > $@

.PHONY: governance
governance: artifacts/GOVERNANCE.md

build:    contracts/ft/target/wasm32-unknown-unknown/release/near_ft.wasm \
          contracts/mock-bridge-integration/target/wasm32-unknown-unknown/release/near_mock_bridge_integration.wasm \
          contracts/mock-bridge-token/target/wasm32-unknown-unknown/release/near_mock_bridge_token.wasm \
          contracts/nft-bridge/target/wasm32-unknown-unknown/release/near_nft_bridge.wasm \
          contracts/nft-wrapped/target/wasm32-unknown-unknown/release/near_nft.wasm \
          contracts/token-bridge/target/wasm32-unknown-unknown/release/near_token_bridge.wasm \
          contracts/wormhole/target/wasm32-unknown-unknown/release/near_wormhole.wasm

contracts/ft/target/wasm32-unknown-unknown/release/near_ft.wasm: contracts/ft/src/*.rs  contracts/ft/Cargo.toml
	cd contracts/ft; cargo build --target wasm32-unknown-unknown --release

contracts/mock-bridge-integration/target/wasm32-unknown-unknown/release/near_mock_bridge_integration.wasm: \
                contracts/mock-bridge-integration/src/*.rs  \
                contracts/mock-bridge-integration/Cargo.toml \
                contracts/mock-bridge-token/target/wasm32-unknown-unknown/release/near_mock_bridge_token.wasm \
                contracts/nft-wrapped/target/wasm32-unknown-unknown/release/near_nft.wasm

	cd contracts/mock-bridge-integration; cargo build --target wasm32-unknown-unknown --release

contracts/mock-bridge-token/target/wasm32-unknown-unknown/release/near_mock_bridge_token.wasm : contracts/mock-bridge-token/src/*.rs  contracts/mock-bridge-token/Cargo.toml
	cd contracts/mock-bridge-token; cargo build --target wasm32-unknown-unknown --release

contracts/nft-bridge/target/wasm32-unknown-unknown/release/near_nft_bridge.wasm: contracts/nft-bridge/src/*.rs  contracts/nft-bridge/Cargo.toml  contracts/nft-wrapped/target/wasm32-unknown-unknown/release/near_nft.wasm
	cd contracts/nft-bridge; cargo build --target wasm32-unknown-unknown --release

contracts/nft-wrapped/target/wasm32-unknown-unknown/release/near_nft.wasm: contracts/nft-wrapped/src/*.rs  contracts/nft-wrapped/Cargo.toml
	cd contracts/nft-wrapped; cargo build --target wasm32-unknown-unknown --release

contracts/token-bridge/target/wasm32-unknown-unknown/release/near_token_bridge.wasm: contracts/token-bridge/src/*.rs  contracts/token-bridge/Cargo.toml contracts/ft/target/wasm32-unknown-unknown/release/near_ft.wasm
	cd contracts/token-bridge; cargo build --target wasm32-unknown-unknown --release

contracts/wormhole/target/wasm32-unknown-unknown/release/near_wormhole.wasm: contracts/wormhole/src/*.rs contracts/wormhole/Cargo.toml
	cd contracts/wormhole; cargo build --target wasm32-unknown-unknown --release

package-lock.json: package.json
	npm install

node_modules: package-lock.json
	touch -m node_modules
	npm ci

nearcore:
	mkdir $@ && \
	cd $@ && \
	git init && \
	git remote add origin https://github.com/near/nearcore && \
	git fetch --depth 1 origin c6eb78ab11d0fb7c7bb9dfa6d712aba449a0140b && \
	git checkout FETCH_HEAD
	cd $@ && make sandbox-release

run: nearcore
	-killall -q Python
	./start_node.sh


.PHONY: clean
clean:
	rm -rf nearcore contracts/*/target


.PHONY: reset-force
reset-force: build
	-echo cleaning and restarting state
	-minikube kubectl -- delete pod guardian-0 --force &
	-minikube kubectl -- delete pod algorand-0 --force &
	-minikube kubectl -- delete pod near-0 --force &
	-echo done

.PHONY: reset
reset: build
	-echo cleaning and restarting state
	-minikube kubectl -- delete pod guardian-0 &
	-minikube kubectl -- delete pod algorand-0 &
	-minikube kubectl -- delete pod near-0 &
	-echo done

.PHONY: cycle
cycle: reset
	(cd ../sdk/js; npm run build)
	npm ci
	ts-node test/sdk.ts

.PHONY: testnet
testnet: build
	cp contracts/*/target/wasm32-unknown-unknown/release/*.wasm .
	ts-node devnet_deploy.ts
