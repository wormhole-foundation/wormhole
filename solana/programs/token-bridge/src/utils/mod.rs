pub(crate) mod cpi;

mod token;
pub use token::*;

mod vaa;
pub use vaa::*;

use anchor_lang::prelude::*;

/// Determine the sender address based on whether an integrator provides his own program ID. If a
/// program ID is provided, it is verified against the sender authority, which requires specific
/// seeds. Otherwise the sender is the authority itself.
pub fn new_sender_address(
    sender_authority: &Signer,
    cpi_program_id: Option<Pubkey>,
) -> Result<Pubkey> {
    match cpi_program_id {
        Some(program_id) => {
            let (expected_authority, _) = Pubkey::find_program_address(
                &[crate::constants::PROGRAM_SENDER_SEED_PREFIX],
                &program_id,
            );
            require_eq!(
                sender_authority.key(),
                expected_authority,
                crate::error::TokenBridgeError::InvalidProgramSender
            );
            Ok(program_id)
        }
        None => Ok(sender_authority.key()),
    }
}

pub fn fix_account_order<'info>(
    account_infos: &[AccountInfo<'info>],
    num_accounts: usize,
    requires_token_program: bool,
    requires_core_bridge_program: bool,
) -> Result<Vec<AccountInfo<'info>>> {
    let mut infos = account_infos.to_vec();

    // This check is inclusive because Core Bridge program, System program and Token program can
    // be in any order.
    if infos.len() >= num_accounts {
        let mut expected_index = num_accounts - 1;
        let mut swap_program_to_index = |program_id: &Pubkey| -> Result<()> {
            let program_idx = infos
                .iter()
                .position(|info| info.key() == *program_id)
                .ok_or(error!(ErrorCode::InvalidProgramId))?;

            // Make sure the program is in the right index.
            if program_idx != expected_index {
                infos.swap(expected_index, program_idx);
            }

            expected_index -= 1;

            Ok(())
        };

        //Swapping programs into their correct positions in reverse order (last to first)
        //The Core Bridge program is always the last account if it is required
        if requires_core_bridge_program {
            swap_program_to_index(&core_bridge_program::ID)?;
        }
        //The Token program is either the last account or preceeds the Core Bridge program
        if requires_token_program {
            swap_program_to_index(&anchor_spl::token::ID)?;
        }
        //Finally, the System program is always required and comes before the other two
        swap_program_to_index(&anchor_lang::system_program::ID)?;
    }

    Ok(infos)
}
