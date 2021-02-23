//! Wormhole External Entity Verifiable Action Approval handling
//! program. Processes instructions related to arbitrary cross-chain
//! messaging.

pub mod error;
pub mod instruction;

#[macro_use]
extern crate solana_program;

use instruction::EEVAA;
use solana_program::{
    account_info::{next_account_info, AccountInfo},
    entrypoint,
    entrypoint::ProgramResult,
    instruction::Instruction,
    program::invoke_signed,
    program_error::ProgramError as ProgErr,
    pubkey::Pubkey,
    rent::{ACCOUNT_STORAGE_OVERHEAD, DEFAULT_LAMPORTS_PER_BYTE_YEAR},
    system_instruction::create_account,
};

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

    msg!("Before account checks! Accounts: {:?}", accounts);
    let program_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
    let system_prog_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
    msg!("Before payer_acc_info");
    let payer_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
    msg!("Before ee_vaa_acc_info");
    let ee_vaa_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;

    msg!("After account checks!");

    // Verify that the EE VAA is not garbage
    let ee_vaa_ix: EEVAAInstruction =
        EEVAAInstruction::deserialize(instruction_data).map_err(|e| {
            msg!("Invalid EE VAA: {}", e);
            e
        })?;

    match ee_vaa_ix {
        EEVAAInstruction::PostEEVAA(eevaa) => {
            let payload = instruction_data;
            let payload_len = payload.len() as u64;

            let ee_vaa_acc_rent =
                ACCOUNT_STORAGE_OVERHEAD * payload_len * LAMPORTS_PER_THREE_BYTEHOURS;

            // transfer_sol(payer_acc_info, program_acc_info, EE_VAA_BASE_TX_FEE)?;

            // Create new account, program pays for it with surplus over TX fee
            let ix: Instruction = create_account(
                payer_acc_info.signer_key().ok_or_else(|| {
                    msg!("Payer must be a signer!");
                    ProgErr::InvalidAccountData
                })?,
                ee_vaa_acc_info.unsigned_key(),
                // ee_vaa_acc_info.signer_key().ok_or_else(|| {
                //     msg!("EE VAA acc must be a signer!");
                //     ProgErr::InvalidAccountData
                // })?,
                ee_vaa_acc_rent,
                payload_len,
                program_id,
            );

            let accounts = &[
                system_prog_acc_info.clone(),
                payer_acc_info.clone(),
                ee_vaa_acc_info.clone(),
            ];

            let (_key, seeds) = derive_eevaa_key(program_id, &eevaa)?;

            msg!("Yooooooooo");
            invoke_signed(
                &ix,
                accounts,
                &[seeds
                    .iter()
                    .map(|seed| seed.as_slice())
                    .collect::<Vec<_>>()
                    .as_slice()][..],
            )?;
            msg!("Greetings after");
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
        msg!("Expected owner {}, got {}", owner, acc.owner);
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

pub fn derive_eevaa_seeds(eevaa: &EEVAA) -> Vec<Vec<u8>> {
    vec![eevaa.id.to_be_bytes().to_vec(), eevaa.payload.clone()]
}

/// Creates program address for the EEVAA program
pub fn derive_eevaa_key(
    program_id: &Pubkey,
    eevaa: &EEVAA,
) -> Result<(Pubkey, Vec<Vec<u8>>), Error> {
    let seeds = derive_eevaa_seeds(eevaa);

    find_program_address(&seeds, program_id)
}

pub fn find_program_address(
    seeds: &Vec<Vec<u8>>,
    program_id: &Pubkey,
) -> Result<(Pubkey, Vec<Vec<u8>>), Error> {
    let mut nonce = [255u8];
    for _ in 0..std::u8::MAX {
        {
            let mut seeds_with_nonce = seeds.to_vec();
            seeds_with_nonce.push(nonce.to_vec());
            let s: Vec<_> = seeds_with_nonce
                .iter()
                .map(|item| item.as_slice())
                .collect();
            if let Ok(address) = Pubkey::create_program_address(&s, program_id) {
                return Ok((address, seeds_with_nonce));
            }
        }
        nonce[0] -= 1;
    }
    msg!("Unable to find a viable program address nonce");
    Err(Error::Internal)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_lamports_per_3bh_non_zero() {
        assert!(LAMPORTS_PER_THREE_BYTEHOURS > 0);
    }
}
