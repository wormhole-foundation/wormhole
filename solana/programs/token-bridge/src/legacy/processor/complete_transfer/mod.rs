mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, state::RegisteredEmitter};
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
        let token_account = crate::zero_copy::TokenAccount::load(recipient_token)?;
        require_keys_eq!(
            token_account.owner(),
            expected_recipient,
            ErrorCode::ConstraintTokenOwner
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
    const NUM_ACCOUNTS: usize = 13;
    const TOKEN_PROGRAM_IDX: usize = NUM_ACCOUNTS - 1;
    const SYSTEM_PROGRAM_IDX: usize = TOKEN_PROGRAM_IDX - 1;

    let mut infos = account_infos.to_vec();
    if infos.len() >= NUM_ACCOUNTS {
        // System program needs to exist in these account infos.
        let system_program_idx = infos
            .iter()
            .position(|info| info.key() == anchor_lang::system_program::ID)
            .ok_or(error!(ErrorCode::InvalidProgramId))?;

        // Make sure System program is in the right index.
        if system_program_idx != SYSTEM_PROGRAM_IDX {
            infos.swap(SYSTEM_PROGRAM_IDX, system_program_idx);
        }

        // Token program needs to exist in these account infos.
        let token_program_idx = infos
            .iter()
            .position(|info| info.key() == anchor_spl::token::ID)
            .ok_or(error!(ErrorCode::InvalidProgramId))?;

        // Make sure Token program is in the right index.
        if token_program_idx != TOKEN_PROGRAM_IDX {
            infos.swap(TOKEN_PROGRAM_IDX, token_program_idx);
        }
    }

    // Done.
    Ok(infos)
}
