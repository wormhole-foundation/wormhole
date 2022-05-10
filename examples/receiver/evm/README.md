# Wormhole Receiver

## Deploy

```bash
npm ci
cp .env.mainnet .env
MNEMONIC="[YOUR_KEY_HERE]" npm run migrate -- --network [NETWORK_KEY_FROM_TRUFFLE_CONFIG]
MNEMONIC="[YOUR_KEY_HERE]" npm run submit-guardian-sets -- --network [NETWORK_KEY_FROM_TRUFFLE_CONFIG]
```
