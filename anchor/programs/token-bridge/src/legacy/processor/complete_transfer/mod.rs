mod native;
mod wrapped;

pub use native::*;
pub use wrapped::*;

use crate::{
    legacy::state::RegisteredEmitter,
    message::{require_valid_token_bridge_posted_vaa, TokenTransfer},
};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, TokenAccount};
use core_bridge_program::{
    state::PostedVaaV1,
    types::{ChainId, SolanaChain},
};

pub fn validate_token_transfer<'ctx, 'info>(
    posted_token_transfer: &'ctx Account<'info, PostedVaaV1<TokenTransfer>>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
    mint: &'ctx Account<'info, Mint>,
    dst_token: &'ctx Account<'info, TokenAccount>,
) -> Result<&'ctx TokenTransfer> {
    let transfer_msg =
        require_valid_token_bridge_posted_vaa(posted_token_transfer, registered_emitter)?;

    // Mint account must agree with the encoded token address.
    require_eq!(Pubkey::from(transfer_msg.token_address), mint.key());

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(transfer_msg.recipient_chain, ChainId::from(SolanaChain));

    // The encoded transfer recipient can either be a token account or the owner of the token
    // account passed into this account context.
    //
    // NOTE: Allowing the encoded transfer recipient to be the token account's owner is a patch.
    let recipient = Pubkey::from(transfer_msg.recipient);
    if recipient != dst_token.key() {
        require_keys_eq!(recipient, dst_token.owner)
    }

    // Done.
    Ok(transfer_msg)
}
