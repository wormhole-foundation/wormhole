//! Persistent storage key definitions for the Wormhole Core contract.
//!
//! All contract state is stored under these keys. Keys are extended
//! automatically on access to maintain TTL (see `STORAGE_TTL_THRESHOLD` and
//! `STORAGE_TTL_EXTENSION`).

use soroban_sdk::{Address, BytesN, contracttype};

/// Storage key enum for all persistent contract state.
///
/// Uses Soroban's `contracttype` for efficient serialization. Keys with
/// parameters (e.g., `GuardianSet(u32)`) create separate storage entries per
/// value.
#[derive(Clone)]
#[contracttype]
pub enum StorageKey {
    /// Index of the currently active guardian set (starts at 0).
    CurrentGuardianSetIndex,
    /// Guardian set data keyed by index; stores `GuardianSetInfo`.
    GuardianSet(u32),
    /// Unix timestamp when a guardian set expires (24h after replacement).
    GuardianSetExpiry(u32),
    /// Tracks consumed governance VAA hashes for replay protection.
    ConsumedGovernanceVAA(BytesN<32>),
    /// Fee charged per message in stroops (10^-7 XLM).
    MessageFee,
    /// Next sequence number for each emitter address.
    EmitterSequence(Address),
    /// Hash of an address used in `from/to` fields.
    AddressTable(BytesN<32>),
}
