# Contract Verification

The various EVM explorer sites (Etherscan, Blockscout, etc.) support contract
verification. This essentially entails uploading the source code to the site,
and they verify that the uploaded source code compiles to the same bytecode
that's actually deployed. This enables the explorer to properly parse the
transaction payloads according to the contract ABI.

## Automated Verification (Recommended)

The easiest way to verify contracts is during deployment by setting `VERIFY_ARGS`
in your `.env` file. The deployment scripts will automatically verify all contracts
using Forge's `verify-contract` parameters ([docs](https://getfoundry.sh/forge/reference/verify-contract/)).

### Etherscan-compatible explorers

For Etherscan and other Etherscan-compatible explorers:

```bash
VERIFY_ARGS="--verify --verifier etherscan --etherscan-api-key <YOUR_API_KEY>"
```

### Blockscout explorers

For Blockscout-based explorers:

```bash
VERIFY_ARGS="--verify --verifier blockscout --verifier-url https://explorer.example.com/api"
```

Then run the deployment scripts as described in the README. All contracts will be
verified automatically during deployment.

## Manual Verification

If you need to verify contracts manually (e.g., after an upgrade or if automated
verification failed), you can use the `forge verify-contract` command.

### Contract Structure

Our contracts are structured as a separate proxy and implementation. Both components
need to be verified:
- **Proxy contract**: Only needs verification once (doesn't change)
- **Implementation contract**: Needs verification after each upgrade

### Verifying Core Contracts

#### Wormhole (Proxy)

```bash
forge verify-contract \
  --etherscan-api-key <YOUR_API_KEY> \
  --verifier-url "https://api.etherscan.io/api" \
  <WORMHOLE_ADDRESS> \
  contracts/Wormhole.sol:Wormhole \
  --watch
```

#### Implementation (Core)

```bash
forge verify-contract \
  --etherscan-api-key <YOUR_API_KEY> \
  --verifier-url "https://api.etherscan.io/api" \
  <IMPLEMENTATION_ADDRESS> \
  contracts/Implementation.sol:Implementation \
  --watch
```

### Verifying TokenBridge Contracts

#### TokenBridge (Proxy)

```bash
forge verify-contract \
  --etherscan-api-key <YOUR_API_KEY> \
  --verifier-url "https://api.etherscan.io/api" \
  <TOKEN_BRIDGE_ADDRESS> \
  contracts/bridge/TokenBridge.sol:TokenBridge \
  --watch
```

#### BridgeImplementation

```bash
forge verify-contract \
  --etherscan-api-key <YOUR_API_KEY> \
  --verifier-url "https://api.etherscan.io/api" \
  <TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS> \
  contracts/bridge/BridgeImplementation.sol:BridgeImplementation \
  --watch
```

### Verifying Proxy Configuration

As a final step when first registering a proxy contract, verify that the proxy
points to the correct implementation. This can be done through the explorer's
proxy verification page:

- **Ethereum**: https://etherscan.io/proxyContractChecker
- Other explorers have similar pages (look for "Is this a proxy?" link on the contract page)

## Notes

- Replace `<YOUR_API_KEY>` with your actual API key for the explorer
- Replace `<WORMHOLE_ADDRESS>`, `<IMPLEMENTATION_ADDRESS>`, etc. with actual deployed addresses
- Replace `--verifier-url` with the appropriate API URL for your chain's explorer
- For Blockscout explorers, use `--verifier blockscout` instead of `--verifier etherscan` and include `--verifier-url`
- The `--watch` flag monitors verification status until completion
