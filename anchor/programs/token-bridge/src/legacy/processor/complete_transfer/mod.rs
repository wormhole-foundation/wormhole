mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use anchor_spl::token::{Mint, TokenAccount};
use core_bridge_program::{constants::SOLANA_CHAIN, state::PostedVaaV1};
use wormhole_vaas::payloads::token_bridge::Transfer;

pub fn validate_token_transfer<'ctx, 'info>(
    posted_token_transfer: &'ctx Account<'info, PostedVaaV1<Transfer>>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
    mint: &'ctx Account<'info, Mint>,
    dst_token: &'ctx Account<'info, TokenAccount>,
) -> Result<u16> {
    let transfer = crate::utils::require_valid_token_bridge_posted_vaa(
        posted_token_transfer,
        registered_emitter,
    )?;

    // Mint account must agree with the encoded token address.
    require_eq!(Pubkey::from(transfer.token_address.0), mint.key());

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.recipient_chain,
        SOLANA_CHAIN,
        TokenBridgeError::RecipientChainNotSolana
    );

    // The encoded transfer recipient can either be a token account or the owner of the token
    // account passed into this account context.
    //
    // NOTE: Allowing the encoded transfer recipient to be the token account's owner is a patch.
    let recipient = Pubkey::from(transfer.recipient.0);
    if recipient != dst_token.key() {
        require_keys_eq!(recipient, dst_token.owner, ErrorCode::ConstraintTokenOwner);
    }

    // Done.
    Ok(transfer.token_chain)
}
