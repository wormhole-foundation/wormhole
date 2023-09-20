mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::*;

/// The Anchor context orders the accounts as:
///
/// 1.  `payer`
/// 2.  `_config`
/// 3.  `src_token`
/// 4.  `mint`               OR `_src_owner`
/// 5.  `custody_token`      OR `wrapped_mint`
/// 6.  `transfer_authority` OR `wrapped_asset`
/// 7.  `custody_authority`  OR `transfer_authority`
/// 8.  `core_bridge_config`
/// 9.  `core_message`
/// 10. `core_emitter`
/// 11. `core_emitter_sequence`
/// 12. `core_fee_collector`
/// 13. `_clock`
/// 14. `sender_authority`
/// 15. `_rent`                  <-- order unspecified
/// 16. `system_program`         <-- order unspecified
/// 17. `token_program`          <-- order unspecified
/// 18. `core_bridge_program`    <-- order unspecified
///
/// Because the legacy implementation did not require specifying where the Rent sysvar, System
/// program, SPL token program and Core Bridge program should be, we ensure that these accounts are
/// 15, 16, 17 and 18 respectively because the Anchor account context requires them to be in these
/// positions.
pub(super) fn order_transfer_tokens_with_payload_account_infos<'info>(
    account_infos: &[AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    const NUM_ACCOUNTS: usize = 18;
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
