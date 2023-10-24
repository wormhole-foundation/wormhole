mod native;
pub use native::*;

mod wrapped;
pub use wrapped::*;

use anchor_lang::prelude::*;

use crate::utils::fix_account_order;

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
/// 14. `_rent`                  <-- order unspecified
/// 15. `system_program`         <-- order unspecified
/// 16. `token_program`          <-- order unspecified
/// 17. `core_bridge_program`    <-- order unspecified
///
/// Because the legacy implementation did not require specifying where the Rent sysvar, System
/// program, SPL token program and Core Bridge program should be, we ensure that these accounts are
/// 14, 15, 16 and 17 respectively because the Anchor account context requires them to be in these
/// positions.
pub(super) fn order_transfer_tokens_account_infos<'info>(
    account_infos: &[AccountInfo<'info>],
) -> Result<Vec<AccountInfo<'info>>> {
    fix_account_order(
        account_infos,
        13,       // start_index
        14,       // system_program_index
        Some(15), // token_program_index
        Some(16), // core_bridge_program_index
    )
}
