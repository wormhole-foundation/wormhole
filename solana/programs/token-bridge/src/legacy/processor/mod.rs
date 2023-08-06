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

use anchor_lang::prelude::*;
use core_bridge_program::legacy::utils::ProcessLegacyInstruction;

use super::instruction::LegacyInstruction;

pub fn process_legacy_instruction(
    program_id: &Pubkey,
    account_infos: &[AccountInfo],
    mut ix_data: &[u8],
) -> Result<()> {
    // TODO: This may not be necessary. Double-check in integration test.
    require!(
        *program_id == crate::ID,
        ErrorCode::DeclaredProgramIdMismatch
    );

    // Deserialize instruction data. The data should match the instruction
    // enum. Otherwise, we bail out.
    match LegacyInstruction::deserialize(&mut ix_data)? {
        LegacyInstruction::Initialize => {
            Initialize::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::AttestToken => {
            AttestToken::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::CompleteTransferNative => {
            CompleteTransferNative::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::CompleteTransferWrapped => {
            CompleteTransferWrapped::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::TransferTokensWrapped => {
            TransferTokensWrapped::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::TransferTokensNative => {
            TransferTokensNative::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::RegisterChain => err!(ErrorCode::Deprecated),
        LegacyInstruction::CreateOrUpdateWrapped => {
            CreateOrUpdateWrapped::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::UpgradeContract => {
            UpgradeContract::process_instruction(program_id, account_infos, ix_data)
        }
        LegacyInstruction::CompleteTransferWithPayloadNative => {
            CompleteTransferWithPayloadNative::process_instruction(
                program_id,
                account_infos,
                ix_data,
            )
        }
        LegacyInstruction::CompleteTransferWithPayloadWrapped => {
            CompleteTransferWithPayloadWrapped::process_instruction(
                program_id,
                account_infos,
                ix_data,
            )
        }
        LegacyInstruction::TransferTokensWithPayloadWrapped => {
            TransferTokensWithPayloadWrapped::process_instruction(
                program_id,
                account_infos,
                ix_data,
            )
        }
        LegacyInstruction::TransferTokensWithPayloadNative => {
            TransferTokensWithPayloadNative::process_instruction(program_id, account_infos, ix_data)
        }
    }
}
