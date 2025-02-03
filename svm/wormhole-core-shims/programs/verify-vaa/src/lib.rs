use solana_program::{
    account_info::AccountInfo, clock::Clock, entrypoint::ProgramResult, msg,
    program::invoke_signed_unchecked, program_error::ProgramError, pubkey::Pubkey, rent::Rent,
    system_instruction, sysvar::Sysvar,
};
use wormhole_svm_definitions::{
    zero_copy::{GuardianSet, GuardianSignatures},
    CORE_BRIDGE_PROGRAM_ID, GUARDIAN_SET_SEED, GUARDIAN_SIGNATURE_LENGTH,
    VERIFY_VAA_SHIM_PROGRAM_ID,
};
use wormhole_svm_shim::verify_vaa::{PostSignaturesData, VerifyHashData, VerifyVaaShimInstruction};

solana_program::declare_id!(VERIFY_VAA_SHIM_PROGRAM_ID);

solana_program::entrypoint!(process_instruction);

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
        Some(VerifyVaaShimInstruction::VerifyHash(data)) => process_verify_hash(accounts, data),
        Some(VerifyVaaShimInstruction::CloseSignatures) => process_close_signatures(accounts),
        _ => Err(ProgramError::InvalidInstructionData),
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
    let payer_info = &accounts[0];

    // Guardian signatures account will store the guardian signatures. This
    // account will be created if it does not already exist. For subsequent
    // calls to this instruction, this account does not need to be a signer.
    let guardian_signatures_info = &accounts[1];
    if guardian_signatures_info.key == payer_info.key {
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

    let start_idx = if guardian_signatures_info.owner != &ID {
        create_guardian_signatures_account(
            payer_info.key,
            guardian_signatures_info.key,
            guardian_signatures_info.lamports(),
            total_signatures,
            accounts,
        )?;

        // Now write some data.
        let mut account_data = guardian_signatures_info.data.borrow_mut();
        account_data[..8].copy_from_slice(&GuardianSignatures::DISCRIMINATOR);
        account_data[8..40].copy_from_slice(payer_info.key.as_ref());
        account_data[40..44].copy_from_slice(&data.guardian_set_index().to_be_bytes());

        GuardianSignatures::MINIMUM_SIZE
    } else {
        let account_data = guardian_signatures_info.data.borrow();
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

        if payer_info.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!("Payer (account #1) must match refund recipient in guardian signatures");
            msg!("Left:");
            msg!("{}", payer_info.key);
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

    let mut account_data = guardian_signatures_info.data.borrow_mut();
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

fn process_verify_hash(accounts: &[AccountInfo], data: VerifyHashData) -> ProgramResult {
    if accounts.len() < 2 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Guardian set account warehouses the guardians' ETH public keys. These
    // keys are used to verify the public keys recovered from verifying the
    // signatures in the guardian signatures account with the provided digest.
    let guardian_set_info = &accounts[0];

    // Guardian signatures account.
    let guardian_signatures_info_data = accounts[1].data.borrow();
    let guardian_signatures = GuardianSignatures::new(&guardian_signatures_info_data)
        .ok_or(ProgramError::InvalidAccountData)?;

    // With the guardian signatures account's encoded guardian set index and
    // provided bump, create the guardian set account's address.
    let expected_address = Pubkey::create_program_address(
        &[
            GUARDIAN_SET_SEED,
            guardian_signatures.guardian_index_be_slice(),
            &[data.guardian_set_bump()],
        ],
        &CORE_BRIDGE_PROGRAM_ID,
    )
    .map_err(|err| {
        msg!("Guardian set (account #1) address creation failed");
        err
    })?;
    if guardian_set_info.key != &expected_address {
        msg!("Guardian set (account #1) seeds constraint violated");
        msg!("Left:");
        msg!("{}", guardian_set_info.key);
        msg!("Right:");
        msg!("{}", expected_address);
        return Err(ProgramError::InvalidAccountData);
    }

    let guardian_set_info_data = guardian_set_info.data.borrow();

    // Unwrapping here is safe because we verified the guardian set's address.
    let guardian_set = GuardianSet::new(&guardian_set_info_data).unwrap();

    // This operation will panic quite far in the future.
    let timestamp = u32::try_from(Clock::get().unwrap().unix_timestamp).unwrap();
    if !guardian_set.is_active(timestamp) {
        msg!("Guardian set (account #1) is expired");
        return Err(ProgramError::InvalidAccountData);
    }

    let guardian_signatures_len = guardian_signatures.guardian_signatures_len();

    // Number of signatures must meet quorum, which is at least two thirds of
    // the guardian set's size.
    if guardian_signatures_len < guardian_set.quorum() {
        msg!("Guardian signatures (account #2) fails to meet quorum");
        return Err(ProgramError::InvalidAccountData);
    }

    let digest = data.digest().0;
    let mut last_guardian_index = None;

    for i in 0..(guardian_signatures_len as usize) {
        // Signature is encoded as:
        // - 1 byte: guardian index
        // - 64 bytes: r and s concatenated
        // - 1 byte: recovery ID
        let signature = guardian_signatures.guardian_signature(i).unwrap();
        let index = signature[0] as usize;
        if let Some(last_index) = last_guardian_index {
            if index <= last_index {
                msg!("Guardian signatures (account #2) has non-increasing guardian index");
                return Err(ProgramError::InvalidAccountData);
            }
        }

        let guardian_pubkey = guardian_set.key_slice(index).ok_or_else(|| {
            msg!("Guardian signatures (account #2) guardian index out of range");
            ProgramError::InvalidAccountData
        })?;

        // Keccak-256 hash the recovered public key and compare it with the
        // guardian set's public key.
        let recovered_pubkey = solana_program::secp256k1_recover::secp256k1_recover(
            &digest,
            signature[65],
            &signature[1..65],
        )
        .map(|pubkey| solana_program::keccak::hashv(&[&pubkey.0]).0)
        .map_err(|_| {
            msg!("Guardian signature index {} is invalid", i);
            ProgramError::InvalidAccountData
        })?;

        if &recovered_pubkey[12..] != guardian_pubkey {
            msg!(
                "Guardian signature index {} does not recover guardian {} pubkey",
                i,
                index
            );
            return Err(ProgramError::InvalidAccountData);
        }

        last_guardian_index.replace(index);
    }

    Ok(())
}

fn process_close_signatures(accounts: &[AccountInfo]) -> ProgramResult {
    if accounts.len() < 2 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Guardian signatures account will be closed by the end of this
    // instruction. Rent will be moved to the refund recipient.
    let guardian_signatures_info = &accounts[0];

    // Recipient of guardian signatures account's lamports.
    let refund_recipient_info = &accounts[1];

    {
        let account_data = guardian_signatures_info.data.borrow();
        let guardian_signatures =
            GuardianSignatures::new(&account_data).ok_or(ProgramError::InvalidAccountData)?;

        if refund_recipient_info.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!(
                "Refund recipient (account #2) must match refund recipient in guardian signatures"
            );
            msg!("Left:");
            msg!("{}", refund_recipient_info.key);
            msg!("Right:");
            msg!("{}", guardian_signatures.refund_recipient());
            return Err(ProgramError::InvalidAccountData);
        }
    }

    let mut guardian_signatures_lamports = guardian_signatures_info.lamports.borrow_mut();
    **refund_recipient_info.lamports.borrow_mut() += **guardian_signatures_lamports;
    **guardian_signatures_lamports = 0;

    guardian_signatures_info.realloc(0, false).unwrap();
    guardian_signatures_info.assign(&Default::default());

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
