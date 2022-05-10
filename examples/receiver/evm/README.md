# Wormhole Receiver

This contract can be used to receive Wormhole messages on chains that don't have a [core bridge contract](https://docs.wormholenetwork.com/wormhole/contracts#core-bridge) deployed.

## Deploy

```bash
npm ci
cp .env.mainnet .env
MNEMONIC="[YOUR_KEY_HERE]" npm run migrate -- --network [NETWORK_KEY_FROM_TRUFFLE_CONFIG]
MNEMONIC="[YOUR_KEY_HERE]" npm run submit-guardian-sets -- --network [NETWORK_KEY_FROM_TRUFFLE_CONFIG]
```
