//! Wormhole-specific errors

use num_enum::TryFromPrimitive;
use solana_program::program_error::ProgramError as ProgErr;

use std::{error, fmt::Display};

/// Custom error type, meant to give more context before collapsing to
/// Solana's error type.
#[derive(Debug, Eq, PartialEq, TryFromPrimitive)]
#[repr(u8)]
pub enum Error {
    Internal = 0,
    InvalidInstructionKind,
    InvalidMagic,
    UnexpectedEndOfBuffer,
    UnexpectedEndOfAccounts,
    InvalidOwner,
    EEVAATooLarge,
}

/// Needed for `std::error::Error`.
impl Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{:?}", self)
    }
}

impl error::Error for Error {}

/// Used in the entrypoint to conveniently return an error Solana
/// understands after printing a Solana-specific message.
impl From<Error> for ProgErr {
    fn from(e: Error) -> Self {
        match e {
            Error::Internal => ProgErr::Custom(Error::Internal as u32),
            Error::InvalidInstructionKind => ProgErr::InvalidInstructionData,
            Error::InvalidMagic => ProgErr::InvalidInstructionData,
            Error::UnexpectedEndOfBuffer => ProgErr::InvalidInstructionData,
            Error::UnexpectedEndOfAccounts => ProgErr::InvalidAccountData,
            Error::InvalidOwner => ProgErr::InvalidAccountData,
            Error::EEVAATooLarge => ProgErr::InvalidInstructionData,
        }
    }
}
