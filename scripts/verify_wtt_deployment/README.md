# Verify WTT deployment

Can be used to check that the on-chain configuration of a new EVM WTT deployment is correct.
Useful when reviewing token bridge governance documents.

It checks:

- Proxy implementation address
- Wormhole chain ID
- EVM chain ID
- Governance config (chainId & address)
- WETH address
- Core bridge address
- Finality
- All chains token bridge registrations (source: TS SDK)

Always upgrade `@wormhole-foundation/sdk` to latest to make sure new chains are checked but deprecated chains are not checked. 

```bash
bun add @wormhole-foundation/sdk@latest
```

## Usage

```bash
bun run verifyTb.ts <RPC> <TOKEN_BRIDGE_ADDRESS> <TOKEN_BRIDGE_IMPLEMENTATION> <CORE_BRIDGE_ADDRESS> <WETH> <WORMHOLE_CHAIN_ID> <EVM_CHAIN_ID>
```

### Example

```bash
bun run verifyTb.ts https://rpc-mainnet.monadinfra.com/rpc/Ksg4ryy9YonTCqsLART0VjfT486B6sW4 0x0B2719cdA2F10595369e6673ceA3Ee2EDFa13BA7 0x32b3b68e9f053E724Da0A9e57F062BFaE6695350 0x194B123c5E96B9b2E49763619985790Dc241CAC0 0x3bd359C1119dA7Da1D913D1C4D2B7c461115433A 48 143
```

## Rate limits

Increase `SLEEP_BETWEEN_RPC_CALLS_IN_MS` if working with a rate limited RPC.
