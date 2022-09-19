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

/// Solana-specific Consistency Level: 1 for optimistic confirmation, 32 for full confirmation.
#[repr(u8)]
pub enum ConsistencyLevel {
    Confirmed = 1,
    Finalized = 32,
}

/// Solana-specific enum describing a Messages on-chain persistence strategy.
pub enum Reliability {
    /// The message will be persisted on-chain forever.
    Permanent,
    /// The message can be replaced by newer messages, useful for low-priority messages.
    Ephemeral,
}

/// A Message data type that can be emitted by Wormhole.
pub struct Message<'a> {
    /// A signed & writable account that can store the message.
    pub account:     Pubkey,
    /// Seeds (if needed) to derive the account key.
    pub seeds:       Option<&'a [&'a [&'a [u8]]]>,
    /// How many Solana blocks to wait before the message is considered safe.
    pub consistency: ConsistencyLevel,
    /// A unique nonce to identify the message; unused/deprecated, can always be 0.
    pub nonce:       u32,
    /// The message payload itself.
    pub payload:     &'a [u8],
    /// Mark message reliability.
    /// A reliable message will never be overwritten, and so is stored on-chain forever. On the
    /// other hand an ephemeral message will be overwritten by a new message with the same emitter
    /// which allows for space-saving. Use ephemeral only when you are sure that missing messages
    /// is not critical to your application.
    pub reliable:    Reliability,
}

impl<'a> Message<'a> {
    /// Create a new (reliable) message with a given payload.
    pub fn new(
        account: Pubkey,
        seeds: Option<&'a [&'a [&'a [u8]]]>,
        consistency: ConsistencyLevel,
        payload: &'a [u8],
    ) -> Self {
        Self {
            account,
            seeds,
            consistency,
            nonce: 0,
            payload,
            reliable: Reliability::Permanent,
        }
    }

    /// Create a new (ephemeral) message with a given payload.
    pub fn new_ephemeral(
        account: Pubkey,
        seeds: Option<&'a [&'a [&'a [u8]]]>,
        consistency: ConsistencyLevel,
        payload: &'a [u8],
    ) -> Self {
        Self {
            account,
            seeds,
            consistency,
            nonce: 0,
            payload,
            reliable: Reliability::Ephemeral,
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

    // Pay Wormhole transfer fee (if there is one).
    if config.params.fee > 0 {
        invoke_signed(
            &solana_program::system_instruction::transfer(
                &payer,
                &fee_collector,
                config.params.fee,
            ),
            accounts,
            &[],
        )?;
    }

    // Invoke the Wormhole post message endpoints to create an on-chain message.
    invoke_signed(
        &match message.reliable {
            Reliability::Permanent => crate::instructions::post_message(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload,
                message.consistency as u8,
            )
            .unwrap(),

            Reliability::Ephemeral => crate::instructions::post_message_unreliable(
                wormhole,
                payer,
                emitter,
                message.account,
                message.nonce,
                message.payload,
                message.consistency as u8,
            )
            .unwrap(),
        },
        accounts,
        &seeds,
    )?;

    Ok(())
}

#[cfg(test)]
mod testing {
    use super::*;

    #[test]
    fn test_message_emission() {
        let _ = post_message(
            Pubkey::new_unique(),
            Pubkey::new_unique(),
            Pubkey::new_unique(),
            &[],
            Message::new(
                Pubkey::new_unique(),
                None,
                ConsistencyLevel::Confirmed,
                &[0, 1, 2, 3, 4, 5, 6, 7, 8, 9],
            ),
        );
    }
}
