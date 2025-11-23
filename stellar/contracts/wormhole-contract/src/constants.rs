#![allow(missing_docs)]

pub const GOVERNANCE_CHAIN_ID: u32 = 1; // Solana by convention

/// Standard governance emitter address (0x00...04)
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
];

/// Stellar/Soroban chain ID in Wormhole.
///
/// Note: Chain IDs are u16 in the Wormhole protocol (2 bytes on wire).
/// All Wormhole chain IDs fit within u16 range (max 65,535).
pub const CHAIN_ID_STELLAR: u16 = 61;

/// Guardian set expiration time in seconds (24 hours): Need to confirm this
/// with the wormhole team
pub const GUARDIAN_SET_EXPIRATION_TIME: u32 = 86400;

/// Core module identifier for governance actions (right-padded "Core")
pub const MODULE_CORE: [u8; 32] = {
    let mut bytes = [0u8; 32];
    bytes[28] = b'C';
    bytes[29] = b'o';
    bytes[30] = b'r';
    bytes[31] = b'e';
    bytes
};

/// Governance action IDs
pub const ACTION_CONTRACT_UPGRADE: u8 = 1;
pub const ACTION_GUARDIAN_SET_UPGRADE: u8 = 2;
pub const ACTION_SET_MESSAGE_FEE: u8 = 3;
pub const ACTION_TRANSFER_FEES: u8 = 4;

/// Minimum balance to maintain in contract (prevent deallocation)
/// 1 XLM in stroops (1 XLM = 10^7 stroops)
pub const MINIMUM_CONTRACT_BALANCE: u64 = 10_000_000;

/// Native XLM token symbol for Stellar Asset Contract
pub const NATIVE_TOKEN_SYMBOL: &str = "native";

/// Native XLM token address for Stellar Asset Contract. These addresses are deterministic and never change
/// Both testnet and mainnet use the same address for native token
pub const NATIVE_TOKEN_ADDRESS: &str = "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC";

// ========== Storage Configuration ==========

/// Minimum TTL threshold for persistent storage entries.
///
/// This value represents approximately 5.8 days of ledger lifetime
/// at the standard rate of 5 seconds per ledger (100,000 ledgers × 5s ≈ 5.8 days).
///
/// When calling `extend_ttl()`, this is the minimum number of ledgers
/// the entry must live if not already longer.
pub const STORAGE_TTL_THRESHOLD: u32 = 100_000;

/// Maximum TTL extension for persistent storage entries.
///
/// This value represents approximately 58 days of ledger lifetime
/// at the standard rate of 5 seconds per ledger (1,000,000 ledgers × 5s ≈ 58 days).
///
/// When calling `extend_ttl()`, entries are extended to live at most this many ledgers.
pub const STORAGE_TTL_EXTENSION: u32 = 1_000_000;

// ========== Payload Structure Constants ==========

/// U256 padding size in governance payloads (Ethereum compatibility).
///
/// Fee and amount fields in governance payloads are encoded as U256 (32 bytes) for
/// Ethereum compatibility. On Stellar/Soroban, we use u64 values, so we skip the
/// first 24 bytes of padding and read only the last 8 bytes as the actual value.
pub const U256_PADDING_BYTES: u32 = 24;

/// Minimum payload length for Contract Upgrade governance action.
///
/// Payload structure:
/// - module: 32 bytes (module identifier, e.g., "Core")
/// - action: 1 byte (action ID = 1 for contract upgrade)
/// - chain: 2 bytes (target chain ID, 0 for all chains or 40 for Stellar)
/// - new_contract_hash: 32 bytes (WASM hash of new contract version)
///
/// Total: 67 bytes minimum
pub const CONTRACT_UPGRADE_PAYLOAD_MIN_LENGTH: u32 = 67;

/// Minimum payload length for Guardian Set Upgrade governance action.
///
/// Payload structure:
/// - module: 32 bytes (module identifier, e.g., "Core")
/// - action: 1 byte (action ID = 2 for guardian set upgrade)
/// - chain: 2 bytes (target chain ID, 0 for all chains or 40 for Stellar)
/// - new_guardian_set_index: 4 bytes (sequential index, must be current + 1)
/// - guardian_count: 1 byte (number of guardians, followed by 20 bytes per guardian)
///
/// Total: 40 bytes minimum (actual length varies based on guardian count)
pub const GUARDIAN_SET_UPGRADE_PAYLOAD_MIN_LENGTH: u32 = 40;

/// Minimum payload length for Set Message Fee governance action.
///
/// Payload structure:
/// - module: 32 bytes (module identifier, e.g., "Core")
/// - action: 1 byte (action ID = 3 for set message fee)
/// - chain: 2 bytes (target chain ID, 0 for all chains or 40 for Stellar)
/// - fee: 32 bytes (U256 format, last 8 bytes are actual u64 fee in stroops)
///
/// Total: 67 bytes minimum
pub const SET_MESSAGE_FEE_PAYLOAD_MIN_LENGTH: u32 = 67;

/// Minimum payload length for Transfer Fees governance action.
///
/// Payload structure:
/// - module: 32 bytes (module identifier, e.g., "Core")
/// - action: 1 byte (action ID = 4 for transfer fees)
/// - chain: 2 bytes (target chain ID, 0 for all chains or 40 for Stellar)
/// - amount: 32 bytes (U256 format, last 8 bytes are actual u64 amount in stroops)
/// - recipient: 32 bytes (ED25519 public key, converted to Stellar Address)
///
/// Total: 99 bytes minimum
pub const TRANSFER_FEES_PAYLOAD_MIN_LENGTH: u32 = 99;

// ========== Event Topics ==========

/// Event namespace for core contract events.
///
/// Used as the first topic in all governance and lifecycle events
/// (contract upgrades, guardian set updates, fee management, initialization).
pub const EVENT_NAMESPACE_CORE: &str = "wormhole_core";

/// Event namespace for message publishing events.
///
/// Used for cross-chain message events that guardians observe and attest to.
pub const EVENT_NAMESPACE_MESSAGES: &str = "wormhole";

/// Event topic for contract upgrade actions.
///
/// Published when a contract upgrade governance action is executed.
pub const EVENT_TOPIC_CONTRACT_UPGRADE: &str = "upgrade";

/// Event topic for guardian set upgrade actions.
///
/// Published when the guardian set is upgraded to a new version.
pub const EVENT_TOPIC_GUARDIAN_SET_UPGRADE: &str = "gs_upg";

/// Event topic for message fee updates.
///
/// Published when the message posting fee is changed via governance.
pub const EVENT_TOPIC_MESSAGE_FEE_SET: &str = "fee_set";

/// Event topic for fee transfers.
///
/// Published when accumulated fees are transferred out via governance.
pub const EVENT_TOPIC_FEE_TRANSFER: &str = "fee_xfer";

/// Event topic for contract initialization.
///
/// Published once during initial contract setup.
pub const EVENT_TOPIC_INIT: &str = "init";

/// Event topic for message publishing.
///
/// Published when a cross-chain message is posted to the contract.
pub const EVENT_TOPIC_MESSAGE_PUBLISHED: &str = "message_published";
