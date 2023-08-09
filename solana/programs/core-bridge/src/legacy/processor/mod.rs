mod governance;
pub use governance::*;

mod initialize;
pub use initialize::*;

mod post_message;
pub use post_message::*;

mod post_message_unreliable;
pub use post_message_unreliable::*;

mod post_vaa;
pub use post_vaa::*;

mod verify_signatures;
pub use verify_signatures::*;

use crate::{
    legacy::instruction::{
        EmptyArgs, LegacyInitializeArgs, LegacyPostMessageArgs, LegacyPostMessageUnreliableArgs,
        LegacyPostVaaArgs, LegacyVerifySignaturesArgs,
    },
    ID,
};
use anchor_lang::prelude::*;
use wormhole_solana_common::process_anchorized_legacy_instruction;

use super::instruction::LegacyInstruction;

pub fn process_legacy_instruction(
    program_id: &Pubkey,
    mut account_infos: &[AccountInfo],
    mut ix_data: &[u8],
) -> Result<()> {
    // TODO: This may not be necessary. Double-check in integration test.
    require!(*program_id == ID, ErrorCode::DeclaredProgramIdMismatch);

    // Deserialize instruction data. The data should match the instruction
    // enum. Otherwise, we bail out.
    match LegacyInstruction::deserialize(&mut ix_data)? {
        LegacyInstruction::Initialize => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyInitialize",
                Initialize,
                account_infos,
                ix_data,
                initialize,
                LegacyInitializeArgs
            )
        }
        LegacyInstruction::PostMessage => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyPostMessage",
                PostMessage,
                account_infos,
                ix_data,
                post_message,
                LegacyPostMessageArgs
            )
        }
        LegacyInstruction::PostVaa => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyPostVaa",
                PostVaa,
                account_infos,
                ix_data,
                post_vaa,
                LegacyPostVaaArgs
            )
        }
        LegacyInstruction::SetMessageFee => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacySetMessageFee",
                SetMessageFee,
                account_infos,
                ix_data,
                set_message_fee,
                EmptyArgs
            )
        }
        LegacyInstruction::TransferFees => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferFees",
                TransferFees,
                account_infos,
                ix_data,
                transfer_fees,
                EmptyArgs
            )
        }
        LegacyInstruction::UpgradeContract => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyUpgradeContract",
                UpgradeContract,
                account_infos,
                ix_data,
                upgrade_contract,
                EmptyArgs
            )
        }
        LegacyInstruction::GuardianSetUpdate => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyGuardianSetUpdate",
                GuardianSetUpdate,
                account_infos,
                ix_data,
                guardian_set_update,
                EmptyArgs
            )
        }
        LegacyInstruction::VerifySignatures => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyVerifySignatures",
                VerifySignatures,
                account_infos,
                ix_data,
                verify_signatures,
                LegacyVerifySignaturesArgs
            )
        }
        LegacyInstruction::PostMessageUnreliable => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyPostMessageUnreliable",
                PostMessageUnreliable,
                account_infos,
                ix_data,
                post_message_unreliable,
                LegacyPostMessageUnreliableArgs
            )
        }
    }
}
