//! **ATTENTION INTEGRATORS!** Token Bridge Program developer kit. It is recommended to use
//! [sdk::cpi](mod@crate::sdk::cpi) for invoking Token Bridge instructions as opposed to the
//! code-generated Anchor CPI (found in [cpi](mod@crate::cpi)) and legacy CPI (found in
//! [legacy::cpi](mod@crate::legacy::cpi)).

pub use crate::{
    constants::{PROGRAM_REDEEMER_SEED_PREFIX, PROGRAM_SENDER_SEED_PREFIX},
    legacy::instruction::{TransferTokensArgs, TransferTokensWithPayloadArgs},
};
