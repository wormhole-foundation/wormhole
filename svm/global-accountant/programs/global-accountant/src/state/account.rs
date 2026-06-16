//! Zero-copy load / store / lazy-init helpers for [`BalanceAccountLayout`].
//!
//! Canonical PDA at `(b"account", chain_be, token_chain_be, token_address)`.
//! The layout is `Pod`, so load/store copy by value to release the borrow
//! before the caller mutates. Lazy init lets the destination Account PDA come
//! into existence on the quorum-completing tx (payer pays rent).

use pinocchio::{
    account::Ref,
    cpi::{Seed, Signer},
    error::ProgramError,
    AccountView, Address, ProgramResult,
};

use crate::definitions::{
    BalanceAccountLayout, GlobalAccountantError, Uint256, ACCOUNT_SEED_PREFIX,
};
use crate::err;
use crate::instructions::pda_init::init_or_upgrade_pda;

/// Read a [`BalanceAccountLayout`]. `InvalidPda` if the buffer is not `LEN`.
pub fn load(account: &AccountView) -> Result<BalanceAccountLayout, ProgramError> {
    let data: Ref<'_, [u8]> = account.try_borrow()?;
    if data.len() != BalanceAccountLayout::LEN {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    let layout = bytemuck::from_bytes::<BalanceAccountLayout>(&data);
    // Defense-in-depth: the offset-0 tag must match, else this is corrupted or
    // foreign data at a correctly-sized address (tag 0 = uninitialised).
    if layout.tag != BalanceAccountLayout::TAG {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    Ok(*layout)
}

/// Write a [`BalanceAccountLayout`] into an account's data buffer.
pub fn store(account: &mut AccountView, value: &BalanceAccountLayout) -> Result<(), ProgramError> {
    let mut data = account.try_borrow_mut()?;
    if data.len() != BalanceAccountLayout::LEN {
        return Err(err(GlobalAccountantError::InvalidPda));
    }
    data.copy_from_slice(bytemuck::bytes_of(value));
    Ok(())
}

/// Lazy-init the canonical Account PDA. Idempotent: a no-op if the PDA already
/// exists and is program-owned. Caller must enforce the canonical bump first
/// (see `verify_account_pda` in `submit_observations`).
pub fn init_if_needed(
    program_id: &Address,
    payer: &AccountView,
    account_pda: &mut AccountView,
    chain: u16,
    token_chain: u16,
    token_address: &[u8; 32],
    canonical_bump: u8,
) -> ProgramResult {
    // Already initialised (program-owned + full length) ⇒ no-op.
    // `init_or_upgrade_pda` only accepts system-owned, empty accounts.
    if account_pda.owner() != &pinocchio_system::ID
        && account_pda.data_len() == BalanceAccountLayout::LEN
    {
        return Ok(());
    }

    let chain_be = chain.to_be_bytes();
    let token_chain_be = token_chain.to_be_bytes();
    let bump_seed = [canonical_bump];
    let seeds_with_bump = [
        Seed::from(ACCOUNT_SEED_PREFIX),
        Seed::from(chain_be.as_slice()),
        Seed::from(token_chain_be.as_slice()),
        Seed::from(token_address.as_slice()),
        Seed::from(bump_seed.as_slice()),
    ];
    let signer = Signer::from(&seeds_with_bump);

    init_or_upgrade_pda(
        payer,
        account_pda,
        program_id,
        signer,
        BalanceAccountLayout::LEN as u64,
    )?;

    // Stamp the keying triple into the freshly-zeroed layout (self-describing
    // record); balance starts at zero, credited/debited by the caller.
    let mut layout: BalanceAccountLayout = bytemuck::Zeroable::zeroed();
    layout.tag = BalanceAccountLayout::TAG;
    layout.chain = chain;
    layout.token_chain = token_chain;
    layout.token_address = *token_address;
    layout.balance = Uint256::ZERO;
    store(account_pda, &layout)
}
