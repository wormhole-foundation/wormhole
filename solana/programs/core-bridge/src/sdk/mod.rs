//! **ATTENTION INTEGRATORS!** Core Bridge Program developer kit. It is recommended to use
//! [sdk::cpi](crate::sdk::cpi) for invoking Core Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](crate::legacy::cpi)).

#[doc(inline)]
pub use crate::{
    constants::{PROGRAM_EMITTER_SEED_PREFIX, SOLANA_CHAIN},
    legacy::instruction::PostMessageArgs,
    processor::{InitMessageV1Args, WriteEncodedVaaArgs, WriteMessageV1Args},
    types,
    zero_copy::{Config, LoadZeroCopy, MessageAccount, VaaAccount},
};

/// Set of structs mirroring the structs deriving Accounts, where each field is a Pubkey. This
/// is useful for specifying accounts for a client.
pub mod accounts {
    pub use crate::{accounts::*, legacy::accounts::*};
}

#[cfg(feature = "cpi")]
pub mod cpi;

/// Instruction builders. These should be used directly when one wants to serialize instruction
/// data when speciying instructions on a client.
pub mod instruction {
    pub use crate::{instruction::*, legacy::instruction as legacy};
}

/// Convenient method to determine the space required for a
/// [PostedMessageV1](crate::zero_copy::PostedMessageV1) account before the account is initialized
/// via [init_message_v1](crate::wormhole_core_bridge_solana::init_message_v1).
pub fn compute_prepared_message_space(payload_size: usize) -> usize {
    crate::state::PostedMessageV1::BYTES_START + payload_size
}
