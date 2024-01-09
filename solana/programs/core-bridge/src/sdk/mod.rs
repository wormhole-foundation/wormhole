//! **ATTENTION INTEGRATORS!** Core Bridge Program developer kit. It is recommended to use
//! [sdk::cpi](crate::sdk::cpi) for invoking Core Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](crate::legacy::cpi)).
//! CPI builders. Methods useful for interacting with the Core Bridge program from another program.

#[cfg(feature = "cpi")]
mod publish_message;
#[cfg(feature = "cpi")]
pub use publish_message::*;

#[cfg(feature = "cpi")]
mod prepare_message;
#[cfg(feature = "cpi")]
pub use prepare_message::*;

#[doc(inline)]
pub use wormhole_raw_vaas::{Header, Payload, Vaa};

/// Sub-module for System program interaction.
#[cfg(feature = "cpi")]
pub mod system_program {
    #[doc(inline)]
    pub use crate::utils::cpi::{create_account_safe, CreateAccountSafe};
}

#[doc(inline)]
pub use crate::{
    constants::{PROGRAM_EMITTER_SEED_PREFIX, SOLANA_CHAIN},
    id,
    legacy::instruction::PostMessageArgs,
    processor::{InitMessageV1Args, WriteMessageV1Args},
    state,
    types::*,
    utils::vaa::{EmitterInfo, VaaAccount},
};
#[doc(inline)]
#[cfg(feature = "cpi")]
pub use crate::{
    cpi::{finalize_message_v1, init_message_v1, write_message_v1},
    legacy::cpi::{post_message_unreliable, PostMessageUnreliable},
    utils::vaa::{claim_vaa, ClaimVaa},
};

pub mod io {
    pub use wormhole_io::{Readable, TypePrefixedPayload, Writeable};
}

pub mod legacy {
    pub use crate::legacy::utils::{
        AccountVariant, LegacyAccount, LegacyAnchorized, ProcessLegacyInstruction,
    };
}

/// Convenient method to determine the space required for a
/// [PostedMessageV1](crate::zero_copy::PostedMessageV1) account before the account is initialized
/// via [init_message_v1](crate::wormhole_core_bridge_solana::init_message_v1).
pub fn compute_prepared_message_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::PAYLOAD_START + payload_size
}

/// Wormhole Core Bridge Program.
pub type CoreBridge = crate::program::WormholeCoreBridgeSolana;
