mod attest_token;
pub use attest_token::*;

mod complete_transfer;
pub use complete_transfer::*;

mod complete_transfer_with_payload;
pub use complete_transfer_with_payload::*;

mod create_or_update_wrapped;
pub use create_or_update_wrapped::*;

mod governance;
pub use governance::*;

mod initialize;
pub use initialize::*;

mod transfer_tokens;
pub use transfer_tokens::*;

mod transfer_tokens_with_payload;
pub use transfer_tokens_with_payload::*;

use crate::{
    legacy::instruction::{
        EmptyArgs, LegacyAttestTokenArgs, LegacyInitializeArgs, LegacyTransferTokensArgs,
        LegacyTransferTokensWithPayloadArgs,
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
        LegacyInstruction::AttestToken => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyAttestToken",
                AttestToken,
                account_infos,
                ix_data,
                attest_token,
                LegacyAttestTokenArgs
            )
        }
        LegacyInstruction::CompleteTransferNative => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferNative",
                CompleteTransferNative,
                account_infos,
                ix_data,
                complete_transfer_native,
                EmptyArgs
            )
        }
        LegacyInstruction::CompleteTransferWrapped => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWrapped",
                CompleteTransferWrapped,
                account_infos,
                ix_data,
                complete_transfer_wrapped,
                EmptyArgs
            )
        }
        LegacyInstruction::TransferTokensWrapped => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWrapped",
                TransferTokensWrapped,
                account_infos,
                ix_data,
                transfer_tokens_wrapped,
                LegacyTransferTokensArgs
            )
        }
        LegacyInstruction::TransferTokensNative => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensNative",
                TransferTokensNative,
                account_infos,
                ix_data,
                transfer_tokens_native,
                LegacyTransferTokensArgs
            )
        }
        LegacyInstruction::RegisterChain => err!(ErrorCode::Deprecated),
        LegacyInstruction::CreateOrUpdateWrapped => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCreateOrUpdateWrapped",
                CreateOrUpdateWrapped,
                account_infos,
                ix_data,
                create_or_update_wrapped,
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
        LegacyInstruction::CompleteTransferWithPayloadNative => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWithPayloadNative",
                CompleteTransferWithPayloadNative,
                account_infos,
                ix_data,
                complete_transfer_with_payload_native,
                EmptyArgs
            )
        }
        LegacyInstruction::CompleteTransferWithPayloadWrapped => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWithPayloadWrapped",
                CompleteTransferWithPayloadWrapped,
                account_infos,
                ix_data,
                complete_transfer_with_payload_wrapped,
                EmptyArgs
            )
        }
        LegacyInstruction::TransferTokensWithPayloadWrapped => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWithPayloadWrapped",
                TransferTokensWithPayloadWrapped,
                account_infos,
                ix_data,
                transfer_tokens_with_payload_wrapped,
                LegacyTransferTokensWithPayloadArgs
            )
        }
        LegacyInstruction::TransferTokensWithPayloadNative => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWithPayloadNative",
                TransferTokensWithPayloadNative,
                account_infos,
                ix_data,
                transfer_tokens_with_payload_native,
                LegacyTransferTokensWithPayloadArgs
            )
        }
    }
}
