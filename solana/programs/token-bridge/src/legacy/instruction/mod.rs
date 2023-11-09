use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize, Pubkey};

/// NOTE: No more instructions should be added to this enum. Instead, add them as Anchor instruction
/// handlers, which will inevitably live in lib.rs.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub enum LegacyInstruction {
    Initialize,
    AttestToken,
    CompleteTransferNative,
    CompleteTransferWrapped,
    TransferTokensWrapped,
    TransferTokensNative,
    RegisterChain,
    CreateOrUpdateWrapped,
    UpgradeContract,
    CompleteTransferWithPayloadNative,
    CompleteTransferWithPayloadWrapped,
    TransferTokensWithPayloadWrapped,
    TransferTokensWithPayloadNative,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct EmptyArgs {}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct InitializeArgs {
    _gap: [u8; 32],
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct AttestTokenArgs {
    pub nonce: u32,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TransferTokensArgs {
    pub nonce: u32,
    pub amount: u64,
    pub relayer_fee: u64,
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct TransferTokensWithPayloadArgs {
    pub nonce: u32,
    pub amount: u64,
    pub redeemer: [u8; 32],
    pub redeemer_chain: u16,
    pub payload: Vec<u8>,
    pub cpi_program_id: Option<Pubkey>,
}

#[cfg(feature = "no-entrypoint")]
mod __no_entrypoint {
    use crate::legacy::accounts;
    use anchor_lang::ToAccountMetas;
    use solana_program::instruction::Instruction;

    use super::*;

    pub fn complete_transfer_native(accounts: accounts::CompleteTransferNative) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferNative),
            accounts.to_account_metas(None),
        )
    }

    pub fn complete_transfer_wrapped(accounts: accounts::CompleteTransferWrapped) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferWrapped),
            accounts.to_account_metas(None),
        )
    }

    pub fn complete_transfer_with_payload_native(
        accounts: accounts::CompleteTransferWithPayloadNative,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferWithPayloadNative),
            accounts.to_account_metas(None),
        )
    }

    pub fn complete_transfer_with_payload_wrapped(
        accounts: accounts::CompleteTransferWithPayloadWrapped,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::CompleteTransferWithPayloadWrapped),
            accounts.to_account_metas(None),
        )
    }

    pub fn transfer_tokens_native(
        accounts: accounts::TransferTokensNative,
        args: TransferTokensArgs,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::TransferTokensNative, args),
            accounts.to_account_metas(None),
        )
    }

    pub fn transfer_tokens_wrapped(
        accounts: accounts::TransferTokensWrapped,
        args: TransferTokensArgs,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::TransferTokensWrapped, args),
            accounts.to_account_metas(None),
        )
    }

    pub fn transfer_tokens_with_payload_native(
        accounts: accounts::TransferTokensWithPayloadNative,
        args: TransferTokensWithPayloadArgs,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::TransferTokensWithPayloadNative, args),
            accounts.to_account_metas(None),
        )
    }

    pub fn transfer_tokens_with_payload_wrapped(
        accounts: accounts::TransferTokensWithPayloadWrapped,
        args: TransferTokensWithPayloadArgs,
    ) -> Instruction {
        Instruction::new_with_borsh(
            crate::ID,
            &(LegacyInstruction::TransferTokensWithPayloadWrapped, args),
            accounts.to_account_metas(None),
        )
    }
}

#[cfg(feature = "no-entrypoint")]
pub use __no_entrypoint::*;
