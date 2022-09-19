//! Message type and helpers for Wormhole to emit messages.

use {
    crate::accounts::{
        Account,
        Config,
        Emitter,
        FeeCollector,
    },
    solana_program::{
        account_info::AccountInfo,
        entrypoint::ProgramResult,
        program::invoke_signed,
        program_error::ProgramError,
        pubkey::Pubkey,
    },
};

type ConsistencyLevel = u8;

/// A Message data type that can be emitted by Wormhole.
pub struct Message<'a> {
    /// A signed & writable account that can store the message.
    pub account:     Pubkey,
    /// Seeds (if needed) to derive the account key.
    pub seeds:       Option<&'a [&'a [&'a [u8]]]>,
    /// How many Solana blocks to wait before the message is considered safe.
    pub consistency: ConsistencyLevel,
    /// A unique number to identify the message; unused, can always be 0.
    pub nonce:       u32,
    /// The message itself!
    pub payload:     &'a [u8],
    /// A message can be marked as reliable or not. A reliable message will never be overwritten,
    /// and will be stored in the chain forever. An unreliable message will be overwritten if a new
    /// message with the same emitter is posted.
    pub reliable:    bool,
}

impl<'a> Message<'a> {
    /// Create a new (reliable) message with a given payload.
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
            reliable: true,
        }
    }

    /// Create a new (unreliable) message with a given payload.
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
            reliable: false,
        }
    }
}

/// This helper method wraps the steps required to emit Wormhole messages. See `Message` to see
/// different message types that can be emitted.
pub fn post_message(
    program_id: Pubkey,
    wormhole: Pubkey,
    payer: Pubkey,
    accounts: &[AccountInfo],
    message: Message,
) -> ProgramResult {
    // Derive wormhole accounts from Wormhole address.
    let fee_collector = FeeCollector::key(&wormhole, ());
    let config = Config::key(&wormhole, ());
    let (emitter, mut emitter_seeds, bump) = Emitter::key(&program_id, ());

    // Extend seeds with bump so it can be used to sign the message.
    let bump = &[bump];
    emitter_seeds.push(bump);

    // Filter for the Config AccountInfo so we can access its data.
    let config = accounts
        .iter()
        .find(|item| *item.key == config)
        .ok_or(ProgramError::NotEnoughAccountKeys)?;

    // Read Config account data.
    let config = Config::get(config).map_err(|_| ProgramError::InvalidAccountData)?;

    // Create a list of seed lists, the seeds for the emitter are inserted first.
    let mut seeds = vec![&*emitter_seeds];
    if let Some(v) = message.seeds {
        seeds.extend(v);
    }

    // Pay Wormhole transfer fee.
    invoke_signed(
        &solana_program::system_instruction::transfer(&payer, &fee_collector, config.params.fee),
        accounts,
        &[],
    )?;

    // Invoke the Wormhole post message endpoints to create an on-chain message.
    invoke_signed(
        &if message.reliable {
            crate::instruction::post_message(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload,
                message.consistency,
            )
            .unwrap()
        } else {
            crate::instruction::post_message_unreliable(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload,
                message.consistency,
            )
            .unwrap()
        },
        accounts,
        &seeds,
    )?;

    Ok(())
}
