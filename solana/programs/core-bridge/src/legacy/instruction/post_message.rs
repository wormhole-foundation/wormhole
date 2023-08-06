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
        pub bridge: Pubkey,
        pub message: Pubkey,
        pub emitter: Pubkey,
        pub emitter_sequence: Pubkey,
        pub payer: Pubkey,
        pub fee_collector: Pubkey,
        pub system_program: Pubkey,
    }

    pub fn post_message(accounts: PostMessage, args: LegacyPostMessageArgs) -> Instruction {
        let accounts = vec![
            AccountMeta::new(accounts.bridge, false),
            AccountMeta::new(accounts.message, false),
            AccountMeta::new(accounts.emitter, true),
            AccountMeta::new(accounts.emitter_sequence, false),
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new(accounts.fee_collector, false),
            AccountMeta::new_readonly(Default::default(), false), // _clock
            AccountMeta::new_readonly(Default::default(), false), // _rent
            AccountMeta::new_readonly(accounts.system_program, false),
        ];

        Instruction::new_with_borsh(crate::ID, &LegacyInstruction::PostMessage(args), accounts)
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
