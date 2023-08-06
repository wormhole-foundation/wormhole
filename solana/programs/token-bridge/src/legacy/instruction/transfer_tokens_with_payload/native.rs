#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::{LegacyInstruction, TransferTokensWithPayloadArgs};
    use solana_program::{
        instruction::{AccountMeta, Instruction},
        pubkey::Pubkey,
    };

    pub struct TransferTokensWithPayloadNative {
        pub payer: Pubkey,
        pub src_token: Pubkey,
        pub mint: Pubkey,
        pub custody_token: Pubkey,
        pub transfer_authority: Pubkey,
        pub custody_authority: Pubkey,
        pub core_bridge_config: Pubkey,
        pub core_message: Pubkey,
        pub core_emitter: Pubkey,
        pub core_emitter_sequence: Pubkey,
        pub core_fee_collector: Option<Pubkey>,
        pub sender_authority: Pubkey,
        pub system_program: Pubkey,
        pub core_bridge_program: Pubkey,
        pub token_program: Pubkey,
    }

    pub fn transfer_tokens_with_payload_native(
        accounts: TransferTokensWithPayloadNative,
        args: TransferTokensWithPayloadArgs,
    ) -> Instruction {
        let core_fee_collector = accounts.core_fee_collector.unwrap_or(crate::ID);

        let accounts = vec![
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new(accounts.src_token, false),
            AccountMeta::new_readonly(accounts.mint, false),
            AccountMeta::new(accounts.custody_token, false),
            AccountMeta::new_readonly(accounts.transfer_authority, false),
            AccountMeta::new_readonly(accounts.custody_authority, false),
            AccountMeta::new(accounts.core_bridge_config, false),
            AccountMeta::new(accounts.core_message, true),
            AccountMeta::new_readonly(accounts.core_emitter, false),
            AccountMeta::new(accounts.core_emitter_sequence, false),
            AccountMeta::new(core_fee_collector, false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(accounts.sender_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(accounts.system_program, false),
            AccountMeta::new_readonly(accounts.core_bridge_program, false),
            AccountMeta::new_readonly(accounts.token_program, false),
        ];

        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::TransferTokensWithPayloadNative, args),
            accounts,
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
