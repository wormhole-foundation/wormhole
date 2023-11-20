mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter, utils::fix_account_order};
use anchor_lang::prelude::*;
use core_bridge_program::sdk as core_bridge;
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

pub fn validate_token_transfer_vaa(
    vaa_acc_info: &AccountInfo,
    registered_emitter: &Account<core_bridge::legacy::LegacyAnchorized<RegisteredEmitter>>,
    recipient_token: &Account<anchor_spl::token::TokenAccount>,
) -> Result<(u16, [u8; 32])> {
    let vaa_key = vaa_acc_info.key();
    let vaa = core_bridge::VaaAccount::load(vaa_acc_info)?;
    let msg =
        crate::utils::vaa::require_valid_token_bridge_vaa(&vaa_key, &vaa, registered_emitter)?;

    let transfer = if let TokenBridgeMessage::Transfer(inner) = msg {
        inner
    } else {
        return err!(TokenBridgeError::InvalidTokenBridgeVaa);
    };

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.recipient_chain(),
        core_bridge::SOLANA_CHAIN,
        TokenBridgeError::RecipientChainNotSolana
    );

    // The encoded transfer recipient can only be a token account.
    require_keys_eq!(
        recipient_token.key(),
        Pubkey::from(transfer.recipient()),
        TokenBridgeError::InvalidRecipient
    );

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
