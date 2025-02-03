use anchor_lang::prelude::*;
use anchor_lang::solana_program::keccak::HASH_BYTES;

declare_id!(VERIFY_VAA_SHIM_PROGRAM_ID);

mod instructions;
pub(crate) use instructions::*;

pub mod state;

pub mod error;

use solana_program::{
    entrypoint::ProgramResult, program::invoke_signed_unchecked, system_instruction,
};
use wormhole_svm_definitions::{
    zero_copy::GuardianSignatures, GUARDIAN_SIGNATURE_LENGTH, VERIFY_VAA_SHIM_PROGRAM_ID,
};
use wormhole_svm_shim::verify_vaa::{PostSignaturesData, VerifyVaaShimInstruction};

fn process_instruction(
    program_id: &Pubkey,
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    // Verify the program ID is what we expect.
    if program_id != &ID {
        return Err(ProgramError::IncorrectProgramId);
    }

    match VerifyVaaShimInstruction::deserialize(instruction_data) {
        Some(VerifyVaaShimInstruction::PostSignatures(data)) => {
            process_post_signatures(accounts, data)
        }
        Some(VerifyVaaShimInstruction::CloseSignatures) => process_close_signatures(accounts),
        _ => Err(ProgramError::InvalidInstructionData),
    }
}

#[program]
pub mod wormhole_verify_vaa_shim {

    use super::*;

    // Temporary while rewrite is in progress.
    pub fn fallback_process_instruction(
        program_id: &Pubkey,
        accounts: &[AccountInfo],
        instruction_data: &[u8],
    ) -> Result<()> {
        process_instruction(program_id, accounts, instruction_data).map_err(Into::into)
    }

    /// This instruction is intended to be invoked via CPI call. It verifies a digest against a GuardianSignatures account
    /// and a core bridge GuardianSet.
    /// Prior to this call, and likely in a separate transaction, `post_signatures` must be called to create the account.
    /// Immediately after this call, `close_signatures` should be called to reclaim the lamports.
    ///
    /// A v1 VAA digest can be computed as follows:
    /// ```rust
    /// # let vaa_body = vec![];
    ///   let message_hash = &solana_program::keccak::hashv(&[&vaa_body]).to_bytes();
    ///   let digest = solana_program::keccak::hash(message_hash.as_slice()).to_bytes();
    /// ```
    ///
    /// A QueryResponse digest can be computed as follows:
    /// ```rust,ignore
    ///   use wormhole_query_sdk::MESSAGE_PREFIX;
    ///   let message_hash = [
    ///     MESSAGE_PREFIX,
    ///     &solana_program::keccak::hashv(&[&bytes]).to_bytes(),
    ///   ]
    ///   .concat();
    ///   let digest = keccak::hash(message_hash.as_slice()).to_bytes();
    /// ```
    pub fn verify_hash(ctx: Context<VerifyHash>, digest: [u8; HASH_BYTES]) -> Result<()> {
        instructions::verify_hash(ctx, digest)
    }
}

fn process_post_signatures(
    accounts: &[AccountInfo],
    data: PostSignaturesData<true>,
) -> ProgramResult {
    if accounts.len() < 3 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Payer will pay the rent for the guardian signatures account if it does
    // not exist. CPI call to System program will fail if the account is not
    // writable or is not a signer.
    let payer = &accounts[0];

    // Guardian signatures account will store the guardian signatures. This
    // account will be created if it does not already exist. For subsequent
    // calls to this instruction, this account does not need to be a signer.
    let guardian_signatures = &accounts[1];
    if guardian_signatures.key == payer.key {
        msg!("Guardian signatures (account #2) cannot be initialized as payer (account #1)");
        return Err(ProgramError::InvalidAccountData);
    }

    // NOTE: We are not checking account at index == 2 because the System
    // program CPI will fail if the System program account is not present.

    let total_signatures = data.total_signatures();
    if total_signatures == 0 {
        msg!("Total signatures must not be 0");
        return Err(ProgramError::InvalidArgument);
    }

    let guardian_signatures_slice = data.guardian_signatures_slice();
    if guardian_signatures_slice.is_empty() {
        msg!("Guardian signatures must not be empty");
        return Err(ProgramError::InvalidArgument);
    }

    let mut guardian_signatures_len = data.guardian_signatures().len() as u32;

    let start_idx = if guardian_signatures.owner != &ID {
        create_guardian_signatures_account(
            payer.key,
            guardian_signatures.key,
            guardian_signatures.lamports(),
            total_signatures,
            accounts,
        )?;

        // Now write some data.
        let mut account_data = guardian_signatures.data.borrow_mut();
        account_data[..8].copy_from_slice(&GuardianSignatures::DISCRIMINATOR);
        account_data[8..40].copy_from_slice(payer.key.as_ref());
        account_data[40..44].copy_from_slice(&data.guardian_set_index().to_be_bytes());

        GuardianSignatures::MINIMUM_SIZE
    } else {
        let account_data = guardian_signatures.data.borrow();
        let guardian_signatures =
            GuardianSignatures::new(&account_data).ok_or(ProgramError::InvalidAccountData)?;

        if compute_guardian_signatures_data_len(total_signatures) != account_data.len() {
            let expected_total_signatures =
                (account_data.len() - GuardianSignatures::MINIMUM_SIZE) / GUARDIAN_SIGNATURE_LENGTH;
            msg!("Total signatures mismatch");
            msg!("Left:");
            msg!("{}", total_signatures);
            msg!("Right:");
            msg!("{}", expected_total_signatures);
            return Err(ProgramError::InvalidArgument);
        }

        if payer.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!("Payer (account #1) must match refund recipient in guardian signatures");
            msg!("Left:");
            msg!("{}", payer.key);
            msg!("Right:");
            msg!("{}", guardian_signatures.refund_recipient());
            return Err(ProgramError::InvalidAccountData);
        }

        if data.guardian_set_index() != guardian_signatures.guardian_set_index() {
            msg!("Guardian set index must match guardian set index in guardian signatures");
            msg!("Left:");
            msg!("{}", data.guardian_set_index());
            msg!("Right:");
            msg!("{}", guardian_signatures.guardian_set_index());
            return Err(ProgramError::InvalidArgument);
        }

        let current_guardian_signatures_len = guardian_signatures.guardian_signatures_len();

        // Update the length, which will be written back to the account.
        guardian_signatures_len += current_guardian_signatures_len;

        GuardianSignatures::MINIMUM_SIZE
            + (current_guardian_signatures_len as usize) * GUARDIAN_SIGNATURE_LENGTH
    };

    let mut account_data = guardian_signatures.data.borrow_mut();
    let end_idx = start_idx + guardian_signatures_slice.len();

    // Check if the account data is not large enough to hold the signatures.
    if end_idx > account_data.len() {
        msg!("Too many input guardian signatures");
        return Err(ProgramError::AccountDataTooSmall);
    }

    // Write signatures.
    account_data[start_idx..end_idx].copy_from_slice(guardian_signatures_slice);

    // Update length.
    account_data[44..48].copy_from_slice(&guardian_signatures_len.to_le_bytes());

    Ok(())
}

fn process_close_signatures(accounts: &[AccountInfo]) -> ProgramResult {
    if accounts.len() < 2 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Guardian signatures account will be closed by the end of this
    // instruction. Rent will be moved to the refund recipient.
    let guardian_signatures = &accounts[0];

    // Recipient of guardian signatures account's lamports.
    let refund_recipient = &accounts[1];

    {
        let account_data = guardian_signatures.data.borrow();
        let guardian_signatures =
            GuardianSignatures::new(&account_data).ok_or(ProgramError::InvalidAccountData)?;

        if refund_recipient.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!(
                "Refund recipient (account #2) must match refund recipient in guardian signatures"
            );
            msg!("Left:");
            msg!("{}", refund_recipient.key);
            msg!("Right:");
            msg!("{}", guardian_signatures.refund_recipient());
            return Err(ProgramError::InvalidAccountData);
        }
    }

    let mut guardian_signatures_lamports = guardian_signatures.lamports.borrow_mut();
    **refund_recipient.lamports.borrow_mut() += **guardian_signatures_lamports;
    **guardian_signatures_lamports = 0;

    guardian_signatures.realloc(0, false).unwrap();
    guardian_signatures.assign(&Default::default());

    Ok(())
}

#[inline(always)]
fn create_guardian_signatures_account(
    payer_key: &Pubkey,
    guardian_signatures_key: &Pubkey,
    current_lamports: u64,
    total_signatures: u8,
    accounts: &[AccountInfo],
) -> ProgramResult {
    // TODO: Should we revert if total signatures exceeds 154? If it does, the
    // System program's allocate attempt will fail. But it might be nicer to
    // have an explicit error for this case.
    let data_len = compute_guardian_signatures_data_len(total_signatures);
    let lamports = Rent::get().unwrap().minimum_balance(data_len);

    if current_lamports == 0 {
        // Create the account.
        let ix = system_instruction::create_account(
            payer_key,
            guardian_signatures_key,
            lamports,
            data_len as u64,
            &ID,
        );

        invoke_signed_unchecked(&ix, accounts, &[])?;
    } else if current_lamports < lamports {
        let lamport_diff = lamports.saturating_sub(current_lamports);

        if lamport_diff != 0 {
            // Fund the account.
            let ix = system_instruction::transfer(payer_key, guardian_signatures_key, lamports);
            invoke_signed_unchecked(&ix, accounts, &[])?;
        }

        let ix = system_instruction::allocate(guardian_signatures_key, data_len as u64);
        invoke_signed_unchecked(&ix, accounts, &[])?;

        let ix = system_instruction::assign(guardian_signatures_key, &ID);
        invoke_signed_unchecked(&ix, accounts, &[])?;
    }

    Ok(())
}

#[inline(always)]
fn compute_guardian_signatures_data_len(total_signatures: u8) -> usize {
    (total_signatures as usize) * GUARDIAN_SIGNATURE_LENGTH + GuardianSignatures::MINIMUM_SIZE
}
