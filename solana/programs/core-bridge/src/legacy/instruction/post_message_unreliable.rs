#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::{cpi::PostMessageArgs, instruction::LegacyInstruction};
    use solana_program::instruction::{AccountMeta, Instruction};

    /// This instruction handler is used to post a new message to the core bridge using an existing
    /// message account.
    ///
    /// The constraints for posting a message using this instruction handler are:
    /// * Emitter must be the same as the message account's emitter.
    /// * The new message must be the same size as the existing message's payload.
    pub fn post_message_unreliable(
        accounts: crate::legacy::accounts::PostMessageUnreliable,
        args: PostMessageArgs,
    ) -> Instruction {
        let fee_collector = accounts.fee_collector.unwrap_or(crate::ID);

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

        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::PostMessageUnreliable, args),
            accounts,
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
