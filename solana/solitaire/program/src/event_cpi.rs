//! Anchor-compatible event emission via self-CPI.
//!
//! Programs built with `solitaire!{ @event_cpi, ... }` install a dispatch
//! guard that intercepts self-CPIs prefixed with [`crate::EVENT_IX_TAG_LE`],
//! returning `Ok` without dispatching to a handler. The observable effect of
//! such a CPI is the log line emitted by the runtime — off-chain indexers
//! match those lines by `discriminator` to extract typed events.

use solana_program::{
    account_info::AccountInfo,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program::invoke_signed,
    pubkey::Pubkey,
};

use crate::{
    error::SolitaireError,
    ExecutionContext,
    Result,
    EVENT_AUTHORITY_SEED,
    EVENT_IX_TAG_LE,
};

/// Emit an Anchor-style event by self-invoking the current program with the
/// `__event_authority` PDA as the sole signing account. The full instruction
/// data is `EVENT_IX_TAG_LE || discriminator || payload`.
///
/// `event_authority` must be the PDA derived from [`EVENT_AUTHORITY_SEED`] for
/// `ctx.program_id`; otherwise [`SolitaireError::InvalidDerive`] is returned.
/// The caller is expected to have separately verified that the program-self
/// account it provides matches `ctx.program_id`.
pub fn emit_event_cpi(
    ctx: &ExecutionContext,
    event_authority: &AccountInfo,
    discriminator: &[u8; 8],
    payload: &[u8],
) -> Result<()> {
    let (expected_authority, bump) =
        Pubkey::find_program_address(&[EVENT_AUTHORITY_SEED], ctx.program_id);
    if event_authority.key != &expected_authority {
        return Err(SolitaireError::InvalidDerive(
            *event_authority.key,
            expected_authority,
        ));
    }

    let mut ix_data =
        Vec::with_capacity(EVENT_IX_TAG_LE.len() + discriminator.len() + payload.len());
    ix_data.extend_from_slice(&EVENT_IX_TAG_LE);
    ix_data.extend_from_slice(discriminator);
    ix_data.extend_from_slice(payload);

    let ix = Instruction {
        program_id: *ctx.program_id,
        accounts: vec![AccountMeta::new_readonly(expected_authority, true)],
        data: ix_data,
    };

    invoke_signed(&ix, ctx.accounts, &[&[EVENT_AUTHORITY_SEED, &[bump]]])?;
    Ok(())
}
