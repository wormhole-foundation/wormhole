//! Wormhole External Entity Verifiable Action Approval handling
//! program. Processes instructions related to arbitrary cross-chain
//! messaging.

#[macro_use]
extern crate solana_program;

pub mod error;
pub mod instruction;
pub mod state;

use solana_program::{
    account_info::AccountInfo, entrypoint, entrypoint::ProgramResult, pubkey::Pubkey,
};

use crate::instruction::EEVAAInstruction;

#[cfg(not(feature = "no-entrypoint"))]
entrypoint!(process_instruction);
fn process_instruction<'a>(
    _program_id: &Pubkey,
    _accounts: &'a [AccountInfo<'a>],
    instruction_data: &[u8],
) -> ProgramResult {
    EEVAAInstruction::deserialize(instruction_data).map_err(|e| {
        msg!("EE VAA Program failed: {}", e);
        e
    })?;
    Ok(())
}
