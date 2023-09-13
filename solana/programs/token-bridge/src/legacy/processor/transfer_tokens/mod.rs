mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::*;

pub(super) fn order_transfer_tokens_account_infos<'a, 'info>(
    account_infos: &'a [AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    const NUM_ACCOUNTS: usize = 17;
    const CORE_BRIDGE_PROGRAM_IDX: usize = NUM_ACCOUNTS - 1;
    const TOKEN_PROGRAM_IDX: usize = CORE_BRIDGE_PROGRAM_IDX - 1;
    const SYSTEM_PROGRAM_IDX: usize = TOKEN_PROGRAM_IDX - 1;

    let mut infos = account_infos.to_vec();

    // This check is inclusive because Core Bridge program, System program and Token program can
    // be in any order.
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

        // Core Bridge program needs to exist in these account infos.
        let core_bridge_program_idx = infos
            .iter()
            .position(|info| info.key() == core_bridge_program::ID)
            .ok_or(error!(ErrorCode::InvalidProgramId))?;

        // Make sure Token program is in the right index.
        if core_bridge_program_idx != CORE_BRIDGE_PROGRAM_IDX {
            infos.swap(CORE_BRIDGE_PROGRAM_IDX, core_bridge_program_idx);
        }
    }

    // Done.
    Ok(infos)
}
