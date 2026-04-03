//! Public protocol constants for the Wormhole Core contract.
//!
//! These constants define core Wormhole protocol parameters used for VAA
//! validation, governance processing, and cross-chain message handling.
//! External integrators may reference these when building on top of the
//! Wormhole Core contract.

/// Chain ID of the governance source chain (Solana).
///
/// All governance VAAs must originate from this chain to be considered valid.
pub const GOVERNANCE_CHAIN_ID: u32 = 1;

/// Standard governance emitter address on Solana (`0x00...04`).
///
/// This 32-byte address identifies the authorized governance contract that
/// can issue guardian set upgrades, fee changes, and contract upgrades.
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
];

/// Wormhole chain ID for Stellar/Soroban (61).
///
/// Used in VAA payloads to target this specific chain. Chain IDs are encoded
/// as 2 bytes (u16) on the wire but stored as u16 for type safety.
pub const CHAIN_ID_STELLAR: u16 = 61;

/// Grace period before an old guardian set expires (24 hours in seconds).
///
/// After a guardian set upgrade, the old set remains valid for this duration
/// to allow in-flight VAAs signed by the previous guardians to be processed.
pub const GUARDIAN_SET_EXPIRATION_TIME: u32 = 86400;

/// Module identifier for Wormhole Core governance actions.
///
/// Right-padded ASCII "Core" in a 32-byte field. Governance payloads must
/// contain this exact value to be recognized as Core contract actions.
pub const MODULE_CORE: [u8; 32] = {
    let mut bytes = [0u8; 32];
    bytes[28] = b'C';
    bytes[29] = b'o';
    bytes[30] = b'r';
    bytes[31] = b'e';
    bytes
};

// ========== Governance Action IDs ==========

/// Action ID for contract WASM upgrade (governance action 1).
pub const ACTION_CONTRACT_UPGRADE: u8 = 1;

/// Action ID for guardian set upgrade (governance action 2).
pub const ACTION_GUARDIAN_SET_UPGRADE: u8 = 2;

/// Action ID for setting the message fee (governance action 3).
pub const ACTION_SET_MESSAGE_FEE: u8 = 3;

/// Action ID for transferring accumulated fees (governance action 4).
pub const ACTION_TRANSFER_FEES: u8 = 4;

// ========== Token Constants ==========

/// Native XLM token symbol for Stellar Asset Contract.
pub const NATIVE_TOKEN_SYMBOL: &str = "native";

/// Native XLM Stellar Asset Contract address (deterministic, same on all
/// networks).
pub const NATIVE_TOKEN_ADDRESS: &str = "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC";

// ========== Storage Configuration ==========

/// TTL threshold for persistent storage renewal (approx. 5.8 days at
/// 5s/ledger).
///
/// Storage entries are extended when their remaining TTL falls below this
/// threshold.
pub const STORAGE_TTL_THRESHOLD: u32 = 100_000;

/// TTL extension amount for persistent storage (approx. 58 days at 5s/ledger).
///
/// When extending TTL, entries are renewed to live this many additional
/// ledgers.
pub const STORAGE_TTL_EXTENSION: u32 = 1_000_000;

// ========== VAA Structure Constants ==========

/// Minimum VAA header size: version (1) + guardian_set_index (4) +
/// num_signatures (1).
pub const VAA_HEADER_MIN_LENGTH: u32 = 6;

// ========== Payload Structure Constants ==========

/// U256 padding bytes to skip when reading u64 values from Ethereum-compatible
/// payloads.
///
/// Governance payloads encode amounts as 32-byte U256 for Ethereum
/// compatibility. On Soroban we read only the low-order 8 bytes as u64,
/// skipping the first 24.
pub const U256_PADDING_BYTES: u32 = 24;

/// Minimum Contract Upgrade payload: module (32) + action (1) + chain (2) +
/// hash (32).
pub const CONTRACT_UPGRADE_PAYLOAD_MIN_LENGTH: u32 = 67;

/// Minimum Guardian Set Upgrade payload: header (35) + index (4) + count (1).
///
/// Actual length depends on guardian count (add 20 bytes per guardian).
pub const GUARDIAN_SET_UPGRADE_PAYLOAD_MIN_LENGTH: u32 = 40;

/// Minimum Set Message Fee payload: module (32) + action (1) + chain (2) + fee
/// U256 (32).
pub const SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH: u32 = 67;

/// Minimum Transfer Fees payload: header (35) + amount U256 (32) + recipient
/// (32).
pub const TRANSFER_FEES_PAYLOAD_MIN_LENGTH: u32 = 99;
