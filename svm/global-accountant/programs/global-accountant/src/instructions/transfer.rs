//! Balance-mutation helpers shared by the quorum-completing branch of
//! `submit_observations` and the signed-VAA backfill in `submit_vaas`.

use pinocchio::{AccountView, Address, ProgramResult};

use crate::definitions::{GlobalAccountantError, Uint256, ACCOUNT_SEED_PREFIX};
use crate::err;
use crate::state::account as account_state;

/// Mutate the source and destination Account PDAs for a Token Bridge transfer:
/// source-side `lock_or_burn`, then destination-side `unlock_or_mint`.
/// Same-chain self-transfers (source == dest PDA) are collapsed onto one
/// in-memory layout so the second mutation observes the first.
#[allow(clippy::too_many_arguments)]
pub fn apply_transfer(
    program_id: &Address,
    payer: &AccountView,
    source_account: &mut AccountView,
    dest_account: &mut AccountView,
    source_chain: u16,
    recipient_chain: u16,
    token_chain: u16,
    token_address: &[u8; 32],
    amount: Uint256,
) -> ProgramResult {
    // ----- Source side -----
    let (src_expected, src_bump) =
        derive_account_pda(program_id, source_chain, token_chain, token_address);
    if source_account.address() != &src_expected {
        return Err(err(GlobalAccountantError::InvalidAccountPda));
    }
    account_state::init_if_needed(
        program_id,
        payer,
        source_account,
        source_chain,
        token_chain,
        token_address,
        src_bump,
    )?;
    let mut src = account_state::load(source_account)?;
    src.lock_or_burn(amount).map_err(err)?;

    // Same-chain self-transfer collapse: apply both ops to one in-memory layout
    // in order. This pins the transient arithmetic (burn-then-mint must underflow
    // when the wrapped balance is below `amount`, even though the net is zero) and
    // guards against a lost-update if this function is ever refactored to a
    // load-both-then-mutate shape.
    let same_pda = source_account.address() == dest_account.address();
    if same_pda {
        src.unlock_or_mint(amount).map_err(err)?;
        account_state::store(source_account, &src)?;
        return Ok(());
    }

    // Distinct destination: flush source, then operate on the destination.
    account_state::store(source_account, &src)?;

    // ----- Destination side -----
    let (dst_expected, dst_bump) =
        derive_account_pda(program_id, recipient_chain, token_chain, token_address);
    if dest_account.address() != &dst_expected {
        return Err(err(GlobalAccountantError::InvalidAccountPda));
    }
    account_state::init_if_needed(
        program_id,
        payer,
        dest_account,
        recipient_chain,
        token_chain,
        token_address,
        dst_bump,
    )?;
    let mut dst = account_state::load(dest_account)?;
    dst.unlock_or_mint(amount).map_err(err)?;
    account_state::store(dest_account, &dst)
}

/// Re-derive the canonical Account PDA address + bump from `(chain, token_chain,
/// token_address)`.
pub fn derive_account_pda(
    program_id: &Address,
    chain: u16,
    token_chain: u16,
    token_address: &[u8; 32],
) -> (Address, u8) {
    let chain_be = chain.to_be_bytes();
    let token_chain_be = token_chain.to_be_bytes();
    Address::find_program_address(
        &[
            ACCOUNT_SEED_PREFIX,
            &chain_be,
            &token_chain_be,
            token_address,
        ],
        program_id,
    )
}
