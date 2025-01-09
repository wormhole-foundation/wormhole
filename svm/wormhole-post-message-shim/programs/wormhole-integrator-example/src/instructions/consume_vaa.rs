declare_program!(wormhole_verify_vaa_shim);

use anchor_lang::{
    prelude::*,
    solana_program::{self, keccak},
};
use wormhole_verify_vaa_shim::cpi::accounts::VerifyVaa;
use wormhole_verify_vaa_shim::program::WormholeVerifyVaaShim;

#[derive(Accounts)]
pub struct ConsumeVaa<'info> {
    /// CHECK: Guardian set used for signature verification by shim.
    /// Derivation is checked by the shim.
    guardian_set: UncheckedAccount<'info>,

    /// CHECK: Stores guardian signatures to be verified by shim.
    guardian_signatures: UncheckedAccount<'info>,

    wormhole_verify_vaa_shim: Program<'info, WormholeVerifyVaaShim>,
}

pub fn consume_vaa(ctx: Context<ConsumeVaa>, vaa_body: Vec<u8>) -> Result<()> {
    // Compute the message hash.
    let message_hash = &solana_program::keccak::hashv(&[&vaa_body]).to_bytes();
    let digest = keccak::hash(message_hash.as_slice()).to_bytes();
    wormhole_verify_vaa_shim::cpi::verify_vaa(
        CpiContext::new(
            ctx.accounts.wormhole_verify_vaa_shim.to_account_info(),
            VerifyVaa {
                guardian_set: ctx.accounts.guardian_set.to_account_info(),
                guardian_signatures: ctx.accounts.guardian_signatures.to_account_info(),
            },
        ),
        digest,
    )?;
    Ok(())
}
