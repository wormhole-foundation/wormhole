//! `register_chain` — Token Bridge governance handler.
//!
//! Stub: reserved in the dispatch table and `Instruction` discriminant set;
//! returns `NotImplemented` until a dedicated PR ports the handler. Planned
//! behavior: validate a Token Bridge `RegisterChain` governance VAA and write
//! (or upgrade) the `ChainRegistration` PDA that `submit_observations` /
//! `submit_vaas` cross-check incoming VAAs against.

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::GlobalAccountantError;
use crate::err;

pub fn process(
    _program_id: &Address,
    _accounts: &mut [AccountView],
    _data: &[u8],
) -> ProgramResult {
    Err(err(GlobalAccountantError::NotImplemented))
}
