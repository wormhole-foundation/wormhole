use borsh::BorshSerialize;
use solana_program::instruction::{
    AccountMeta,
    Instruction,
};
use solana_program::pubkey::Pubkey;
use solana_program::system_program;
use solana_program::sysvar::rent;
use solana_program::sysvar::clock;

use wormhole_sdk::{
    id,
    config,
    fee_collector,
    sequence,
};

use messenger_common::Message;

use crate::Instruction::{
    RecvMessage,
    SendMessage,
};


/// Create a RecvMessage instruction.
pub fn recv_message(program_id: Pubkey, payer: Pubkey, vaa: Pubkey) -> Instruction {
    Instruction {
        program_id,
        data: RecvMessage.try_to_vec().unwrap(),
        accounts: vec![
            AccountMeta::new_readonly(payer, true),
            AccountMeta::new_readonly(vaa, false),
        ],
    }
}


/// Create a SendMessage instruction.
pub fn send_message(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    payload: Message,
    nonce: u32,
) -> Instruction {
    let wormhole = id();
    let config = config(&wormhole);
    let fee_collector = fee_collector(&wormhole);
    let sequence = sequence(&wormhole, &emitter);

    // Note that accounts are passed in in order of useful-ness. The payer and message accounts are
    // used to invoke Wormhole. Many of the example send_message* instruction handlers will only
    // pop off as many accounts as required.
    Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(message, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new(config, false),
            AccountMeta::new_readonly(emitter, false),
            AccountMeta::new(sequence, false),
            AccountMeta::new_readonly(clock::id(), false),
            AccountMeta::new_readonly(rent::id(), false),
            AccountMeta::new_readonly(system_program::id(), false),
            AccountMeta::new_readonly(wormhole, false),
        ],
        data: SendMessage(payload, nonce).try_to_vec().unwrap(),
    }
}

/// Create a SendMessageRaw instruction. This does the same as SendMessage however the instruction
/// handler does not use the Wormhole SDK helper API.
pub fn send_message_raw(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    payload: Message,
    nonce: u32,
) -> Instruction {
    let wormhole = id();
    let config = config(&wormhole);
    let fee_collector = fee_collector(&wormhole);
    let sequence = sequence(&wormhole, &emitter);

    // Note that accounts are passed in in order of useful-ness. The payer and message accounts are
    // used to invoke Wormhole. Many of the example send_message* instruction handlers will only
    // pop off as many accounts as required.
    Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(payer, true),
            AccountMeta::new(message, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new(config, false),
            AccountMeta::new_readonly(emitter, false),
            AccountMeta::new(sequence, false),
            AccountMeta::new_readonly(clock::id(), false),
            AccountMeta::new_readonly(rent::id(), false),
            AccountMeta::new_readonly(system_program::id(), false),
            AccountMeta::new_readonly(wormhole, false),
        ],
        data: SendMessage(payload, nonce).try_to_vec().unwrap(),
    }
}
