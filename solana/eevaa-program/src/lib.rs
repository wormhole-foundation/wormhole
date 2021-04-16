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
    system_instruction,
};

use std::io::Write;

use crate::{
    error::Error::{self, *},
    instruction::EEVAAInstruction,
};

/// How much from posting an EE VAA goes to guardians
pub const EEVAA_BASE_TX_FEE: u64 = 18 * 10_000;

pub const HOURS_IN_YEAR: u64 = 24 * 365;

// TODO(drozdziak1): Extract the rent price on-chain
/// How much does it cost to store a byte for 3 hrs on Solana?
pub const LAMPORTS_PER_THREE_BYTEHOURS: u64 = 3 * DEFAULT_LAMPORTS_PER_BYTE_YEAR / HOURS_IN_YEAR;

#[cfg(not(feature = "no-entrypoint"))]
entrypoint!(eevaa_entrypoint);
fn eevaa_entrypoint<'a>(
    program_id: &Pubkey,
    accounts: &'a [AccountInfo<'a>],
    instruction_data: &[u8],
) -> ProgramResult {
    let mut acc_iter = accounts.iter();

    // Verify that the EEVAA program instruction is not garbage
    let eevaa_ix: EEVAAInstruction =
        EEVAAInstruction::deserialize(instruction_data).map_err(|e| {
            msg!("Invalid EE VAA: {}", e);
            e
        })?;

    use EEVAAInstruction::*;
    match eevaa_ix {
        PostEEVAA(eevaa) => {
            msg!("Before PostEEVAA account checks! Accounts: {:?}", accounts);
            msg!("Before program fee acc info");
            let eevaa_fee_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
            msg!("Before system prog acc info");
            let system_prog_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
            msg!("Before payer_acc_info");
            let payer_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;
            msg!("Before ee_vaa_acc_info");
            let eevaa_acc_info: &AccountInfo = next_account_info(&mut acc_iter)?;

            msg!("After PostEEVAA account checks!");

            let payload = instruction_data;
            let payload_len = payload.len() as u64;

            // How many lamports to keep the payload for a couple hours?
            let eevaa_acc_rent_amount =
                ACCOUNT_STORAGE_OVERHEAD * payload_len * LAMPORTS_PER_THREE_BYTEHOURS;

            // Regenerate the new EEVAA account
            let (eevaa_acc_key, eevaa_acc_seeds) = derive_eevaa_key(program_id, &eevaa)?;

            // Create new account
            let ix_create_acc: Instruction = system_instruction::create_account(
                payer_acc_info.signer_key().ok_or_else(|| {
                    msg!("Payer must be a signer!");
                    ProgErr::InvalidAccountData
                })?,
                &eevaa_acc_key,
                eevaa_acc_rent_amount,
                payload_len,
                program_id,
            );

            let accounts = &[
                system_prog_acc_info.clone(),
                payer_acc_info.clone(),
                eevaa_acc_info.clone(),
            ];

            msg!("PostEEVAA: Before invoke_signed for creating new account");
            invoke_signed(
                &ix_create_acc,
                accounts,
                &[eevaa_acc_seeds
                    .iter()
                    .map(|seed| seed.as_slice())
                    .collect::<Vec<_>>()
                    .as_slice()][..],
            )?;
            msg!("PostEEVAA: After invoke_signed for creating the new account");

            // Insert payload into the account
            let mut data = eevaa_acc_info.try_borrow_mut_data()?;
            data.write_all(payload).map_err(|e| {
                msg!(
                    "Could not write payload to EEVAA account: {}",
                    e.to_string()
                );
                Error::Internal
            })?;

            let (_eevaa_fee_acc_key, eevaa_fee_acc_seeds) = derive_eevaa_fee_key(program_id)?;

            // Pay the EEVAA posting fee
            let ix_pay_fee: Instruction = system_instruction::transfer(
                payer_acc_info.signer_key().ok_or_else(|| {
                    msg!("Payer must be a signer!");
                    ProgErr::InvalidAccountData
                })?,
                eevaa_fee_acc_info.unsigned_key(),
                EEVAA_BASE_TX_FEE,
            );

            let accounts = &[
                system_prog_acc_info.clone(),
                payer_acc_info.clone(),
                eevaa_fee_acc_info.clone(),
            ];

            msg!("PostEEVAA: Before invoke_signed for paying the fee");
            invoke_signed(
                &ix_pay_fee,
                accounts,
                &[eevaa_fee_acc_seeds
                    .iter()
                    .map(|seed| seed.as_slice())
                    .collect::<Vec<_>>()
                    .as_slice()][..],
            )?;
            msg!("PostEEVAA: After invoke_signed for paying the fee");
        }
        Initialize(init_params) => {
            msg!("Initializing EEVAA program!");
            msg!("Before Initialize account checks");

            let eevaa_fee_acc_info = next_account_info(&mut acc_iter)?;
            let system_prog_acc_info = next_account_info(&mut acc_iter)?;
            let fee_payer_acc_info = next_account_info(&mut acc_iter)?;

            msg!("After Initialize account checks");

            let (eevaa_fee_acc_key, eevaa_fee_acc_seeds) = derive_eevaa_fee_key(program_id)?;

            msg!("Creating the EEVAA program fee account");
            let ix: Instruction = system_instruction::create_account(
                fee_payer_acc_info.signer_key().ok_or_else(|| {
                    msg!("Payer must be a signer!");
                    ProgErr::InvalidAccountData
                })?,
                &eevaa_fee_acc_key,
                init_params.eevaa_fee_acc_rent,
                0,
                program_id,
            );

            let accounts = &[
                eevaa_fee_acc_info.clone(),
                system_prog_acc_info.clone(),
                fee_payer_acc_info.clone(),
            ];

            msg!("Init: Before invoke_signed");
            invoke_signed(
                &ix,
                accounts,
                &[eevaa_fee_acc_seeds
                    .iter()
                    .map(|seed| seed.as_slice())
                    .collect::<Vec<_>>()
                    .as_slice()][..],
            )?;
            msg!("Init: After invoke_signed");
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

/// Creates unique seeds for `eevaa`
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

/// Seeds for the EEVAA fee account
pub fn derive_eevaa_fee_seeds() -> Vec<Vec<u8>> {
    vec!["EevaaBridgeFees".as_bytes().to_vec()]
}

/// Finds a key for the EEVAA fee account
pub fn derive_eevaa_fee_key(program_id: &Pubkey) -> Result<(Pubkey, Vec<Vec<u8>>), Error> {
    find_program_address(&derive_eevaa_fee_seeds(), program_id)
}

/// Retries [`Pubkey::create_program_address`] with a nonce, returns the pubkey
/// and the seeds that had worked.
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
