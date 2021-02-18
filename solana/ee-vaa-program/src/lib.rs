//! Wormhole External Entity Verifiable Action Approval handling
//! program. Processes instructions related to arbitrary cross-chain
//! messaging.

#[macro_use]
extern crate solana_program;

pub mod error;
pub mod instruction;

use solana_program::{
    account_info::next_account_info, account_info::AccountInfo, entrypoint,
    entrypoint::ProgramResult, program_error::ProgramError as ProgErr, pubkey::Pubkey,
    rent::ACCOUNT_STORAGE_OVERHEAD, rent::DEFAULT_LAMPORTS_PER_BYTE_YEAR,
    system_instruction::create_account,
};

use std::convert::TryFrom;

use crate::{
    error::Error::{self, *},
    instruction::EEVAAInstruction,
};

/// How much from posting an EE VAA goes to guardians
pub const EE_VAA_BASE_TX_FEE: u64 = 18 * 10_000;

pub const HOURS_IN_YEAR: u64 = 24 * 365;

/// How much does it cost to store a byte for 3 hrs on Solana?
pub const LAMPORTS_PER_THREE_BYTEHOURS: u64 = 3 * DEFAULT_LAMPORTS_PER_BYTE_YEAR / HOURS_IN_YEAR;

#[cfg(not(feature = "no-entrypoint"))]
entrypoint!(ee_vaa_entrypoint);
fn ee_vaa_entrypoint<'a>(
    program_id: &Pubkey,
    accounts: &'a [AccountInfo<'a>],
    instruction_data: &[u8],
) -> ProgramResult {
    let mut acc_iter = accounts.iter();
    let program_acc: &AccountInfo = next_account_info_with_owner(&mut acc_iter, program_id)?;
    let payer_acc: &AccountInfo = next_account_info(&mut acc_iter)?;

    // Verify that the EE VAA is not garbage
    let ix: EEVAAInstruction = EEVAAInstruction::deserialize(instruction_data).map_err(|e| {
        msg!("Invalid EE VAA Payload: {}", e);
        e
    })?;

    match ix {
        EEVAAInstruction::PostEEVAA(ee_vaa) => {
            let payload = instruction_data;
            let payload_len = payload.len() as u64;

            let ee_vaa_acc_rent =
                ACCOUNT_STORAGE_OVERHEAD * payload_len * LAMPORTS_PER_THREE_BYTEHOURS;

            let amount = EE_VAA_BASE_TX_FEE + ee_vaa_acc_rent;

            // Transfer full fee from payer
            transfer_sol(payer_acc, program_acc, amount)?;

            let ee_vaa_acc = unimplemented!();

            // Create new account, program pays for it with surplus over TX FEE
            let ix = create_account(
                program_id,
                ee_vaa_acc,
                ee_vaa_acc_rent,
                payload_len,
                program_id,
            );


        }
    }

    Ok(())
}

pub fn next_account_info_with_owner<'a, 'b, I: Iterator<Item = &'a AccountInfo<'b>>>(
    iter: &mut I,
    owner: &Pubkey,
) -> Result<I::Item, error::Error> {
    let acc = iter.next().ok_or(UnexpectedEndOfAccounts)?;
    if acc.owner != owner {
        return Err(InvalidOwner);
    }
    Ok(acc)
}

pub fn transfer_sol(
    payer_account: &AccountInfo,
    recipient_account: &AccountInfo,
    amount: u64,
) -> ProgramResult {
    let mut payer_balance = payer_account.try_borrow_mut_lamports()?;
    **payer_balance = payer_balance
        .checked_sub(amount)
        .ok_or(ProgErr::InsufficientFunds)?;
    let mut recipient_balance = recipient_account.try_borrow_mut_lamports()?;
    **recipient_balance = recipient_balance
        .checked_add(amount)
        .ok_or(ProgErr::InvalidArgument)?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_lamports_per_3bh_non_zero() {
        assert!(LAMPORTS_PER_THREE_BYTEHOURS > 0);
    }
}
