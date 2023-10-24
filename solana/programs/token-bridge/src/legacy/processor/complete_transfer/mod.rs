mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter, utils::fix_account_order};
use anchor_lang::prelude::*;
use core_bridge_program::{
    legacy::utils::LegacyAnchorized,
    sdk::{self as core_bridge_sdk, LoadZeroCopy},
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

pub fn validate_token_transfer_vaa(
    vaa_acc_info: &AccountInfo,
    registered_emitter: &Account<LegacyAnchorized<0, RegisteredEmitter>>,
    recipient_token: &AccountInfo,
    recipient: &Option<AccountInfo>,
) -> Result<(u16, [u8; 32])> {
    let vaa_key = vaa_acc_info.key();
    let vaa = core_bridge_sdk::VaaAccount::load(vaa_acc_info)?;
    let msg = crate::utils::require_valid_token_bridge_vaa(&vaa_key, &vaa, registered_emitter)?;

    let transfer = if let TokenBridgeMessage::Transfer(inner) = msg {
        inner
    } else {
        return err!(TokenBridgeError::InvalidTokenBridgeVaa);
    };

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.recipient_chain(),
        core_bridge_sdk::SOLANA_CHAIN,
        TokenBridgeError::RecipientChainNotSolana
    );

    // The encoded transfer recipient can either be a token account or the owner of an ATA.
    //
    // NOTE: Allowing the encoded transfer recipient to be an ATA's owner is a patch.
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

        // Finally verify that the token account is an ATA belonging to the expected recipient.
        let token_account = crate::zero_copy::TokenAccount::load(recipient_token)?;
        require_keys_eq!(
            recipient_token.key(),
            anchor_spl::associated_token::get_associated_token_address(
                &expected_recipient,
                &token_account.mint()
            ),
            TokenBridgeError::InvalidAssociatedTokenAccount
        );
    }

    // Done.
    Ok((transfer.token_chain(), transfer.token_address()))
}

/// The Anchor context orders the accounts as:
///
/// 1.  `payer`
/// 2.  `config`
/// 3.  `vaa`
/// 4.  `claim`
/// 5.  `registered_emitter`
/// 6.  `recipient_token`
/// 7.  `payer_token`
/// 8.  `custody_token`     OR `wrapped_mint`
/// 9.  `mint`              OR `wrapped_asset`
/// 10. `custody_authority` OR `mint_authority`
/// 11. `recipient`          <-- order unspecified
/// 12. `system_program`     <-- order unspecified
/// 13. `token_program`      <-- order unspecified
///
/// Because the legacy implementation did not require specifying where the recipient, System program
/// and SPL token program should be, we ensure that these accounts are 11, 12 and 13 respectively
/// because the Anchor account context requires them to be in these positions.
///
/// NOTE: Recipient as account #11 is optional based on whether the VAA's encoded recipient is the
/// owner of the `recipient_token` account. See the complete transfer account contexts for more
/// info.
pub fn order_complete_transfer_account_infos<'info>(
    account_infos: &[AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    fix_account_order(
        account_infos,
        10,       // start_index
        11,       // system_program_index
        Some(12), // token_program_index
        None,     // core_bridge_program_index
    )
}
