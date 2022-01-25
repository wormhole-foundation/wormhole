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

# Migrate

## Mainnet

Migrations on mainnet have to go through governance. Once the guardians sign the
upgrade VAA, the contract can be upgraded by submitting the signed VAA to the
appropriate contract. For example, to upgrade the token bridge on mainnet,
in `wormhole/clients/token_bridge/`:

``` sh
node main.js terra execute_governance_vaa <signed VAA (hex)> --rpc "https://lcd.terra.dev" --chain_id "columbus-5" --mnemonic "..." --token_bridge "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf"
```

## Testnet


For example, to migrate the token bridge to 37262, run in `tools/`:

``` sh
node migrate_testnet.js --code_id 37262 --contract terra1pseddrv0yfsn76u4zxrjmtf45kdlmalswdv39a --mnemonic "..."
```
