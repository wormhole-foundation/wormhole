use crate::types::Commitment;
use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyPostMessageArgs {
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

    pub struct PostMessage {
        pub config: Pubkey,
        pub message: Pubkey,
        pub emitter: Pubkey,
        pub emitter_sequence: Pubkey,
        pub payer: Pubkey,
        pub fee_collector: Option<Pubkey>,
        pub system_program: Pubkey,
    }

    pub fn post_message(accounts: PostMessage, args: LegacyPostMessageArgs) -> Instruction {
        let fee_collector = match accounts.fee_collector {
            Some(fee_collector) => fee_collector,
            None => Pubkey::default(),
        };

        let accounts = vec![
            AccountMeta::new(accounts.config, false),
            AccountMeta::new(accounts.message, true),
            AccountMeta::new_readonly(accounts.emitter, true),
            AccountMeta::new(accounts.emitter_sequence, false),
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new(fee_collector, false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(accounts.system_program, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
        ];

        Instruction::new_with_borsh(crate::ID, &(LegacyInstruction::PostMessage, args), accounts)
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
