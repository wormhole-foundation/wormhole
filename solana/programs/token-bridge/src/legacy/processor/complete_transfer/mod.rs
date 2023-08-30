mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::{constants::SOLANA_CHAIN, zero_copy::PostedVaaV1};
use wormhole_raw_vaas::token_bridge::{TokenBridgeMessage, Transfer};

pub fn validate_posted_token_transfer<'ctx>(
    vaa_acc_key: &'ctx Pubkey,
    vaa_acc_data: &'ctx [u8],
    registered_emitter: &'ctx Account<'_, RegisteredEmitter>,
    recipient_token: &'ctx AccountInfo<'_>,
) -> Result<Transfer<'ctx>> {
    let vaa = PostedVaaV1::parse(vaa_acc_data)?;
    let msg =
        crate::utils::require_valid_posted_token_bridge_vaa(vaa_acc_key, &vaa, registered_emitter)?;

    let transfer = match msg {
        TokenBridgeMessage::Transfer(inner) => inner,
        _ => return err!(TokenBridgeError::InvalidTokenBridgeVaa),
    };

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.recipient_chain(),
        SOLANA_CHAIN,
        TokenBridgeError::RecipientChainNotSolana
    );

    // The encoded transfer recipient can either be a token account or the owner of the
    // token account passed into this account context.
    //
    // NOTE: Allowing the encoded transfer recipient to be the token account's owner is a
    // patch.
    let recipient = Pubkey::from(transfer.recipient());
    if recipient != recipient_token.key() {
        crate::zero_copy::TokenAccount::require_owner(
            &recipient_token.try_borrow_data()?,
            &recipient,
        )?;
    }

    // Done.
    Ok(transfer)
}
