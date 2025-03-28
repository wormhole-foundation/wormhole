#![deny(dead_code, unused_imports, unused_mut, unused_variables)]

use solana_program::{
    account_info::AccountInfo,
    entrypoint::ProgramResult,
    instruction::{AccountMeta, Instruction},
    msg,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use wormhole_svm_definitions::{
    find_shim_message_address, Finality, ANCHOR_EVENT_CPI_SELECTOR, CORE_BRIDGE_PROGRAM_ID,
    EVENT_AUTHORITY_SEED, MESSAGE_EVENT_DISCRIMINATOR, POST_MESSAGE_SHIM_EVENT_AUTHORITY,
    POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP, POST_MESSAGE_SHIM_PROGRAM_ID as ID,
};
use wormhole_svm_shim::post_message::PostMessageShimInstruction;

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

    // Determine whether instruction data is post message.
    match PostMessageShimInstruction::<Finality>::deserialize(instruction_data) {
        Some(PostMessageShimInstruction::PostMessage(_)) => process_post_message(accounts),
        None => process_privileged_invoke(accounts),
    }
}

#[inline(always)]
fn process_post_message(accounts: &[AccountInfo]) -> ProgramResult {
    // This instruction requires 11 accounts. If there are more remaining, we
    // won't do anything with them. We perform this check upfront so we can
    // index into the accounts slice.
    if accounts.len() < 11 {
        return Err(ProgramError::NotEnoughAccountKeys);
    }

    // Verify accounts.

    // Wormhole Core Bridge config. Wormhole Core Bridge program's post message
    // instruction requires this account be mutable.
    let bridge_info = &accounts[0];

    // Wormhole Message. Wormhole Core Bridge program's post message
    // instruction requires this account to be a mutable signer.
    let message_info = &accounts[1];

    // Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge
    // program's post message instruction requires this account to be a signer.
    let emitter_info = &accounts[2];
    let emitter_key = emitter_info.key;

    // The Wormhole Post Message Shim program uses a PDA per emitter because
    // these messages are already bottle-necked by sequence and the Wormhole
    // Core Bridge program enforces that the emitter must be identical for
    // reused accounts.
    //
    // While this could be managed by the integrator, it seems more effective
    // to have the Wormhole Post Message Shim program manage these accounts.
    let (expected_message_key, message_bump) = find_shim_message_address(emitter_key, &ID);
    if message_info.key != &expected_message_key {
        msg!("Message (account #2) seeds constraint violated");
        msg!("Actual:");
        msg!("{}", message_info.key);
        msg!("Expected:");
        msg!("{}", expected_message_key);
        return Err(ProgramError::InvalidSeeds);
    }

    // Emitter's sequence account. Wormhole Core Bridge program's post message
    // instruction requires this account to be mutable.
    let sequence_info = &accounts[3];

    // Payer will pay the rent for the Wormhole Core Bridge emitter sequence
    // and message on the first post message call. Subsequent calls will not
    // require more lamports for rent.
    let payer_info = &accounts[4];

    // Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post
    // message instruction requires this account to be mutable.
    let fee_collector_info = &accounts[5];

    // Clock sysvar, which will be used after the message is posted.
    let clock_info = &accounts[6];

    // System program. Wormhole Core Bridge program's post message instruction
    // requires the System program to create the message account and emitter
    // sequence account if they have not been created yet.
    let system_program_info = &accounts[7];

    // Wormhole Core Bridge program.
    let wormhole_program_info = &accounts[8];

    // We only want to use the Wormhole Core Bridge program to post messages.
    if wormhole_program_info.key != &CORE_BRIDGE_PROGRAM_ID {
        msg!("Wormhole program (account #9) address constraint violated");
        return Err(ProgramError::InvalidAccountData);
    }

    // Wormhole Post Message Shim program's self-CPI authority. The emit message
    // instruction will verify that this account is a signer and the pubkey is
    // correct.
    let event_authority_info = &accounts[9];

    // NOTE: We are not checking account at index == 10 because the self CPI
    // will fail if this program executable is not present.

    // Perform two CPIs:
    // 1. Post the message.
    // 2. Emit the event (with self CPI)
    //
    // The max length of instruction data is 60 bytes between the two
    // instructions, so we will reuse the same allocated memory for both.
    const MAX_CPI_DATA_LEN: usize = 60;
    const POST_MESSAGE_UNRELIABLE_DATA_LEN: usize = 10;

    // The post message unreliable instruction needs more accounts than the self
    // CPI call (which only needs one). We will initialize the instruction with
    // the Wormhole Core Bridge program's ID and the post message accounts.
    let mut cpi_ix = Instruction {
        program_id: *wormhole_program_info.key,
        accounts: vec![
            AccountMeta::new(*bridge_info.key, false),
            AccountMeta::new(expected_message_key, true),
            AccountMeta::new_readonly(*emitter_key, true),
            AccountMeta::new(*sequence_info.key, false),
            AccountMeta::new(*payer_info.key, true),
            AccountMeta::new(*fee_collector_info.key, false),
            AccountMeta::new_readonly(*clock_info.key, false),
            AccountMeta::new_readonly(*system_program_info.key, false),
        ],
        data: Vec::with_capacity(MAX_CPI_DATA_LEN),
    };

    let emitter_key_bytes = emitter_key.to_bytes();

    // First post message.
    {
        let cpi_data = &mut cpi_ix.data;

        // Safety: Because the capacity is > 10, it is safe to write to the
        // first 10 elements and set the vector's length to 10.
        //
        // Encoding the post message instruction data is:
        // 1. 1 byte for the selector.
        // 2. 4 bytes for the nonce.
        // 3. 4 bytes for the payload length (zero in this case).
        // 4. 1 byte for the consistency level (finality).
        //
        // Nonce and consistency level will not be read from the Wormhole Core
        // Bridge message account (only from this program's instruction data),
        // so we might as well set them to zero.
        unsafe {
            core::ptr::write_bytes(cpi_data.as_mut_ptr(), 0, POST_MESSAGE_UNRELIABLE_DATA_LEN);
            cpi_data.set_len(POST_MESSAGE_UNRELIABLE_DATA_LEN);
        }

        // We only need to encode the selector for post message unreliable.
        cpi_data[0] = 8;

        // There is no account data being borrowed at this point that the CPI'ed
        // program will be using, so this is safe.
        solana_program::program::invoke_signed_unchecked(
            &cpi_ix,
            accounts,
            &[&[&emitter_key_bytes, &[message_bump]]],
        )?;
    }

    // Finally perform self CPI.
    {
        // Parse the sequence from the account and emit the event reading the
        // account after avoids having to handle when the account doesn't exist.
        let sequence_info_data = sequence_info.data.borrow();
        let sequence = u64::from_le_bytes(sequence_info_data[..8].try_into().unwrap()) - 1;

        // NOTE: If the Wormhole Core Bridge ever changes to use Clock::get(),
        // this clock account info may not exist anymore (so we will have to
        // perform Clock::get() here instead).
        let clock_info_data = clock_info.data.borrow();
        let encoded_unix_timestamp = &clock_info_data[32..40];

        // Casting to u32 matches the Wormhole Core Bridge's timestamp. This
        // operation will panic quite far in the future.
        //
        // Clock encodes unix timestamp as little-endian i64, so the first 4
        // bytes will be the timestamp encoded as u32. We expect the last 4
        // bytes to be zero.
        assert!(encoded_unix_timestamp[4..] == [0, 0, 0, 0]);

        let cpi_accounts = &mut cpi_ix.accounts;

        // Safety: Setting the length reduces the previous length from the last
        // CPI call.
        unsafe {
            cpi_accounts.set_len(1);
        }
        cpi_accounts[0] = AccountMeta::new_readonly(*event_authority_info.key, true);

        // "Emit" the MessageEvent. Its schema is:
        //
        // struct MessageEvent {
        //     emitter: Pubkey,
        //     sequence: u64,
        //     submission_time: u32,
        // }
        //
        // The invoke below emulates the Anchor event CPI pattern.
        let cpi_data = &mut cpi_ix.data;

        // Safety: The capacity of this vector is 60. This data will be
        // overwritten for the next CPI call.
        unsafe {
            core::ptr::write_bytes(
                cpi_data
                    .as_mut_ptr()
                    .offset(POST_MESSAGE_UNRELIABLE_DATA_LEN as isize),
                0,
                MAX_CPI_DATA_LEN - POST_MESSAGE_UNRELIABLE_DATA_LEN,
            );
            cpi_data.set_len(MAX_CPI_DATA_LEN);
        }
        cpi_data[..8].copy_from_slice(&ANCHOR_EVENT_CPI_SELECTOR);
        cpi_data[8..16].copy_from_slice(&MESSAGE_EVENT_DISCRIMINATOR);
        cpi_data[16..48].copy_from_slice(&emitter_key_bytes);
        cpi_data[48..56].copy_from_slice(&sequence.to_le_bytes());
        cpi_data[56..60].copy_from_slice(&encoded_unix_timestamp[..4]);

        cpi_ix.program_id = ID;

        // There is no account data being borrowed at this point that the CPI'ed
        // program will be using, so this is safe.
        solana_program::program::invoke_signed_unchecked(
            &cpi_ix,
            accounts,
            &[&[
                EVENT_AUTHORITY_SEED,
                &[POST_MESSAGE_SHIM_EVENT_AUTHORITY_BUMP],
            ]],
        )?;
    }

    Ok(())
}

// Event CPI.

#[inline(always)]
fn process_privileged_invoke(accounts: &[AccountInfo]) -> ProgramResult {
    // We want to ensure that this program's authority (which is the Anchor
    // event CPI PDA) is the one invoking this instruction. If anything else
    // invokes this instruction, we will pretend that the instruction does not
    // exist by reverting with `InvalidInstructionData`.

    if accounts.is_empty()
        || !accounts[0].is_signer
        || accounts[0].key != &POST_MESSAGE_SHIM_EVENT_AUTHORITY
    {
        return Err(ProgramError::InvalidInstructionData);
    }

    Ok(())
}
