#![cfg(feature = "program")]

use num_traits::FromPrimitive;
use solana_sdk::{decode_error::DecodeError, info, program_error::PrintProgramError};

use crate::error::*;

impl PrintProgramError for Error {
    fn print<E>(&self)
    where
        E: 'static + std::error::Error + DecodeError<E> + PrintProgramError + FromPrimitive,
    {
        match self {
            Error::ExpectedToken => info!("Error: ExpectedToken"),
            Error::ExpectedAccount => info!("Error: ExpectedAccount"),
            Error::ExpectedBridge => info!("Error: ExpectedBridge"),
            Error::ExpectedGuardianSet => info!("Error: ExpectedGuardianSet"),
            Error::ExpectedWrappedAssetMeta => info!("Error: ExpectedWrappedAssetMeta"),
            Error::UninitializedState => info!("Error: State is unititialized"),
            Error::InvalidProgramAddress => info!("Error: InvalidProgramAddress"),
            Error::InvalidVAAFormat => info!("Error: InvalidVAAFormat"),
            Error::InvalidVAAAction => info!("Error: InvalidVAAAction"),
            Error::InvalidVAASignature => info!("Error: InvalidVAASignature"),
            Error::AlreadyExists => info!("Error: AlreadyExists"),
            Error::InvalidDerivedAccount => info!("Error: InvalidDerivedAccount"),
            Error::TokenMintMismatch => info!("Error: TokenMintMismatch"),
            Error::WrongMintOwner => info!("Error: WrongMintOwner"),
            Error::WrongTokenAccountOwner => info!("Error: WrongTokenAccountOwner"),
            Error::ParseFailed => info!("Error: ParseFailed"),
            Error::GuardianSetExpired => info!("Error: GuardianSetExpired"),
            Error::VAAClaimed => info!("Error: VAAClaimed"),
            Error::WrongBridgeOwner => info!("Error: WrongBridgeOwner"),
            Error::OldGuardianSet => info!("Error: OldGuardianSet"),
            Error::GuardianIndexNotIncreasing => info!("Error: GuardianIndexNotIncreasing"),
            Error::ExpectedTransferOutProposal => info!("Error: ExpectedTransferOutProposal"),
            Error::VAAProposalMismatch => info!("Error: VAAProposalMismatch"),
            Error::SameChainTransfer => info!("Error: SameChainTransfer"),
            Error::VAATooLong => info!("Error: VAATooLong"),
            Error::CannotWrapNative => info!("Error: CannotWrapNative"),
            Error::VAAAlreadySubmitted => info!("Error: VAAAlreadySubmitted"),
            Error::GuardianSetMismatch => info!("Error: GuardianSetMismatch"),
        }
    }
}
