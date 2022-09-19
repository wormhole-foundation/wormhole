//! Functions for creating instructions for CPI calls.

use {
    crate::{
        accounts::Account,
        Config,
        FeeCollector,
        Sequence,
    },
    borsh::BorshSerialize,
    solana_program::{
        instruction::{
            AccountMeta,
            Instruction,
        },
        pubkey::Pubkey,
        sysvar,
    },
    wormhole::WormholeError,
};

#[derive(Debug, PartialEq, BorshSerialize)]
struct PostMessageData<'a> {
    nonce:             u32,
    payload:           &'a [u8],
    consistency_level: u8,
}

pub fn post_message(
    wormhole: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: &[u8],
    consistency_level: u8,
) -> Result<Instruction, WormholeError> {
    let bridge = Config::key(&wormhole, ());
    let fee_collector = FeeCollector::key(&wormhole, ());
    let sequence = Sequence::key(&wormhole, emitter);

    Ok(Instruction {
        program_id: wormhole,
        accounts:   vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(message, true),
            AccountMeta::new_readonly(emitter, true),
            AccountMeta::new(sequence, false),
            AccountMeta::new(payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data:       (
            1,
            PostMessageData {
                nonce,
                payload,
                consistency_level,
            },
        )
            .try_to_vec()?,
    })
}

pub fn post_message_unreliable(
    program_id: Pubkey,
    payer: Pubkey,
    emitter: Pubkey,
    message: Pubkey,
    nonce: u32,
    payload: &[u8],
    consistency_level: u8,
) -> Result<Instruction, WormholeError> {
    let bridge = Config::key(&program_id, ());
    let fee_collector = FeeCollector::key(&program_id, ());
    let sequence = Sequence::key(&program_id, emitter);

    Ok(Instruction {
        program_id,
        accounts: vec![
            AccountMeta::new(bridge, false),
            AccountMeta::new(message, true),
            AccountMeta::new_readonly(emitter, true),
            AccountMeta::new(sequence, false),
            AccountMeta::new(payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(sysvar::clock::id(), false),
            AccountMeta::new_readonly(sysvar::rent::id(), false),
            AccountMeta::new_readonly(solana_program::system_program::id(), false),
        ],
        data: (
            1,
            PostMessageData {
                nonce,
                payload,
                consistency_level,
            },
        )
            .try_to_vec()?,
    })
}
