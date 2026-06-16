//! Zero-copy load/store/verify helpers for `ChainRegistrationLayout`.
//!
//! Read-only on the submit paths; written only by `register_chain`.

use pinocchio::{account::Ref, error::ProgramError, AccountView, Address, ProgramResult};

use crate::definitions::{
    ChainRegistrationLayout, GlobalAccountantError, CHAIN_REGISTRATION_SEED_PREFIX,
};
use crate::err;

/// Read a [`ChainRegistrationLayout`]. Returns `MissingChainRegistration` if
/// the account is system-owned (no registration VAA landed) or `InvalidPda` on
/// wrong length. Caller must verify the canonical address first.
pub fn load(account: &AccountView) -> Result<ChainRegistrationLayout, ProgramError> {
    if account.owner() == &pinocchio_system::ID {
        return Err(err(GlobalAccountantError::MissingChainRegistration));
    }
    let data: Ref<'_, [u8]> = account.try_borrow()?;
    if data.len() != ChainRegistrationLayout::LEN {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    let layout = bytemuck::from_bytes::<ChainRegistrationLayout>(&data);
    // Defense-in-depth: offset-0 tag must match (tag 0 = uninitialised).
    if layout.tag != ChainRegistrationLayout::TAG {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    Ok(*layout)
}

/// Write a [`ChainRegistrationLayout`] into the account's data buffer (used by
/// `register_chain` after allocation).
pub fn store(
    account: &mut AccountView,
    value: &ChainRegistrationLayout,
) -> Result<(), ProgramError> {
    let mut data = account.try_borrow_mut()?;
    if data.len() != ChainRegistrationLayout::LEN {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    data.copy_from_slice(bytemuck::bytes_of(value));
    Ok(())
}

/// Chain-registration cross-check used by both submit paths:
/// 1. The account must live at the canonical address (rejecting a foreign
///    account masquerading as the registration PDA).
/// 2. The on-disk `emitter_address` must equal the body header's emitter.
pub fn verify(
    program_id: &Address,
    registration_pda: &AccountView,
    body_chain: u16,
    body_emitter: &[u8; 32],
) -> ProgramResult {
    let chain_be = body_chain.to_be_bytes();
    let (expected, _bump) =
        Address::find_program_address(&[CHAIN_REGISTRATION_SEED_PREFIX, &chain_be], program_id);
    if registration_pda.address() != &expected {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    let layout = load(registration_pda)?;
    if layout.emitter_address != *body_emitter {
        return Err(err(GlobalAccountantError::UnregisteredEmitter));
    }
    Ok(())
}
