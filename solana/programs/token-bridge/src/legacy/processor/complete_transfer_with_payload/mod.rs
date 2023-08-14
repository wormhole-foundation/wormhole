mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use crate::{error::TokenBridgeError, legacy::state::RegisteredEmitter};
use anchor_lang::prelude::*;
use anchor_spl::token::TokenAccount;
use core_bridge_program::{constants::SOLANA_CHAIN, state::PostedVaaV1Bytes};

pub fn validate_token_transfer_with_payload<'ctx, 'info>(
    posted_vaa: &'ctx Account<'info, PostedVaaV1Bytes>,
    registered_emitter: &'ctx Account<'info, RegisteredEmitter>,
    redeemer_authority: &'ctx Signer<'info>,
    recipient_token: &'ctx Account<'info, TokenAccount>,
) -> Result<(u16, [u8; 32])> {
    let msg = crate::utils::require_valid_token_bridge_posted_vaa(posted_vaa, registered_emitter)?;
    match msg.transfer_with_message() {
        Some(transfer) => {
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
                let (pda, _) = Pubkey::find_program_address(&[b"redeemer"], &redeemer);
                require_keys_eq!(
                    redeemer_authority,
                    pda,
                    TokenBridgeError::InvalidProgramRedeemer
                )
            } else {
                // The redeemer must be the token account owner if the redeemer authority is the
                // same as the redeemer (i.e. the signer of this transaction, which does not
                // represent a program's PDA.
                require!(
                    redeemer == recipient_token.owner,
                    ErrorCode::ConstraintTokenOwner
                );
            }

            // Done.
            Ok((transfer.token_chain(), transfer.token_address()))
        }
        None => {
            err!(TokenBridgeError::InvalidTokenBridgeVaa)
        }
    }
}
