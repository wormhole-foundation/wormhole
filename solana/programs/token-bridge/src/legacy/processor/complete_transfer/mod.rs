mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use anchor_spl::token::TokenAccount;
use core_bridge_program::{constants::SOLANA_CHAIN, state::PostedVaaV1Bytes};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

pub fn validate_token_transfer<'ctx, 'info>(
    posted_vaa: &'ctx Account<'info, PostedVaaV1Bytes>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
    dst_token: &'ctx Account<'info, TokenAccount>,
) -> Result<(u16, [u8; 32])> {
    let payload =
        crate::utils::require_valid_token_bridge_posted_vaa(posted_vaa, registered_emitter)?;
    let msg =
        TokenBridgeMessage::parse(&payload).map_err(|_| TokenBridgeError::CannotParseMessage)?;
    match msg.transfer() {
        Some(transfer) => {
            // This token bridge transfer must be intended to be redeemed on Solana.
            require_eq!(
                transfer.recipient_chain(),
                SOLANA_CHAIN,
                TokenBridgeError::RecipientChainNotSolana
            );

            // The encoded transfer recipient can either be a token account or the owner of the token
            // account passed into this account context.
            //
            // NOTE: Allowing the encoded transfer recipient to be the token account's owner is a patch.
            let recipient = Pubkey::from(transfer.recipient());
            if recipient != dst_token.key() {
                require_keys_eq!(recipient, dst_token.owner, ErrorCode::ConstraintTokenOwner);
            }

            // Done.
            Ok((transfer.token_chain(), transfer.token_address()))
        }
        None => {
            err!(TokenBridgeError::InvalidTokenBridgeVaa)
        }
    }
}
