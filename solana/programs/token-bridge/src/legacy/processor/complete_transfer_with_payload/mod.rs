mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, legacy::state::RegisteredEmitter};
use anchor_lang::prelude::*;
use core_bridge_program::{
    constants::SOLANA_CHAIN, legacy::utils::LegacyAnchorized, zero_copy::PostedVaaV1,
};
use wormhole_raw_vaas::token_bridge::{TokenBridgeMessage, TransferWithMessage};

pub fn validate_posted_token_transfer_with_payload<'ctx>(
    vaa_acc_key: &'ctx Pubkey,
    vaa_acc_data: &'ctx [u8],
    registered_emitter: &'ctx Account<'_, LegacyAnchorized<0, RegisteredEmitter>>,
    redeemer_authority: &'ctx Signer<'_>,
    dst_token: &'ctx AccountInfo<'_>,
) -> Result<TransferWithMessage<'ctx>> {
    let vaa = PostedVaaV1::parse(vaa_acc_data)?;
    let msg =
        crate::utils::require_valid_posted_token_bridge_vaa(vaa_acc_key, &vaa, registered_emitter)?;

    let transfer = match msg {
        TokenBridgeMessage::TransferWithMessage(inner) => inner,
        _ => return err!(TokenBridgeError::InvalidTokenBridgeVaa),
    };

    // This token bridge transfer must be intended to be redeemed on Solana.
    require_eq!(
        transfer.redeemer_chain(),
        SOLANA_CHAIN,
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
        require_keys_eq!(
            redeemer,
            crate::zero_copy::TokenAccount::parse(&dst_token.try_borrow_data()?)?.owner(),
            ErrorCode::ConstraintTokenOwner
        );
    }

    // Done.
    Ok(transfer)
}
