# Deploy

First build the contracts


``` sh
docker build -f Dockerfile.build -o artifacts .
```

Then, for example, to deploy `token_bridge.wasm`, run in the `tools` directory

``` sh
npm ci
node deploy_single.js --network mainnet --artifact ../artifacts/token_bridge.wasm --mnemonic "..."
```

which will print something along the lines of

``` sh
Storing WASM: ../artifacts/token_bridge.wasm (367689 bytes)
Deploy fee:  88446uluna
Code ID:  2435
```
