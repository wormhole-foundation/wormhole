pub mod cpi;

pub mod vaa;

use crate::error::TokenBridgeError;
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
                TokenBridgeError::InvalidProgramSender
            );
            Ok(program_id)
        }
        None => {
            // Make sure this sender is not executable. This check is a security measure to prevent
            // someone impersonating his program as the sender address if he still holds the
            // keypair used to deploy his program.
            require!(
                !sender_authority.executable,
                TokenBridgeError::ExecutableDisallowed
            );

            Ok(sender_authority.key())
        }
    }
}

pub fn fix_account_order<'info>(
    account_infos: &[AccountInfo<'info>],
    start_index: usize,
    system_program_index: usize,
    token_program_index: Option<usize>,
    core_bridge_program_index: Option<usize>,
) -> Result<Vec<AccountInfo<'info>>> {
    let mut infos = account_infos.to_vec();

    // This check is inclusive because Core Bridge program, System program and Token program can
    // be in any order.
    let expected_num = start_index
        + 2 // unnecessary account + system program
        + usize::from(token_program_index.is_some())
        + usize::from(core_bridge_program_index.is_some());
    if infos.len() >= expected_num {
        let mut swap_program_to_index =
            |expected_index: usize, program_id: &Pubkey| -> Result<()> {
                let program_idx = start_index
                    + infos
                        .iter()
                        .skip(start_index)
                        .position(|info| info.key() == *program_id)
                        .ok_or(error!(ErrorCode::InvalidProgramId))?;

                // Make sure the program is in the right index.
                if program_idx != expected_index {
                    infos.swap(expected_index, program_idx);
                }

                Ok(())
            };

        // If there is a specified Core Bridge program index, fix its location.
        if let Some(expected_index) = core_bridge_program_index {
            swap_program_to_index(expected_index, &core_bridge_program::ID)?;
        }

        // If there is a specified Token program index, fix its location.
        if let Some(expected_index) = token_program_index {
            swap_program_to_index(expected_index, &anchor_spl::token::ID)?;
        }

        // Finally fix the System Program location.
        swap_program_to_index(system_program_index, &anchor_lang::system_program::ID)?;
    }

    Ok(infos)
}

/// Convenient trait to determine amount truncation for encoded token transfer amounts.
pub trait TruncateAmount {
    fn mint_decimals(&self) -> u8;

    fn truncate_amount(&self, amount: u64) -> u64 {
        match self
            .mint_decimals()
            .saturating_sub(crate::constants::MAX_DECIMALS)
        {
            0 => amount,
            diff => {
                let divisor = u64::pow(10, diff.into());
                (amount / divisor) * divisor
            }
        }
    }
}

impl TruncateAmount for anchor_spl::token::Mint {
    fn mint_decimals(&self) -> u8 {
        self.decimals
    }
}
