#![deny(dead_code, unused_imports, unused_mut, unused_variables)]

use solana_program::{
    account_info::{next_account_info, AccountInfo},
    declare_id,
    entrypoint::ProgramResult,
    instruction::{AccountMeta, Instruction},
    msg,
    program_error::ProgramError,
    pubkey::Pubkey,
};
use wormhole_svm_definitions::{
    find_shim_message_address, CORE_BRIDGE_PROGRAM_ID, MESSAGE_AUTHORITY_SEED,
    POST_MESSAGE_SHIM_EVENT_AUTHORITY, POST_MESSAGE_SHIM_PROGRAM_ID,
};
use wormhole_svm_shim::post_message::{
    EmitMessageData, PostMessageData, PostMessageShimInstruction,
};

declare_id!(POST_MESSAGE_SHIM_PROGRAM_ID);

const MESSAGE_AUTHORITY_SIGNER_SEEDS: &[&[u8]; 2] = &[MESSAGE_AUTHORITY_SEED, &[255]];

solana_program::entrypoint!(process_instruction);

#[inline]
fn process_instruction(
    program_id: &Pubkey,
    accounts: &[AccountInfo],
    instruction_data: &[u8],
) -> ProgramResult {
    // Verify the program ID is what we expect.
    if program_id != &POST_MESSAGE_SHIM_PROGRAM_ID {
        return Err(ProgramError::IncorrectProgramId);
    }

    // Determine whether instruction data is self CPI or post message.
    match PostMessageShimInstruction::deserialize(instruction_data) {
        Some(PostMessageShimInstruction::PostMessage(data)) => process_post_message(accounts, data),
        Some(PostMessageShimInstruction::EmitMessage(_)) => process_emit_message(accounts),
        None => Err(ProgramError::InvalidInstructionData),
    }
}

#[inline]
fn process_emit_message(accounts: &[AccountInfo]) -> ProgramResult {
    // Verify account.
    let message_authority_info = next_account_info(&mut accounts.iter())?;

    if !message_authority_info.is_signer {
        msg!("Message authority (account #1) is not signer");
        return Err(ProgramError::MissingRequiredSignature);
    }

    if message_authority_info.key != &POST_MESSAGE_SHIM_EVENT_AUTHORITY {
        msg!("Message authority (account #1) seeds constraint violated");
        msg!("Left:");
        msg!("{}", message_authority_info.key);
        msg!("Right:");
        msg!("{}", POST_MESSAGE_SHIM_EVENT_AUTHORITY);
        return Err(ProgramError::InvalidSeeds);
    }

    Ok(())
}

#[inline]
fn process_post_message(accounts: &[AccountInfo], data: PostMessageData) -> ProgramResult {
    // Verify accounts.
    let accounts_iter = &mut accounts.iter();

    // Wormhole Core Bridge config. Wormhole Core Bridge program's post message
    // instruction requires this account be mutable.
    let bridge_info = next_account_info(accounts_iter)?;

    // Wormhole Message. Wormhole Core Bridge program's post message
    // instruction requires this account to be a mutable signer.
    let message_info = next_account_info(accounts_iter)?;

    // Emitter of the Wormhole Core Bridge message. Wormhole Core Bridge
    // program's post message instruction requires this account to be a signer.
    let emitter_info = next_account_info(accounts_iter)?;
    let emitter_key = emitter_info.key;

    // The Wormhole Post Message Shim program uses a PDA per emitter because
    // these messages are already bottle-necked by sequence and the Wormhole
    // Core Bridge program enforces that the emitter must be identical for
    // reused accounts.
    //
    // While this could be managed by the integrator, it seems more effective
    // to have the Wormhole Post Message Shim program manage these accounts.
    let (expected_message_key, message_bump) = find_shim_message_address(emitter_key);
    if message_info.key != &expected_message_key {
        msg!("Message (account #2) seeds constraint violated");
        msg!("Left:");
        msg!("{}", message_info.key);
        msg!("Right:");
        msg!("{}", expected_message_key);
        return Err(ProgramError::InvalidSeeds);
    }

    // Emitter's sequence account. Wormhole Core Bridge program's post message
    // instruction requires this account to be mutable.
    let sequence_info = next_account_info(accounts_iter)?;

    // Payer will pay the Wormhole Core Bridge fee to post a message. The fee
    // amount is encoded in the [core_bridge_config] account data.
    let payer_info = next_account_info(accounts_iter)?;

    // Wormhole Core Bridge fee collector. Wormhole Core Bridge program's post
    // message instruction requires this account to be mutable.
    let fee_collector_info = next_account_info(accounts_iter)?;

    // Clock sysvar, which will be used after the message is posted.
    let clock_info = next_account_info(accounts_iter)?;

    // System program. Wormhole Core Bridge program's post message instruction
    // requires the System program to create the message account and emitter
    // sequence account if they have not been created yet.
    let system_program_info = next_account_info(accounts_iter)?;

    // Rent sysvar. Wormhole Core Bridge program's post message instruction
    // requires this account.
    let rent_info = next_account_info(accounts_iter)?;

    // Wormhole Core Bridge program.
    let wormhole_program_info = next_account_info(accounts_iter)?;

    // We only want to use the Wormhole Core Bridge program to post messages.
    if wormhole_program_info.key != &CORE_BRIDGE_PROGRAM_ID {
        msg!("Wormhole program (account #10) address constraint violated");
        return Err(ProgramError::InvalidAccountData);
    }

    // Wormhole Post Message Shim program's self-CPI authority. The emit message
    // instruction will verify that this account is a signer and the pubkey is
    // correct.
    let event_authority_info = next_account_info(accounts_iter)?;

    let PostMessageData {
        nonce,
        finality,
        payload: _,
    } = data;

    // Using the data passed in from the instruction, we can construct the
    // post message unreliable data. If the guardians do not end up checking the
    // message account, we can just use zeros for the nonce and finality.
    let mut post_message_unreliable_data = Vec::with_capacity({
        1 // selector
        + 4 // nonce
        + 4 // payload length (zero)
        + 1 // finality
    });
    post_message_unreliable_data.push(8); // selector
    post_message_unreliable_data.extend_from_slice(&nonce.to_le_bytes());
    post_message_unreliable_data.extend_from_slice(&u32::to_le_bytes(0));
    post_message_unreliable_data.push(finality as u8);

    let post_message_unreliable_ix = Instruction {
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
            AccountMeta::new_readonly(*rent_info.key, false),
        ],
        data: post_message_unreliable_data,
    };

    // There is no account data being borrowed at this point, so this is safe.
    solana_program::program::invoke_signed_unchecked(
        &post_message_unreliable_ix,
        accounts,
        &[&[emitter_key.as_ref(), &[message_bump]]],
    )?;

    let event_cpi_ix = {
        // Parse the sequence from the account and emit the event reading the
        // account after avoids having to handle when the account doesn't exist.
        let sequence = {
            let account_data = sequence_info.data.borrow();
            u64::from_le_bytes(account_data[..8].try_into().unwrap()).saturating_sub(1)
        };

        // NOTE: If the Wormhole Core Bridge ever changes to use Clock::get(), this clock account
        // info may not exist anymore (so we will have to perform Clock::get() here instead).
        let clock_data = clock_info.data.borrow();
        let submission_time = i64::from_le_bytes(clock_data[32..40].try_into().unwrap());

        Instruction {
            program_id: ID,
            accounts: vec![AccountMeta::new_readonly(*event_authority_info.key, true)],
            data: PostMessageShimInstruction::EmitMessage(EmitMessageData {
                emitter: *emitter_key,
                sequence,
                submission_time: submission_time as u32,
            })
            .to_vec(),
        }
    };

    // There is no account data being borrowed at this point, so this is safe.
    solana_program::program::invoke_signed_unchecked(
        &event_cpi_ix,
        accounts,
        &[MESSAGE_AUTHORITY_SIGNER_SEEDS],
    )?;

    Ok(())
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn test_message_authority_signer_seeds() {
        let actual = Pubkey::create_program_address(
            MESSAGE_AUTHORITY_SIGNER_SEEDS,
            &POST_MESSAGE_SHIM_PROGRAM_ID,
        )
        .unwrap();
        assert_eq!(actual, POST_MESSAGE_SHIM_EVENT_AUTHORITY);
    }
}
