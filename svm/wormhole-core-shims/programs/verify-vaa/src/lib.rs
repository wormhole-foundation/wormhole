#![deny(dead_code, unused_imports, unused_mut, unused_variables)]

use solana_program::{
    account_info::AccountInfo,
    clock::Clock,
    entrypoint::ProgramResult,
    instruction::{AccountMeta, Instruction},
    msg,
    program::invoke_signed_unchecked,
    program_error::ProgramError,
    pubkey::Pubkey,
    rent::Rent,
    system_instruction,
    sysvar::Sysvar,
};
use wormhole_svm_definitions::{
    zero_copy::{GuardianSet, GuardianSignatures},
    CORE_BRIDGE_PROGRAM_ID, GUARDIAN_SET_SEED, GUARDIAN_SIGNATURE_LENGTH,
    VERIFY_VAA_SHIM_PROGRAM_ID as ID,
};
use wormhole_svm_shim::verify_vaa::{PostSignaturesData, VerifyHashData, VerifyVaaShimInstruction};

solana_program::entrypoint_no_alloc!(process_instruction);

#[inline]
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

#[inline(always)]
fn process_post_signatures(
    accounts: &[AccountInfo],
    data: PostSignaturesData<true>,
) -> ProgramResult {
    if accounts.len() < 3 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Payer will pay the rent for the guardian signatures account if it does
    // not exist. CPI calls to System program will fail if the account is not
    // writable and is not a signer.
    let payer_info = &accounts[0];

    // Guardian signatures account will store the guardian signatures. This
    // account will be created if it does not already exist. For subsequent
    // calls to this instruction, this account does not need to be a signer.
    let guardian_signatures_info = &accounts[1];

    // NOTE: We are not checking account at index == 2 because the System
    // program CPI will fail if the System program account is not present.

    let guardian_signatures_slice = data.guardian_signatures_slice();
    if guardian_signatures_slice.is_empty() {
        msg!("Guardian signatures must not be empty");
        return Err(ProgramError::InvalidArgument);
    }

    let total_signatures = data.total_signatures();
    let mut guardian_signatures_len = data.guardian_signatures().len() as u32;

    let start_data_index = if guardian_signatures_info.owner != &ID {
        if guardian_signatures_info.key == payer_info.key {
            msg!("Guardian signatures (account #2) cannot be initialized as payer (account #1)");
            return Err(ProgramError::InvalidAccountData);
        }

        // There is no point in creating this account if there are no
        // signatures to write.
        if total_signatures == 0 {
            msg!("Total signatures must not be 0");
            return Err(ProgramError::InvalidArgument);
        }

        // TODO: Should we revert if total signatures exceeds 154? If it does, the
        // System program's allocate attempt will fail. But it might be nicer to
        // have an explicit error for this case.
        create_account_reliably(
            payer_info.key,
            guardian_signatures_info.key,
            guardian_signatures_info.lamports(),
            compute_guardian_signatures_data_len(total_signatures),
            accounts,
        )?;

        // Now write some data.
        let mut account_data = guardian_signatures_info.data.borrow_mut();
        account_data[..8].copy_from_slice(&GuardianSignatures::DISCRIMINATOR);
        account_data[8..40].copy_from_slice(&payer_info.key.to_bytes());
        account_data[40..44].copy_from_slice(&data.guardian_set_index().to_be_bytes());

        GuardianSignatures::MINIMUM_SIZE
    } else {
        // Because the payer encoded in the guardian signatures is the only
        // authority to write to this account, we must check that it is a
        // signer.
        if !payer_info.is_signer {
            msg!("Payer (account #1) must be a signer");
            return Err(ProgramError::MissingRequiredSignature);
        }

        let account_data = guardian_signatures_info.data.borrow();
        let guardian_signatures = GuardianSignatures::new(&account_data).ok_or_else(|| {
            msg!("Guardian signatures (account #2) failed to deserialize");
            ProgramError::InvalidAccountData
        })?;

        // This check may not be necessary. But we ensure that the total
        // signatures is consistent with the account data (even though it is
        // only used when creating the guardian signatures account).
        if compute_guardian_signatures_data_len(total_signatures) != account_data.len() {
            let expected_total_signatures =
                (account_data.len() - GuardianSignatures::MINIMUM_SIZE) / GUARDIAN_SIGNATURE_LENGTH;
            msg!("Total signatures mismatch");
            msg!("Actual:");
            msg!("{}", total_signatures);
            msg!("Expected:");
            msg!("{}", expected_total_signatures);
            return Err(ProgramError::InvalidArgument);
        }

        // Only the original payer (refund recipient) can write to this account
        // with subsequent calls to this instruction.
        if payer_info.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!("Payer (account #1) must match refund recipient");
            msg!("Actual:");
            msg!("{}", payer_info.key);
            msg!("Expected:");
            msg!("{}", guardian_signatures.refund_recipient());
            return Err(ProgramError::InvalidAccountData);
        }

        // Same as the total signatures check above, this check may not be
        // necessary. But we ensure that the guardian set index is consistent
        // with the account data (even though it is only used when creating the
        // guardian signatures account).
        if data.guardian_set_index() != guardian_signatures.guardian_set_index() {
            msg!("Guardian set index mismatch");
            msg!("Actual:");
            msg!("{}", data.guardian_set_index());
            msg!("Expected:");
            msg!("{}", guardian_signatures.guardian_set_index());
            return Err(ProgramError::InvalidArgument);
        }

        let current_guardian_signatures_len = guardian_signatures.guardian_signatures_len();

        // Update the length, which will be written back to the account. Simply
        // adding is safe because the number of signatures will never
        // exceed 154 (which is much smaller than u32::MAX).
        guardian_signatures_len += current_guardian_signatures_len;

        GuardianSignatures::MINIMUM_SIZE
            + (current_guardian_signatures_len as usize) * GUARDIAN_SIGNATURE_LENGTH
    };

    let mut account_data = guardian_signatures_info.data.borrow_mut();

    // This operation is safe because this sum will never reach usize::MAX.
    let end_data_index = start_data_index + guardian_signatures_slice.len();

    // Check if the account data is not large enough to hold the signatures.
    if end_data_index > account_data.len() {
        msg!("Too many input guardian signatures");
        return Err(ProgramError::AccountDataTooSmall);
    }

    // Write signatures.
    account_data[start_data_index..end_data_index].copy_from_slice(guardian_signatures_slice);

    // Update length.
    account_data[44..48].copy_from_slice(&guardian_signatures_len.to_le_bytes());

    Ok(())
}

#[inline(always)]
fn process_verify_hash(accounts: &[AccountInfo], data: VerifyHashData) -> ProgramResult {
    if accounts.len() < 2 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Guardian set account warehouses the guardians' ETH public keys. These
    // keys are used to verify the public keys recovered from verifying the
    // signatures in the guardian signatures account with the provided digest.
    let guardian_set_info = &accounts[0];

    // Guardian signatures account. This account was created in the post
    // signatures instruction. Its signatures will be validated against the
    // guardian pubkeys found in the guardian set account.
    let guardian_signatures_info = &accounts[1];
    if guardian_signatures_info.owner != &ID {
        msg!("Guardian signatures (account #2) must be owned by this program");
        return Err(ProgramError::InvalidAccountData);
    }

    let guardian_signatures_info_data = guardian_signatures_info.data.borrow();
    let guardian_signatures =
        GuardianSignatures::new(&guardian_signatures_info_data).ok_or_else(|| {
            msg!("Guardian signatures (account #2) failed to deserialize");
            ProgramError::InvalidAccountData
        })?;

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
        msg!("Actual:");
        msg!("{}", guardian_set_info.key);
        msg!("Expected:");
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
        let signature = guardian_signatures.guardian_signature_slice(i).unwrap();
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

#[inline(always)]
fn process_close_signatures(accounts: &[AccountInfo]) -> ProgramResult {
    if accounts.len() < 2 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Guardian signatures account will be closed by the end of this
    // instruction. Rent will be moved to the refund recipient.
    let guardian_signatures_info = &accounts[0];

    // Recipient of guardian signatures account's lamports. This pubkey must
    // match the refund recipient stored in the guardian signatures account.
    let refund_recipient_info = &accounts[1];

    // Because the refund recipient encoded in the guardian signatures is the
    // only authority to close this account, we must check that it is a signer.
    if !refund_recipient_info.is_signer {
        msg!("Refund recipient (account #2) must be a signer");
        return Err(ProgramError::MissingRequiredSignature);
    }

    {
        let account_data = guardian_signatures_info.data.borrow();
        let guardian_signatures =
            GuardianSignatures::new(&account_data).ok_or(ProgramError::InvalidAccountData)?;

        if refund_recipient_info.key.as_ref() != guardian_signatures.refund_recipient_slice() {
            msg!("Refund recipient (account #2) mismatch");
            msg!("Actual:");
            msg!("{}", refund_recipient_info.key);
            msg!("Expected:");
            msg!("{}", guardian_signatures.refund_recipient());
            return Err(ProgramError::InvalidAccountData);
        }
    }

    let mut guardian_signatures_lamports = guardian_signatures_info.lamports.borrow_mut();
    **refund_recipient_info.lamports.borrow_mut() += **guardian_signatures_lamports;
    **guardian_signatures_lamports = 0;

    // The refund recipient is the only authority that can close this account.
    // So even if lamports were sent to the guardian signatures account to keep
    // it open, the refund recipient will still get the lamports he is owed.
    // Therefore, we do not need to make sure that the account's size is zero
    // and that its owner is ssigned to the System program.
    //
    // When the guardian signatures account's lamports are zero, SVM runtime
    // will ensure that this account will be closed.

    Ok(())
}

#[inline(always)]
fn create_account_reliably(
    payer_key: &Pubkey,
    account_key: &Pubkey,
    current_lamports: u64,
    data_len: usize,
    accounts: &[AccountInfo],
) -> ProgramResult {
    let lamports = Rent::get().unwrap().minimum_balance(data_len);

    if current_lamports == 0 {
        let ix = system_instruction::create_account(
            payer_key,
            account_key,
            lamports,
            data_len as u64,
            &ID,
        );

        invoke_signed_unchecked(&ix, accounts, &[])?;
    } else {
        const MAX_CPI_DATA_LEN: usize = 36;
        const TRANSFER_ALLOCATE_DATA_LEN: usize = 12;
        const SYSTEM_PROGRAM_SELECTOR_LEN: usize = 4;

        // Perform up to three CPIs:
        // 1. Transfer lamports from payer to account (may not be necessary).
        // 2. Allocate data to the account.
        // 3. Assign the account owner to this program.
        //
        // The max length of instruction data is 36 bytes among the three
        // instructions, so we will reuse the same allocated memory for all.
        let mut cpi_ix = Instruction {
            program_id: solana_program::system_program::ID,
            accounts: vec![
                AccountMeta::new(*payer_key, true),
                AccountMeta::new(*account_key, true),
            ],
            data: Vec::with_capacity(MAX_CPI_DATA_LEN),
        };

        // Safety: Because capacity is > 12, it is safe to set this length and
        // to set the first 4 elements to zero, which covers the System program
        // instruction selectors.
        //
        // The transfer and allocate instructions are 12 bytes long:
        // - 4 bytes for the discriminator
        // - 8 bytes for the lamports (transfer) or data length (allocate)
        //
        // The last 8 bytes will be copied to the data slice.
        unsafe {
            let cpi_data = &mut cpi_ix.data;

            core::ptr::write_bytes(cpi_data.as_mut_ptr(), 0, TRANSFER_ALLOCATE_DATA_LEN);
            cpi_data.set_len(TRANSFER_ALLOCATE_DATA_LEN);
        }

        // We will have to transfer the remaining lamports needed to cover rent
        // for the account.
        let lamport_diff = lamports.saturating_sub(current_lamports);

        // Only invoke transfer if there are lamports required.
        if lamport_diff != 0 {
            let cpi_data = &mut cpi_ix.data;

            cpi_data[0] = 2; // transfer selector
            cpi_data[SYSTEM_PROGRAM_SELECTOR_LEN..TRANSFER_ALLOCATE_DATA_LEN]
                .copy_from_slice(&lamport_diff.to_le_bytes());

            invoke_signed_unchecked(&cpi_ix, accounts, &[])?;
        }

        let cpi_accounts = &mut cpi_ix.accounts;

        // Safety: Setting the length reduces the previous length from the last
        // CPI call.
        //
        // Both allocate and assign instructions require one account (the
        // account being created).
        unsafe {
            cpi_accounts.set_len(1);
        }

        // Because the payer and account are writable signers, we can simply
        // overwrite the pubkey of the first account.
        cpi_accounts[0].pubkey = *account_key;

        {
            let cpi_data = &mut cpi_ix.data;

            cpi_data[0] = 8; // allocate selector
            cpi_data[SYSTEM_PROGRAM_SELECTOR_LEN..TRANSFER_ALLOCATE_DATA_LEN]
                .copy_from_slice(&(data_len as u64).to_le_bytes());

            invoke_signed_unchecked(&cpi_ix, accounts, &[])?;
        }

        {
            let cpi_data = &mut cpi_ix.data;

            // Safety: The capacity of this vector is 36. This data will be
            // overwritten for the next CPI call.
            unsafe {
                core::ptr::write_bytes(
                    cpi_data
                        .as_mut_ptr()
                        .offset(TRANSFER_ALLOCATE_DATA_LEN as isize),
                    0,
                    MAX_CPI_DATA_LEN - TRANSFER_ALLOCATE_DATA_LEN,
                );
                cpi_data.set_len(MAX_CPI_DATA_LEN);
            }

            cpi_data[0] = 1; // assign selector
            cpi_data[SYSTEM_PROGRAM_SELECTOR_LEN..MAX_CPI_DATA_LEN].copy_from_slice(&ID.to_bytes());

            invoke_signed_unchecked(&cpi_ix, accounts, &[])?;
        }
    }

    Ok(())
}

#[inline(always)]
fn compute_guardian_signatures_data_len(total_signatures: u8) -> usize {
    (total_signatures as usize) * GUARDIAN_SIGNATURE_LENGTH + GuardianSignatures::MINIMUM_SIZE
}
