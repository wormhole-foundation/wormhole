#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::instruction::LegacyInstruction;
    use solana_program::{
        instruction::{AccountMeta, Instruction},
        pubkey::Pubkey,
    };

    pub struct CompleteTransferWithPayloadNative {
        pub payer: Pubkey,
        pub posted_vaa: Pubkey,
        pub claim: Pubkey,
        pub registered_emitter: Pubkey,
        pub dst_token: Pubkey,
        pub redeemer_authority: Pubkey,
        pub custody_token: Pubkey,
        pub mint: Pubkey,
        pub custody_authority: Pubkey,
        pub system_program: Pubkey,
        pub core_bridge_program: Pubkey,
        pub token_program: Pubkey,
    }

    pub fn complete_transfer_with_payload_native(
        accounts: CompleteTransferWithPayloadNative,
    ) -> Instruction {
        let accounts = vec![
            AccountMeta::new(accounts.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(accounts.posted_vaa, false),
            AccountMeta::new(accounts.claim, false),
            AccountMeta::new_readonly(accounts.registered_emitter, false),
            AccountMeta::new(accounts.dst_token, false),
            AccountMeta::new_readonly(accounts.redeemer_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _relayer_fee_token
            AccountMeta::new(accounts.custody_token, false),
            AccountMeta::new_readonly(accounts.mint, false),
            AccountMeta::new_readonly(accounts.custody_authority, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(accounts.system_program, false),
            AccountMeta::new_readonly(accounts.core_bridge_program, false),
            AccountMeta::new_readonly(accounts.token_program, false),
        ];

        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferWithPayloadNative),
            accounts,
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
