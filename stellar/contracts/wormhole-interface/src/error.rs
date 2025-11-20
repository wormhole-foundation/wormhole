//! Error types for the Wormhole Core contract.
//!
//! This module defines all possible error conditions that can occur when
//! interacting with the Wormhole Core contract. External users must handle
//! these errors when calling contract methods.

#![allow(missing_docs)]

use soroban_sdk::contracterror;

#[contracterror]
#[derive(Copy, Clone, Debug, Eq, PartialEq, PartialOrd, Ord)]
#[repr(u32)]
pub enum Error {
    InvalidVAAFormat = 1,
    InvalidGuardianSetIndex = 2,
    GuardianSetExpired = 3,
    InsufficientSignatures = 4,
    SignaturesNotAscending = 5,
    GuardianIndexOutOfBounds = 6,
    InvalidSignature = 7,
    VAAAlreadyProcessed = 8,
    InvalidEmitterAddress = 9,
    InvalidPayload = 10,

    // Initialization Errors
    AlreadyInitialized = 20,
    NotInitialized = 21,

    // Governance Errors
    InvalidGovernanceModule = 30,
    InvalidGovernanceAction = 31,
    InvalidGovernanceChain = 32,
    InvalidGovernanceEmitter = 33,
    InvalidGuardianSetSequence = 34,
    EmptyGuardianSet = 35,
    GovernanceVAAAlreadyConsumed = 36,

    // Storage Errors
    StorageError = 40,
    GuardianSetNotFound = 41,
    GuardianSetAlreadyExists = 42,

    // Fee Errors
    InsufficientFeePaid = 50,
    InsufficientFees = 51,
    InvalidRecipient = 52,
    TransferFailed = 53,
    InvalidFeeAmount = 54,

}
