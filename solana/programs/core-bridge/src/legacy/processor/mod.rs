mod governance;
mod post_message;
mod post_message_unreliable;
mod post_vaa;
mod verify_signatures;

pub use governance::*;
pub use post_message::*;
pub use post_message_unreliable::*;
pub use post_vaa::*;
pub use verify_signatures::*;

use crate::ID;
use anchor_lang::prelude::*;
use wormhole_solana_common::process_anchorized_legacy_instruction;

use super::instruction::LegacyInstruction;

pub fn process_legacy_instruction(
    program_id: &Pubkey,
    mut account_infos: &[AccountInfo],
    ix_data: &[u8],
) -> Result<()> {
    // TODO: This may not be necessary. Double-check in integration test.
    require!(*program_id == ID, ErrorCode::DeclaredProgramIdMismatch);

    // Deserialize instruction data. The data should match the instruction
    // enum. Otherwise, we bail out.
    match LegacyInstruction::try_from_slice(ix_data)? {
        LegacyInstruction::Initialize(_) => err!(ErrorCode::Deprecated),
        LegacyInstruction::PostMessage(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyPostMessage",
            PostMessage,
            account_infos,
            ix_data,
            post_message,
            args
        ),
        LegacyInstruction::PostVaa(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyPostVaa",
            PostVaa,
            account_infos,
            ix_data,
            post_vaa,
            args
        ),
        LegacyInstruction::SetMessageFee(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacySetMessageFee",
            SetMessageFee,
            account_infos,
            ix_data,
            set_message_fee,
            args
        ),
        LegacyInstruction::TransferFees(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyTransferFees",
            TransferFees,
            account_infos,
            ix_data,
            transfer_fees,
            args
        ),
        LegacyInstruction::UpgradeContract(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyUpgradeContract",
            UpgradeContract,
            account_infos,
            ix_data,
            upgrade_contract,
            args
        ),
        LegacyInstruction::GuardianSetUpdate(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyGuardianSetUpdate",
            GuardianSetUpdate,
            account_infos,
            ix_data,
            guardian_set_update,
            args
        ),
        LegacyInstruction::VerifySignatures(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyVerifySignatures",
            VerifySignatures,
            account_infos,
            ix_data,
            verify_signatures,
            args
        ),
        LegacyInstruction::PostMessageUnreliable(args) => process_anchorized_legacy_instruction!(
            ID,
            "LegacyPostMessageUnreliable",
            PostMessageUnreliable,
            account_infos,
            ix_data,
            post_message_unreliable,
            args
        ),
    }
}
