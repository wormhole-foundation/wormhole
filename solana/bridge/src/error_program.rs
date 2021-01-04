#![cfg(feature = "program")]

use num_traits::FromPrimitive;
use solana_program::{decode_error::DecodeError, program_error::PrintProgramError};

use crate::error::*;

impl PrintProgramError for Error {
    fn print<E>(&self)
        where
            E: 'static + std::error::Error + DecodeError<E> + PrintProgramError + FromPrimitive,
    {
        match self {
            Error::ExpectedToken => msg!("Error: ExpectedToken"),
            Error::ExpectedAccount => msg!("Error: ExpectedAccount"),
            Error::ExpectedBridge => msg!("Error: ExpectedBridge"),
            Error::ExpectedGuardianSet => msg!("Error: ExpectedGuardianSet"),
            Error::ExpectedWrappedAssetMeta => msg!("Error: ExpectedWrappedAssetMeta"),
            Error::UninitializedState => msg!("Error: State is unititialized"),
            Error::InvalidProgramAddress => msg!("Error: InvalidProgramAddress"),
            Error::InvalidVAAFormat => msg!("Error: InvalidVAAFormat"),
            Error::InvalidVAAAction => msg!("Error: InvalidVAAAction"),
            Error::InvalidVAASignature => msg!("Error: InvalidVAASignature"),
            Error::AlreadyExists => msg!("Error: AlreadyExists"),
            Error::InvalidDerivedAccount => msg!("Error: InvalidDerivedAccount"),
            Error::TokenMintMismatch => msg!("Error: TokenMintMismatch"),
            Error::WrongMintOwner => msg!("Error: WrongMintOwner"),
            Error::WrongTokenAccountOwner => msg!("Error: WrongTokenAccountOwner"),
            Error::ParseFailed => msg!("Error: ParseFailed"),
            Error::GuardianSetExpired => msg!("Error: GuardianSetExpired"),
            Error::VAAClaimed => msg!("Error: VAAClaimed"),
            Error::WrongBridgeOwner => msg!("Error: WrongBridgeOwner"),
            Error::OldGuardianSet => msg!("Error: OldGuardianSet"),
            Error::GuardianIndexNotIncreasing => msg!("Error: GuardianIndexNotIncreasing"),
            Error::ExpectedTransferOutProposal => msg!("Error: ExpectedTransferOutProposal"),
            Error::VAAProposalMismatch => msg!("Error: VAAProposalMismatch"),
            Error::SameChainTransfer => msg!("Error: SameChainTransfer"),
            Error::VAATooLong => msg!("Error: VAATooLong"),
            Error::CannotWrapNative => msg!("Error: CannotWrapNative"),
            Error::VAAAlreadySubmitted => msg!("Error: VAAAlreadySubmitted"),
            Error::GuardianSetMismatch => msg!("Error: GuardianSetMismatch"),
            Error::InsufficientFees => msg!("Error: InsufficientFees"),
            Error::InvalidOwner => msg!("Error: InvalidOwner"),
            Error::InvalidSysvar => msg!("Error: InvalidSysvar"),
            Error::InvalidChain => msg!("Error: InvalidChain"),
        }
    }
}
