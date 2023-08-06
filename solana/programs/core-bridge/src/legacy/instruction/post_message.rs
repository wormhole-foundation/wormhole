use crate::types::Commitment;
use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// Arguments used to post a new Wormhole (Core Bridge) message either using
/// [post_message](crate::legacy::instruction::post_message) or
/// [post_message_unreliable](crate::legacy::instruction::post_message_unreliable).
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct PostMessageArgs {
    pub nonce: u32,
    pub payload: Vec<u8>,
    pub commitment: Commitment,
}

#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::LegacyInstruction;
    use solana_program::instruction::{AccountMeta, Instruction};

    use super::*;

    pub fn post_message(
        accounts: crate::legacy::accounts::PostMessage,
        args: PostMessageArgs,
    ) -> Instruction {
        let fee_collector = accounts.fee_collector.unwrap_or(crate::ID);
        let (emitter, emitter_is_signer) = match accounts.emitter {
            Some(emitter) => (emitter, true),
            None => (crate::ID, false),
        };

        // This part is a hack. But to avoid having to return a result, we assume that if the
        // payload provided is empty, then the message is already prepared (so the message does not
        // have to be a signer).
        let message_is_signer = !args.payload.is_empty();

        let accounts = vec![
            AccountMeta::new(accounts.config, false),
            AccountMeta::new(accounts.message, message_is_signer),
            AccountMeta::new_readonly(emitter, emitter_is_signer),
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
