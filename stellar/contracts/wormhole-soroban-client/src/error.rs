//! Error types for the Wormhole Core contract.
//!
//! All contract operations return `Result<T, WormholeError>`. Error codes are
//! grouped by category (VAA, initialization, governance, storage, fees) with
//! reserved numeric ranges for future expansion.

use soroban_sdk::contracterror;

/// Errors that can occur during Wormhole Core contract operations.
///
/// Error codes are organized into ranges:
/// - 1-19: VAA parsing and verification errors
/// - 20-29: Contract initialization errors
/// - 30-39: Governance action errors
/// - 40-49: Storage and guardian set errors
/// - 50-59: Fee-related errors
/// - 60+: Message posting errors
#[contracterror]
#[derive(Copy, Clone, Debug, Eq, PartialEq, PartialOrd, Ord)]
#[repr(u32)]
pub enum WormholeError {
    // ========== VAA Errors (1-19) ==========

    /// VAA bytes are malformed or truncated.
    InvalidVAAFormat = 1,
    /// Referenced guardian set index does not exist.
    InvalidGuardianSetIndex = 2,
    /// Guardian set used for signing has expired.
    GuardianSetExpired = 3,
    /// VAA has fewer signatures than required quorum.
    InsufficientSignatures = 4,
    /// Guardian signature indices are not in ascending order.
    SignaturesNotAscending = 5,
    /// Signature references a guardian index beyond set size.
    GuardianIndexOutOfBounds = 6,
    /// ECDSA signature verification failed.
    InvalidSignature = 7,
    /// VAA has already been processed (replay attempt).
    VAAAlreadyProcessed = 8,
    /// Emitter address is malformed or invalid.
    InvalidEmitterAddress = 9,
    /// Payload bytes are malformed or insufficient length.
    InvalidPayload = 10,

    // ========== Initialization Errors (20-29) ==========

    /// Contract has already been initialized.
    AlreadyInitialized = 20,
    /// Contract must be initialized before this operation.
    NotInitialized = 21,

    // ========== Governance Errors (30-39) ==========

    /// Governance payload module identifier is not "Core".
    InvalidGovernanceModule = 30,
    /// Governance action ID does not match expected action.
    InvalidGovernanceAction = 31,
    /// Governance VAA chain ID is not valid for this contract.
    InvalidGovernanceChain = 32,
    /// Governance VAA emitter address is not authorized.
    InvalidGovernanceEmitter = 33,
    /// New guardian set index is not current + 1.
    InvalidGuardianSetSequence = 34,
    /// Guardian set must contain at least one guardian.
    EmptyGuardianSet = 35,
    /// Governance VAA has already been consumed (replay protection).
    GovernanceVAAAlreadyConsumed = 36,

    // ========== Storage Errors (40-49) ==========

    /// Generic storage operation failure.
    StorageError = 40,
    /// Requested guardian set does not exist in storage.
    GuardianSetNotFound = 41,
    /// Cannot overwrite an existing guardian set.
    GuardianSetAlreadyExists = 42,

    // ========== Fee Errors (50-59) ==========

    /// Emitter has not approved sufficient fee for message posting.
    InsufficientFeePaid = 50,
    /// Contract balance too low for requested fee transfer.
    InsufficientFees = 51,
    /// Fee transfer recipient address is invalid.
    InvalidRecipient = 52,
    /// Token transfer operation failed.
    TransferFailed = 53,
    /// Fee amount is invalid (e.g., exceeds safe limits).
    InvalidFeeAmount = 54,

    // ========== Message Errors (60+) ==========

    /// Consistency level value is not recognized.
    InvalidConsistencyLevel = 60,
}
