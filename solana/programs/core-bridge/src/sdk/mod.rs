//! **ATTENTION INTEGRATORS!** Core Bridge Program developer kit. It is recommended to use
//! [sdk::cpi](crate::sdk::cpi) for invoking Core Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](crate::legacy::cpi)).

pub use crate::{
    constants::{PROGRAM_EMITTER_SEED_PREFIX, SOLANA_CHAIN},
    legacy::instruction::PostMessageArgs,
    processor::{
        ClosePostedVaaV1Directive, InitMessageV1Args, PostVaaV1Directive,
        ProcessEncodedVaaDirective, ProcessMessageV1Directive,
    },
    state, types, zero_copy,
};

pub mod accounts {
    //! Set of structs mirroring the structs deriving Accounts, where each field is a Pubkey. This
    //! is useful for specifying accounts for a client.

    pub use crate::{accounts::*, legacy::accounts::*};
}

/// CPI builders. Methods useful for interacting with the Core Bridge program from another program.
#[cfg(feature = "cpi")]
pub mod cpi;

pub mod instruction {
    //! Instruction builders. These should be used directly when one wants to serialize instruction
    //! data when speciying instructions on a client.
    pub use crate::{accounts, instruction::*, legacy::instruction as legacy};
}

/// The program ID of the Core Bridge program.
pub static PROGRAM_ID: anchor_lang::prelude::Pubkey = crate::ID;

/// Convenient method to determine the space required for a [PostedMessageV1](state::PostedVaaV1)
/// account when it is being prepared via
/// [init_message_v1](crate::wormhole_core_bridge_solana::init_message_v1) and
/// [process_message_v1](crate::wormhole_core_bridge_solana::process_message_v1).
pub fn compute_init_message_v1_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::BYTES_START + payload_size
}
