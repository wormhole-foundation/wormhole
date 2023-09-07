use crate::types::Commitment;
use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyPostMessageUnreliableArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
    pub commitment: Commitment,
}

#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::LegacyInstruction;
    use solana_program::{
        instruction::{AccountMeta, Instruction},
        pubkey::Pubkey,
    };

    use super::*;

    pub struct PostMessageUnreliable {
        pub bridge: Pubkey,
        pub message: Pubkey,
        pub emitter: Pubkey,
        pub emitter_sequence: Pubkey,
        pub payer: Pubkey,
        pub fee_collector: Option<Pubkey>,
        pub system_program: Pubkey,
    }

    /// This instruction handler is used to post a new message to the core bridge using an existing
    /// message account.
    ///
    /// The constraints for posting a message using this instruction handler are:
    /// * Emitter must be the same as the message account's emitter.
    /// * The new message must be the same size as the existing message's payload.
    pub fn post_message_unreliable(
        accounts: PostMessageUnreliable,
        args: LegacyPostMessageUnreliableArgs,
    ) -> Instruction {
        let fee_collector = match accounts.fee_collector {
            Some(fee_collector) => fee_collector,
            None => Pubkey::default(),
        };

        let accounts = vec![
            AccountMeta::new(accounts.bridge, false),
            AccountMeta::new(accounts.message, true),
            AccountMeta::new_readonly(accounts.emitter, true),
            AccountMeta::new(accounts.emitter_sequence, false),
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(accounts.system_program, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
        ];

        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::PostMessageUnreliable, args),
            accounts,
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
