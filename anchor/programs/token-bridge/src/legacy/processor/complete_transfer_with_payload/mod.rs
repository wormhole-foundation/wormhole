mod native;
mod wrapped;

pub use native::*;
pub use wrapped::*;

use crate::{
    legacy::state::RegisteredEmitter,
    message::{require_valid_token_bridge_posted_vaa, TokenTransferWithPayload},
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, TokenAccount};
use core_bridge_program::{
    state::PostedVaaV1,
    types::{ChainId, SolanaChain},
};

pub fn validate_token_transfer_with_payload<'ctx, 'info>(
    posted_token_transfer: &'ctx Account<'info, PostedVaaV1<TokenTransferWithPayload>>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
    mint: &'ctx Account<'info, Mint>,
    redeemer_authority: &'ctx Signer<'info>,
    redeemer_token: &'ctx Account<'info, TokenAccount>,
) -> Result<&'ctx TokenTransferWithPayload> {
    let transfer_msg =
        require_valid_token_bridge_posted_vaa(posted_token_transfer, registered_emitter)?;

    // Mint account must agree with the encoded token address.
    require_eq!(Pubkey::from(transfer_msg.token_address), mint.key());

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(transfer_msg.redeemer_chain, ChainId::from(SolanaChain));

    // The encoded transfer recipient can either be the signer of this instruction or a program
    // whose signer is a PDA using the seeds [b"redeemer"] (and the encoded redeemer is the program
    // ID). If the latter, the transfer redeemer can be any PDA that signs for this instruction.
    //
    // NOTE: Requiring that the transfer redeemer be a signer is a patch.
    let redeemer = Pubkey::from(transfer_msg.redeemer);
    if redeemer != redeemer_authority.key() {
        let (redeemer_pda, _) = Pubkey::find_program_address(&[b"redeemer"], &redeemer);
        require_keys_eq!(redeemer, redeemer_pda)
    }

    // Token account owner must be either the VAA-specified recipient, or the redeemer account (for
    // regular wallets, these two are equal, for programs the latter is a PDA).
    let token_owner = redeemer_token.owner;
    require!(
        redeemer == token_owner || redeemer_authority.key() == token_owner,
        ErrorCode::ConstraintTokenOwner
    );

    // Done.
    Ok(transfer_msg)
}
