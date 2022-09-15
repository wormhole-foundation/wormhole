//! Message type and helpers for Wormhole to emit messages.

use {
    crate::accounts::{
        config,
        emitter,
        fee_collector,
        read_config,
    },
    bridge::types::ConsistencyLevel,
    solana_program::{
        account_info::AccountInfo,
        entrypoint::ProgramResult,
        program::invoke_signed,
        program_error::ProgramError,
        pubkey::Pubkey,
    },
};

pub struct Message<'a> {
    pub account:     Pubkey,
    pub seeds:       Option<&'a [&'a [&'a [u8]]]>,
    pub consistency: ConsistencyLevel,
    pub nonce:       u32,
    pub payload:     &'a [u8],
    pub reliable:    bool,
}

impl<'a> Message<'a> {
    pub fn new(
        account: Pubkey,
        seeds: Option<&'a [&'a [&'a [u8]]]>,
        consistency: ConsistencyLevel,
        nonce: u32,
        payload: &'a [u8],
    ) -> Self {
        Self {
            account,
            seeds,
            consistency,
            nonce,
            payload,
            reliable: Reliability::Permanent,
        }
    }

    pub fn new_unrelible(
        account: Pubkey,
        seeds: Option<&'a [&'a [&'a [u8]]]>,
        consistency: ConsistencyLevel,
        nonce: u32,
        payload: &'a [u8],
    ) -> Self {
        Self {
            account,
            seeds,
            consistency,
            nonce,
            payload,
            reliable: Reliability::Ephemeral,
        }
    }
}

/// This helper method wraps the steps required to invoke Wormhole, it takes care of fee payment,
/// emitter derivation, and function invocation. This will be the right thing to use if you need to
/// simply emit a message in the most straight forward way possible.
pub fn post_message(
    program_id: Pubkey,
    wormhole: Pubkey,
    payer: Pubkey,
    accounts: &[AccountInfo],
    message: Message,
) -> ProgramResult {
    // Derive wormhole accounts from Wormhole address.
    let fee_collector = fee_collector(&wormhole);
    let config = config(&wormhole);
    let (emitter, mut emitter_seeds, bump) = emitter(&program_id);

    // Extend seeds with bump so it can be used to sign the message.
    let bump = &[bump];
    emitter_seeds.push(bump);

    // Filter for the Config AccountInfo so we can access its data.
    let config = accounts
        .iter()
        .find(|item| *item.key == config)
        .ok_or(ProgramError::NotEnoughAccountKeys)?;

    // Read Config account data.
    let config = read_config(config).map_err(|_| ProgramError::InvalidAccountData)?;

    // Create a list of seed lists, the seeds for the emitter are inserted first.
    let mut seeds = vec![&*emitter_seeds];
    if let Some(v) = message.seeds {
        seeds.extend(v);
    }

    // Pay Wormhole transfer fee.
    invoke_signed(
        &solana_program::system_instruction::transfer(&payer, &fee_collector, config.fee),
        accounts,
        &[],
    )?;

    // Invoke the Wormhole post message endpoints to create an on-chain message.
    if message.reliable {
        invoke_signed(
            &bridge::instructions::post_message(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload.to_vec(),
                message.consistency,
            )
            .unwrap(),
            accounts,
            &seeds,
        )?;
    } else {
        invoke_signed(
            &bridge::instructions::post_message_unreliable(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload.to_vec(),
                message.consistency,
            )
            .unwrap(),
            accounts,
            &seeds,
        )?;
    }

    Ok(())
}
