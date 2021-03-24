//! Program entrypoint definitions

use solana_program::{
    account_info::AccountInfo,entrypoint, entrypoint::ProgramResult,
    pubkey::Pubkey,
};

#[cfg(not(feature = "no-entrypoint"))]
entrypoint!(process_instruction);
fn process_instruction<'a>(
    _program_id: &Pubkey,
    _accounts: &'a [AccountInfo<'a>],
    _instruction_data: &[u8],
) -> ProgramResult {
    Ok(())
}
