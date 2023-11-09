//! **ATTENTION INTEGRATORS!** Token Bridge Program developer kit. It is recommended to use
//! this module for invoking Token Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](crate::legacy::cpi)).

#[doc(inline)]
pub use core_bridge_program::sdk as core_bridge;

#[doc(inline)]
pub use crate::{
    constants::{PROGRAM_REDEEMER_SEED_PREFIX, PROGRAM_SENDER_SEED_PREFIX},
    legacy::cpi::{
        complete_transfer_native, complete_transfer_with_payload_native,
        complete_transfer_with_payload_wrapped, complete_transfer_wrapped, transfer_tokens_native,
        transfer_tokens_with_payload_native, transfer_tokens_with_payload_wrapped,
        transfer_tokens_wrapped, CompleteTransferNative, CompleteTransferWithPayloadNative,
        CompleteTransferWithPayloadWrapped, CompleteTransferWrapped, TransferTokensNative,
        TransferTokensWithPayloadNative, TransferTokensWithPayloadWrapped, TransferTokensWrapped,
    },
    legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs},
    state,
};

/// Set of structs mirroring the structs deriving Accounts, where each field is a Pubkey. This
/// is useful for specifying accounts for a client.
pub mod accounts {
    #[doc(inline)]
    pub use crate::{accounts::*, legacy::accounts::*};
}

/// Instruction builders. These should be used directly when one wants to serialize instruction
/// data when speciying instructions on a client.
pub mod instruction {
    #[doc(inline)]
    pub use crate::{accounts, instruction::*, legacy::instruction as legacy};
}

/// Wormhole Token Bridge Program.
pub type TokenBridge = crate::program::WormholeTokenBridgeSolana;
