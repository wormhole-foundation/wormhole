use crate::{error::WormholeVaaVerificationComparisonError, state::GuardianSignatures};
use anchor_lang::prelude::*;

#[derive(Accounts)]
#[instruction(_guardian_signatures: Vec<[u8; 66]>, total_signatures: u8)]
pub struct PostSignatures<'info> {
    #[account(mut)]
    payer: Signer<'info>,

    #[account(
        init_if_needed,
        payer = payer,
        space = 8 + GuardianSignatures::compute_size(usize::from(total_signatures))
    )]
    guardian_signatures: Account<'info, GuardianSignatures>,

    system_program: Program<'info, System>,
}

impl<'info> PostSignatures<'info> {
    pub fn constraints(guardian_signatures: &Vec<[u8; 66]>) -> Result<()> {
        // Signatures should not be empty, since this is used by is_initialized.
        // Additionally, there is no reason for it to be.
        require!(
            !guardian_signatures.is_empty(),
            WormholeVaaVerificationComparisonError::EmptyGuardianSignatures
        );

        // Done.
        Ok(())
    }
}

/// Creates or appends to a GuardianSignatures account for subsequent use by consume_vaa.
/// This is necessary as the Wormhole query response (220 bytes)
/// and 13 guardian signatures (a quorum of the current 19 mainnet guardians, 66 bytes each)
/// alongside the required accounts is larger than the transaction size limit on Solana (1232 bytes).
///
/// This instruction allows for the initial payer to append additional signatures to the account by calling the instruction again.
/// This may be necessary if a quorum of signatures from the current guardian set grows larger than can fit into a single transaction.
///
/// The GuardianSignatures account can be closed by anyone with a successful consume_vaa instruction
/// or by the initial payer via close_signatures, either of which will refund the initial payer.
#[access_control(PostSignatures::constraints(&guardian_signatures))]
pub fn post_signatures(
    ctx: Context<PostSignatures>,
    mut guardian_signatures: Vec<[u8; 66]>,
    _total_signatures: u8,
) -> Result<()> {
    if ctx.accounts.guardian_signatures.is_initialized() {
        require_eq!(
            ctx.accounts.guardian_signatures.refund_recipient,
            ctx.accounts.payer.key(),
            WormholeVaaVerificationComparisonError::WriteAuthorityMismatch
        );
        ctx.accounts
            .guardian_signatures
            .guardian_signatures
            .append(&mut guardian_signatures);
    } else {
        ctx.accounts
            .guardian_signatures
            .set_inner(GuardianSignatures {
                refund_recipient: ctx.accounts.payer.key(),
                guardian_signatures,
            });
    }
    // Done.
    Ok(())
}
