mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, zero_copy::PostedVaaV1,
};
use wormhole_raw_vaas::token_bridge::{TokenBridgeMessage, Transfer};

pub fn validate_posted_token_transfer<'ctx>(
    vaa_acc_key: &'ctx Pubkey,
    vaa_acc_data: &'ctx [u8],
    registered_emitter: &'ctx Account<'_, LegacyAnchorized<0, RegisteredEmitter>>,
    recipient_token: &'ctx AccountInfo<'_>,
    recipient: &'ctx Option<AccountInfo>,
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
    let expected_recipient = Pubkey::from(transfer.recipient());
    if expected_recipient != recipient_token.key() {
        require!(recipient.is_some(), ErrorCode::ConstraintTokenOwner);

        let recipient = recipient.as_ref().unwrap();

        // If the owner of this token account is the token program, someone is trolling by calling
        // the complete transfer instruction with an ATA owned by the encoded recipient,
        // which itself is an ATA. Go away.
        require_keys_neq!(
            *recipient.owner,
            anchor_spl::token::ID,
            TokenBridgeError::NestedTokenAccount
        );

        // Check that the recipient provided is the expected recipient.
        //
        // NOTE: If the provided account is the rent sysvar, this means an integrator is calling
        // complete transfer with an old integration but expecting the redemption to be successful
        // if the encoded VAA recipient is a token account owner. It is expected for integrators to
        // handle this case by providing the recipient account info instead of the rent sysvar if he
        // expects to redeem transfers to token accounts whose owner is encoded in the VAA.
        require_keys_eq!(
            recipient.key(),
            expected_recipient,
            TokenBridgeError::InvalidRecipient
        );

        // Finally check that the owner of the recipient token account is the encoded recipient.
        crate::zero_copy::TokenAccount::require_owner(
            &recipient_token.try_borrow_data()?,
            &expected_recipient,
        )?;
    }

    // Done.
    Ok(transfer)
}
