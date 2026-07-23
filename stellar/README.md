# Wormhole on Stellar

This directory contains the implementation of the Wormhole Core contract for Stellar (Wormhole
`Chain ID = 61`). It verifies Wormhole VAAs, supports the standard Core governance actions, and
exposes a small client crate for other Soroban contracts to call.

## Repository Structure

```
contracts/
├── wormhole-soroban-client/     # Public API crate (for external integrations)
│   └── src/
│       ├── lib.rs               # Re-exports and WormholeCoreInterface trait
│       ├── types.rs             # VAA, Signature, GuardianSetInfo, etc.
│       ├── bytes_reader.rs      # Binary parsing utilities
│       ├── error.rs             # 46 error variants
│       └── constants.rs         # Protocol constants
│
└── wormhole-contract/           # Implementation crate
    └── src/
        ├── lib.rs               # Contract entry point
        ├── initialize.rs        # One-time setup
        ├── storage.rs           # StorageKey enum
        ├── vaa.rs               # Signature verification
        ├── message.rs           # Cross-chain message posting
        ├── utils/mod.rs         # Crypto utilities
        └── governance/          # Governance actions
            ├── mod.rs
            ├── action.rs        # GovernanceAction trait
            ├── contract_upgrade.rs   # Action 1
            ├── guardian_set.rs       # Action 2
            ├── set_message_fee.rs    # Action 3
            └── transfer_fees.rs      # Action 4
```

### Why Two Crates?

- **wormhole-soroban-client**: Lightweight public API. External contracts depend only on this,
  resulting in smaller WASM binaries.
- **wormhole-contract**: Full implementation with storage access and business logic.

## Quick start

### Prerequisites

Follow Stellar
[setup](https://developers.stellar.org/docs/build/smart-contracts/getting-started/setup) guide

### Build

``` bash
cd stellar
stellar contract build --optimize
```

## Contract Interface

See [ARCHITECTURE.md](ARCHITECTURE.md) for the complete contract interface, core types, and protocol details.

## Testing

### Unit Tests

```bash
# Run all tests
cargo test --lib
```

### Integration Tests

Integration tests deploy to Stellar testnet and exercise the full contract lifecycle:

```bash
# Prerequisites: stellar CLI, jq, curl
cd stellar/contracts/wormhole-contract/src/tests/
./run-integration-tests.sh testnet
```

This script:
1. Deploys a fresh contract via `scripts/deploy.sh`
2. Verifies guardian set upgrades (0 → 1 → 2)
3. Tests message fee governance
4. Tests fee transfers
5. Posts messages with and without fees

## Deployment

This repo includes a deployment helper at `scripts/deploy.sh` which builds, deploys, and
(optionally) initializes the contract using network-specific config in `scripts/config/`.

```bash
# Deploy to testnet using scripts/config/testnet.yaml
stellar/scripts/deploy.sh testnet

# Deploy to mainnet using scripts/config/mainnet.yaml (requires explicit confirmation flag)
stellar/scripts/deploy.sh mainnet --yes
```

## Integration Example

In your `Cargo.toml` add:

```toml
[dependencies]
wormhole-soroban-client = { path = "../wormhole-soroban-client" }
```


If you want to verify a Wormhole VAA and read the payload:

```rust
#![no_std]
use soroban_sdk::{contract, contractimpl, Address, Bytes, BytesN, Env};
use wormhole_soroban_client::{WormholeClient, VAA};

#[contract]
pub struct ReceiverContract;

#[contractimpl]
impl ReceiverContract {
    pub fn receive_message(
        env: Env,
        wormhole: Address,
        vaa_bytes: Bytes,
    ) -> (u32, BytesN<32>, Bytes) {
        let client = WormholeClient::new(&env, &wormhole);

        // Verifies signatures against the stored guardian set
        client.verify_vaa(&vaa_bytes);

        let vaa = client.parse_vaa(&vaa_bytes);

        (vaa.emitter_chain, vaa.emitter_address, vaa.payload)
    }
}
```



## License

Apache-2.0. See `LICENSE` at the repository root.
