mod attest_token;
mod complete_transfer;
mod complete_transfer_with_payload;
mod create_or_update_wrapped;
mod governance;
mod transfer_tokens;
mod transfer_tokens_with_payload;

pub use attest_token::*;
pub use complete_transfer::*;
pub use complete_transfer_with_payload::*;
pub use create_or_update_wrapped::*;
pub use governance::*;
pub use transfer_tokens::*;
pub use transfer_tokens_with_payload::*;

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
        LegacyInstruction::AttestToken(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyAttestToken",
                AttestToken,
                account_infos,
                ix_data,
                attest_token,
                args
            )
        }
        LegacyInstruction::CompleteTransferNative(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferNative",
                CompleteTransferNative,
                account_infos,
                ix_data,
                complete_transfer_native,
                args
            )
        }
        LegacyInstruction::CompleteTransferWrapped(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWrapped",
                CompleteTransferWrapped,
                account_infos,
                ix_data,
                complete_transfer_wrapped,
                args
            )
        }
        LegacyInstruction::TransferTokensWrapped(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWrapped",
                TransferTokensWrapped,
                account_infos,
                ix_data,
                transfer_tokens_wrapped,
                args
            )
        }
        LegacyInstruction::TransferTokensNative(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensNative",
                TransferTokensNative,
                account_infos,
                ix_data,
                transfer_tokens_native,
                args
            )
        }
        LegacyInstruction::RegisterChain(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyRegisterChain",
                RegisterChain,
                account_infos,
                ix_data,
                register_chain,
                args
            )
        }
        LegacyInstruction::CreateOrUpdateWrapped(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCreateOrUpdateWrapped",
                CreateOrUpdateWrapped,
                account_infos,
                ix_data,
                create_or_update_wrapped,
                args
            )
        }
        LegacyInstruction::UpgradeContract(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyUpgradeContract",
                UpgradeContract,
                account_infos,
                ix_data,
                upgrade_contract,
                args
            )
        }
        LegacyInstruction::CompleteTransferWithPayloadNative(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWithPayloadNative",
                CompleteTransferWithPayloadNative,
                account_infos,
                ix_data,
                complete_transfer_with_payload_native,
                args
            )
        }
        LegacyInstruction::CompleteTransferWithPayloadWrapped(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyCompleteTransferWithPayloadWrapped",
                CompleteTransferWithPayloadWrapped,
                account_infos,
                ix_data,
                complete_transfer_with_payload_wrapped,
                args
            )
        }
        LegacyInstruction::TransferTokensWithPayloadWrapped(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWithPayloadWrapped",
                TransferTokensWithPayloadWrapped,
                account_infos,
                ix_data,
                transfer_tokens_with_payload_wrapped,
                args
            )
        }
        LegacyInstruction::TransferTokensWithPayloadNative(args) => {
            process_anchorized_legacy_instruction!(
                ID,
                "LegacyTransferTokensWithPayloadNative",
                TransferTokensWithPayloadNative,
                account_infos,
                ix_data,
                transfer_tokens_with_payload_native,
                args
            )
        }
    }
}
