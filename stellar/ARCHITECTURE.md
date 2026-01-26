# Wormhole Stellar Architecture

Technical documentation for the Wormhole Core Contract implementation on Stellar/Soroban.

## Overview

This contract enables Stellar (**Chain ID 61**) to participate in Wormhole's cross-chain messaging protocol:

- **Guardian Network**: 19 validators observe and attest to cross-chain messages
- **VAAs (Verifiable Action Approvals)**: Signed attestations requiring 13-of-19 guardian signatures
- **Core Contracts**: On-chain contracts that verify VAAs and post messages for guardian observation

### Why Two Crates?

- **wormhole-soroban-client**: Lightweight public API. External contracts depend only on this, resulting in smaller WASM binaries.
- **wormhole-contract**: Full implementation with storage access and business logic.

## Contract Interface

The contract implements `WormholeCoreInterface` with the following public functions:

### Initialization

```rust
/// Initialize the contract with the initial guardian set.
/// Can only be called once.
fn initialize(
    env: Env,
    initial_guardians: Vec<BytesN<20>>,  // Ethereum addresses
    governance_emitter: BytesN<32>,       // Authorized governance source
) -> Result<(), WormholeError>;

/// Check if the contract has been initialized.
fn is_initialized(env: Env) -> bool;
```

### VAA Operations

```rust
/// Verify VAA signatures against stored guardian set.
fn verify_vaa(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

/// Parse VAA bytes into structured data (no signature verification).
fn parse_vaa(env: Env, vaa_bytes: Bytes) -> Result<VAA, WormholeError>;
```

### Governance Actions

All governance requires a signed VAA from the Wormhole governance source (Solana chain 1).

```rust
/// Action 1: Upgrade contract WASM to new hash.
fn submit_contract_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

/// Action 2: Install new guardian set (index must be current + 1).
fn submit_guardian_set_upgrade(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

/// Action 3: Update message posting fee.
fn submit_set_message_fee(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;

/// Action 4: Transfer accumulated fees to recipient.
fn submit_transfer_fees(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;
```

### Message Posting

```rust
/// Post a cross-chain message for guardian attestation.
/// Returns the assigned sequence number.
fn post_message(
    env: Env,
    emitter: Address,              // Must authorize the call
    nonce: u32,                    // Caller-provided deduplication
    payload: Bytes,                // Application-specific data
    consistency_level: ConsistencyLevel,
) -> Result<u64, WormholeError>;
```

### State Queries

```rust
fn get_current_guardian_set_index(env: Env) -> u32;
fn get_guardian_set(env: Env, index: u32) -> Result<GuardianSetInfo, WormholeError>;
fn get_guardian_set_expiry(env: Env, index: u32) -> Option<u64>;
fn get_message_fee(env: Env) -> u64;
fn get_emitter_sequence(env: Env, emitter: Address) -> u64;
fn get_posted_message_hash(env: Env, emitter: Address, sequence: u64) -> Option<BytesN<32>>;
fn get_last_fee_transfer(env: Env) -> Option<u64>;
fn get_contract_balance(env: Env) -> i128;
fn is_governance_vaa_consumed(env: Env, vaa_bytes: Bytes) -> Result<(), WormholeError>;
fn get_chain_id() -> u32;
fn get_governance_chain_id() -> u32;
fn get_governance_emitter(env: Env) -> BytesN<32>;
```

## Core Types

### VAA (Verifiable Action Approval)

The fundamental cross-chain message format:

```rust
pub struct VAA {
    pub version: u8,
    pub guardian_set_index: u32,
    pub signatures: Vec<Signature>,
    pub timestamp: u32,
    pub nonce: u32,
    pub emitter_chain: u32,
    pub emitter_address: BytesN<32>,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: Bytes,
    pub body_hash: BytesN<32>,
}
```

Binary layout:

```
┌─────────────────────────────────────────────────────────────┐
│ Header (6 bytes)                                            │
│   version (1) │ guardian_set_index (4) │ signature_count (1)│
├─────────────────────────────────────────────────────────────┤
│ Signatures (66 bytes each)                                  │
│   guardian_index (1) │ r (32) │ s (32) │ v (1)              │
├─────────────────────────────────────────────────────────────┤
│ Body (variable)                                             │
│   timestamp (4) │ nonce (4) │ emitter_chain (2) │           │
│   emitter_address (32) │ sequence (8) │                     │
│   consistency_level (1) │ payload (variable)                │
└─────────────────────────────────────────────────────────────┘
```

### GuardianSetInfo

Stores guardian set metadata on-chain:

```rust
pub struct GuardianSetInfo {
    pub keys: Vec<BytesN<20>>,  // Ethereum addresses (20 bytes each)
    pub expiration_time: u64,    // 0 if current set, timestamp if expired
}
```

### Signature

Individual guardian signature:

```rust
pub struct Signature {
    pub guardian_index: u8,
    pub r: BytesN<32>,
    pub s: BytesN<32>,
    pub v: u8,
}
```

### ConsistencyLevel

Finality requirements for message attestation:

```rust
pub enum ConsistencyLevel {
    Confirmed = 0,   // Faster, less secure
    Finalized = 1,   // Slower, maximum security
}
```

## Signature Verification

1. Serialize VAA body to bytes
2. Double hash: `keccak256(keccak256(body))`
3. For each signature, recover secp256k1 public key
4. Derive Ethereum address from public key
5. Compare against guardian set keys
6. Require 13-of-19 signatures (quorum)

## Governance Flow

All governance actions follow the same pattern:

1. Parse and verify VAA signatures
2. Verify governance source (chain 1, emitter `0x...04`)
3. Check VAA not already consumed (replay protection)
4. Parse action-specific payload
5. Validate payload (e.g., sequential guardian set index)
6. **Consume VAA before execution** (critical for contract upgrades)
7. Execute the action

## Protocol Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `CHAIN_ID_STELLAR` | 61 | Stellar's Wormhole chain ID |
| `GOVERNANCE_CHAIN_ID` | 1 | Solana (governance source) |
| `GOVERNANCE_EMITTER` | `0x...04` | Authorized governance address |
| `GUARDIAN_SET_EXPIRATION_TIME` | 86,400s | 24 hours grace period |
| `MINIMUM_CONTRACT_BALANCE` | 10^7 stroops | 1 XLM minimum |
| `STORAGE_TTL_THRESHOLD` | 100,000 | ~5.8 days |
| `STORAGE_TTL_EXTENSION` | 1,000,000 | ~58 days |

## Security Features

### Replay Protection

- Governance VAA hashes are tracked and cannot be reprocessed
- VAAs are marked consumed **before** execution (critical for contract upgrades)

### Guardian Set Management

- Sets can only upgrade sequentially (n → n+1)
- Old sets expire after 24 hours (grace period for in-flight VAAs)
- Cannot overwrite existing sets

### Balance Protection

- Contract must maintain ≥1 XLM to prevent Stellar account deallocation
- Fee transfers validate remaining balance

## Error Handling

Errors are categorized by range:

| Range | Category | Examples |
|-------|----------|----------|
| 1-19 | VAA Errors | `InvalidVAAFormat`, `InvalidSignature`, `InsufficientSignatures` |
| 20-29 | Initialization | `NotInitialized`, `AlreadyInitialized` |
| 30-39 | Governance | `InvalidGovernanceChain`, `GovernanceVAAAlreadyConsumed` |
| 40-49 | Storage | `GuardianSetNotFound`, `EmptyGuardianSet` |
| 50-59 | Fees | `InsufficientFeePaid`, `InsufficientFunds` |
| 60-69 | Parsing | `InvalidPayload`, `UnexpectedEndOfInput` |
