//! **ATTENTION INTEGRATORS!** Token Bridge Program developer kit. It is recommended to use
//! this module for invoking Token Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](crate::legacy::cpi)).

#[doc(inline)]
pub use core_bridge_program::sdk::{EmitterInfo, Timestamp, VaaAccount, SOLANA_CHAIN};

#[doc(inline)]
pub use wormhole_raw_vaas::token_bridge::{
    Attestation, TokenBridgeMessage, TokenBridgePayload, Transfer, TransferWithMessage,
};

#[doc(inline)]
#[cfg(feature = "cpi")]
pub use crate::legacy::cpi::{
    complete_transfer_native, complete_transfer_with_payload_native,
    complete_transfer_with_payload_wrapped, complete_transfer_wrapped, transfer_tokens_native,
    transfer_tokens_with_payload_native, transfer_tokens_with_payload_wrapped,
    transfer_tokens_wrapped, CompleteTransferNative, CompleteTransferWithPayloadNative,
    CompleteTransferWithPayloadWrapped, CompleteTransferWrapped, TransferTokensNative,
    TransferTokensWithPayloadNative, TransferTokensWithPayloadWrapped, TransferTokensWrapped,
};
#[doc(inline)]
pub use crate::{
    constants::{PROGRAM_REDEEMER_SEED_PREFIX, PROGRAM_SENDER_SEED_PREFIX},
    legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs},
    state,
};

pub mod io {
    pub use core_bridge_program::sdk::io::*;
}

/// Wormhole Token Bridge Program.
pub type TokenBridge = crate::program::WormholeTokenBridgeSolana;
