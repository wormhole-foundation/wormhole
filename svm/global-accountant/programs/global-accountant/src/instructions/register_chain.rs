//! `register_chain` — not yet implemented.
//!
//! Would be the Token Bridge governance handler: validates a Token Bridge
//! `RegisterChain` governance VAA and writes (or upgrades) the
//! `ChainRegistration` PDA that `submit_observations` / `submit_vaas`
//! cross-check incoming VAAs against.

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
