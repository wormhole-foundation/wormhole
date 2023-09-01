#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::LegacyInstruction;
    use solana_program::{
        instruction::{AccountMeta, Instruction},
        pubkey::Pubkey,
    };

    pub struct CompleteTransferNative {
        pub payer: Pubkey,
        pub posted_vaa: Pubkey,
        pub claim: Pubkey,
        pub registered_emitter: Pubkey,
        pub recipient_token: Pubkey,
        pub payer_token: Pubkey,
        pub custody_token: Pubkey,
        pub mint: Pubkey,
        pub custody_authority: Pubkey,
        pub recipient: Option<Pubkey>,
        pub system_program: Pubkey,
        pub core_bridge_program: Pubkey,
        pub token_program: Pubkey,
    }

    pub fn complete_transfer_native(accounts: CompleteTransferNative) -> Instruction {
        let recipient = accounts.recipient.unwrap_or(crate::ID);

        let accounts = vec![
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(accounts.posted_vaa, false),
            AccountMeta::new(accounts.claim, false),
            AccountMeta::new_readonly(accounts.registered_emitter, false),
            AccountMeta::new(accounts.recipient_token, false),
            AccountMeta::new(accounts.payer_token, false),
            AccountMeta::new(accounts.custody_token, false),
            AccountMeta::new_readonly(accounts.mint, false),
            AccountMeta::new_readonly(accounts.custody_authority, false),
            AccountMeta::new_readonly(recipient, false),
            AccountMeta::new_readonly(accounts.system_program, false),
            AccountMeta::new_readonly(accounts.core_bridge_program, false),
            AccountMeta::new_readonly(accounts.token_program, false),
        ];

        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferNative),
            accounts,
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
