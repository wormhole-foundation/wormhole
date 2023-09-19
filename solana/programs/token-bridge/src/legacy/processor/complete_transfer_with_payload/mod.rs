mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, legacy::state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::{
    legacy::utils::LegacyAnchorized,
    sdk::{self as core_bridge_sdk, LoadZeroCopy},
};
use wormhole_raw_vaas::token_bridge::TokenBridgeMessage;

pub fn validate_posted_token_transfer_with_payload(
    vaa_acc_info: &AccountInfo,
    registered_emitter: &Account<LegacyAnchorized<0, RegisteredEmitter>>,
    redeemer_authority: &Signer,
    dst_token: &AccountInfo,
) -> Result<(u16, [u8; 32])> {
    let vaa_key = vaa_acc_info.key();
    let vaa = core_bridge_sdk::VaaAccount::load(vaa_acc_info)?;
    let msg =
        crate::utils::require_valid_posted_token_bridge_vaa(&vaa_key, &vaa, registered_emitter)?;

    let transfer = if let TokenBridgeMessage::TransferWithMessage(inner) = msg {
        inner
    } else {
        return err!(TokenBridgeError::InvalidTokenBridgeVaa);
    };

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.redeemer_chain(),
        core_bridge_sdk::SOLANA_CHAIN,
        TokenBridgeError::RedeemerChainNotSolana
    );

    // The encoded transfer recipient can either be the signer of this instruction or a
    // program whose signer is a PDA using the seeds [b"redeemer"] (and the encoded redeemer
    // is the program ID). If the latter, the transfer redeemer can be any PDA that signs
    // for this instruction.
    //
    // NOTE: Requiring that the transfer redeemer be a signer is a patch.
    let redeemer = Pubkey::from(transfer.redeemer());
    let redeemer_authority = redeemer_authority.key();
    if redeemer != redeemer_authority {
        let (expected_authority, _) = Pubkey::find_program_address(
            &[crate::constants::PROGRAM_REDEEMER_SEED_PREFIX],
            &redeemer,
        );
        require_keys_eq!(
            redeemer_authority,
            expected_authority,
            TokenBridgeError::InvalidProgramRedeemer
        )
    } else {
        // The redeemer must be the token account owner if the redeemer authority is the
        // same as the redeemer (i.e. the signer of this transaction, which does not
        // represent a program's PDA.
        let token = crate::zero_copy::TokenAccount::load(dst_token)?;
        require_keys_eq!(redeemer, token.owner(), ErrorCode::ConstraintTokenOwner);
    }

    // Done.
    Ok((transfer.token_chain(), transfer.token_address()))
}

pub fn order_complete_transfer_with_payload_account_infos<'info>(
    account_infos: &[AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    const NUM_ACCOUNTS: usize = 14;
    const TOKEN_PROGRAM_IDX: usize = NUM_ACCOUNTS - 1;
    const SYSTEM_PROGRAM_IDX: usize = TOKEN_PROGRAM_IDX - 1;

    let mut infos = account_infos.to_vec();

    // This check is inclusive because System program and Token program can be in any order.
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
