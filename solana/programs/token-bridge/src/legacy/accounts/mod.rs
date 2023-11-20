//! A set of structs mirroring the structs deriving [Accounts](anchor_lang::prelude::Accounts),
//! where each field is a [Pubkey]. This is useful for specifying self for a client.
//!
//! NOTE: This is similar to how [accounts](mod@crate::accounts) is generated via Anchor's
//! [program][anchor_lang::prelude::program] macro.

use anchor_lang::prelude::{Pubkey, ToAccountMetas};
use solana_program::instruction::AccountMeta;

pub struct CompleteTransferNative {
    pub payer: Pubkey,
    pub vaa: Pubkey,
    pub claim: Pubkey,
    pub registered_emitter: Pubkey,
    pub recipient_token: Pubkey,
    pub payer_token: Pubkey,
    pub custody_token: Pubkey,
    pub mint: Pubkey,
    pub custody_authority: Pubkey,
    pub system_program: Pubkey,
    pub token_program: Pubkey,
}

impl ToAccountMetas for CompleteTransferNative {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(self.vaa, false),
            AccountMeta::new(self.claim, false),
            AccountMeta::new_readonly(self.registered_emitter, false),
            AccountMeta::new(self.recipient_token, false),
            AccountMeta::new(self.payer_token, false),
            AccountMeta::new(self.custody_token, false),
            AccountMeta::new_readonly(self.mint, false),
            AccountMeta::new_readonly(self.custody_authority, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
        ]
    }
}

pub struct CompleteTransferWrapped {
    pub payer: Pubkey,
    pub vaa: Pubkey,
    pub claim: Pubkey,
    pub registered_emitter: Pubkey,
    pub recipient_token: Pubkey,
    pub payer_token: Pubkey,
    pub wrapped_mint: Pubkey,
    pub wrapped_asset: Pubkey,
    pub mint_authority: Pubkey,
    pub system_program: Pubkey,
    pub token_program: Pubkey,
}

impl ToAccountMetas for CompleteTransferWrapped {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(self.vaa, false),
            AccountMeta::new(self.claim, false),
            AccountMeta::new_readonly(self.registered_emitter, false),
            AccountMeta::new(self.recipient_token, false),
            AccountMeta::new(self.payer_token, false),
            AccountMeta::new(self.wrapped_mint, false),
            AccountMeta::new_readonly(self.wrapped_asset, false),
            AccountMeta::new_readonly(self.mint_authority, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
        ]
    }
}

pub struct CompleteTransferWithPayloadNative {
    pub payer: Pubkey,
    pub vaa: Pubkey,
    pub claim: Pubkey,
    pub registered_emitter: Pubkey,
    pub dst_token: Pubkey,
    pub redeemer_authority: Pubkey,
    pub custody_token: Pubkey,
    pub mint: Pubkey,
    pub custody_authority: Pubkey,
    pub system_program: Pubkey,
    pub token_program: Pubkey,
}

impl ToAccountMetas for CompleteTransferWithPayloadNative {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(self.vaa, false),
            AccountMeta::new(self.claim, false),
            AccountMeta::new_readonly(self.registered_emitter, false),
            AccountMeta::new(self.dst_token, false),
            AccountMeta::new_readonly(self.redeemer_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _relayer_fee_token
            AccountMeta::new(self.custody_token, false),
            AccountMeta::new_readonly(self.mint, false),
            AccountMeta::new_readonly(self.custody_authority, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
        ]
    }
}

pub struct CompleteTransferWithPayloadWrapped {
    pub payer: Pubkey,
    pub vaa: Pubkey,
    pub claim: Pubkey,
    pub registered_emitter: Pubkey,
    pub dst_token: Pubkey,
    pub redeemer_authority: Pubkey,
    pub wrapped_mint: Pubkey,
    pub wrapped_asset: Pubkey,
    pub mint_authority: Pubkey,
    pub system_program: Pubkey,
    pub token_program: Pubkey,
}

impl ToAccountMetas for CompleteTransferWithPayloadWrapped {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new_readonly(self.vaa, false),
            AccountMeta::new(self.claim, false),
            AccountMeta::new_readonly(self.registered_emitter, false),
            AccountMeta::new(self.dst_token, false),
            AccountMeta::new_readonly(self.redeemer_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _relayer_fee_token
            AccountMeta::new(self.wrapped_mint, false),
            AccountMeta::new_readonly(self.wrapped_asset, false),
            AccountMeta::new_readonly(self.mint_authority, false),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
        ]
    }
}

pub struct TransferTokensNative {
    /// CHECK: Transaction payer (mut signer).
    pub payer: Pubkey,
    /// CHECK: Source Token Account (mut).
    pub src_token: Pubkey,
    /// CHECK: Mint (read-only).
    pub mint: Pubkey,
    /// CHECK: Transfer Authority (mut, seeds = \[mint.key\], seeds::program =
    /// token_bridge_program).
    pub custody_token: Pubkey,
    /// CHECK: Transfer Authority (read-only, seeds = \["authority_signer"\], seeds::program =
    /// token_bridge_program).
    pub transfer_authority: Pubkey,
    /// CHECK: Custody Authority (read-only, seeds = \["custody_signer"\], seeds::program =
    /// token_bridge_program).
    pub custody_authority: Pubkey,
    /// CHECK: Core Bridge Program Data (read-only, seeds = \["Bridge"\], seeds::program =
    /// core_bridge_program).
    pub core_bridge_config: Pubkey,
    /// CHECK: Core Bridge Message (mut).
    pub core_message: Pubkey,
    /// CHECK: Core Bridge Emitter (read-only, seeds = \["emitter"\], seeds::program =
    /// token_bridge_program).
    pub core_emitter: Pubkey,
    /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\],
    /// seeds::program = core_bridge_program).
    pub core_emitter_sequence: Pubkey,
    /// CHECK: Core Bridge Fee Collector (mut, seeds = \["fee_collector"\], seeds::program =
    /// core_bridge_program).
    pub core_fee_collector: Option<Pubkey>,
    /// CHECK: System Program.
    pub system_program: Pubkey,
    /// CHECK: Token Program.
    pub token_program: Pubkey,
    /// CHECK: Core Bridge Program.
    pub core_bridge_program: Pubkey,
}

impl ToAccountMetas for TransferTokensNative {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new(self.src_token, false),
            AccountMeta::new_readonly(self.mint, false),
            AccountMeta::new(self.custody_token, false),
            AccountMeta::new_readonly(self.transfer_authority, false),
            AccountMeta::new_readonly(self.custody_authority, false),
            AccountMeta::new_readonly(self.core_bridge_config, false),
            AccountMeta::new(self.core_message, true),
            AccountMeta::new_readonly(self.core_emitter, false),
            AccountMeta::new(self.core_emitter_sequence, false),
            AccountMeta::new(self.core_fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
            AccountMeta::new_readonly(self.core_bridge_program, false),
        ]
    }
}

pub struct TransferTokensWrapped {
    /// CHECK: Transaction payer (mut signer).
    pub payer: Pubkey,
    /// CHECK: Source Token Account (mut).
    pub src_token: Pubkey,
    /// CHECK: Wrapped Mint (mut, seeds = \["wrapped", token_chain, token_address\],
    /// seeds::program = token_bridge_program).
    pub wrapped_mint: Pubkey,
    /// CHECK: Wrapped Asset (read-only, seeds = \[wrapped_mint.key\], seeds::program =
    /// token_bridge_program).
    pub wrapped_asset: Pubkey,
    /// CHECK: Transfer Authority (read-only, seeds = \["authority_signer"\], seeds::program =
    /// token_bridge_program).
    pub transfer_authority: Pubkey,
    /// CHECK: Core Bridge Program Data (read-only, seeds = \["Bridge"\], seeds::program =
    /// core_bridge_program).
    pub core_bridge_config: Pubkey,
    /// CHECK: Core Bridge Message (mut).
    pub core_message: Pubkey,
    /// CHECK: Core Bridge Emitter (read-only, seeds = \["emitter"\], seeds::program =
    /// token_bridge_program).
    pub core_emitter: Pubkey,
    /// CHECK: Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\],
    /// seeds::program = core_bridge_program).
    pub core_emitter_sequence: Pubkey,
    /// CHECK: Core Bridge Fee Collector (mut, seeds = \["fee_collector"\], seeds::program =
    /// core_bridge_program).
    pub core_fee_collector: Option<Pubkey>,
    /// CHECK: System Program.
    pub system_program: Pubkey,
    /// CHECK: Token Program.
    pub token_program: Pubkey,
    /// CHECK: Core Bridge Program.
    pub core_bridge_program: Pubkey,
}

impl ToAccountMetas for TransferTokensWrapped {
    fn to_account_metas(
        &self,
        _is_signer: Option<bool>,
    ) -> Vec<solana_program::instruction::AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new(self.src_token, false),
            AccountMeta::new_readonly(crate::ID, false), // _src_owner
            AccountMeta::new(self.wrapped_mint, false),
            AccountMeta::new_readonly(self.wrapped_asset, false),
            AccountMeta::new_readonly(self.transfer_authority, false),
            AccountMeta::new_readonly(self.core_bridge_config, false),
            AccountMeta::new(self.core_message, true),
            AccountMeta::new_readonly(self.core_emitter, false),
            AccountMeta::new(self.core_emitter_sequence, false),
            AccountMeta::new(self.core_fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
            AccountMeta::new_readonly(self.core_bridge_program, false),
        ]
    }
}

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
    pub token_program: Pubkey,
    pub core_bridge_program: Pubkey,
}

impl ToAccountMetas for TransferTokensWithPayloadNative {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new(self.src_token, false),
            AccountMeta::new_readonly(self.mint, false),
            AccountMeta::new(self.custody_token, false),
            AccountMeta::new_readonly(self.transfer_authority, false),
            AccountMeta::new_readonly(self.custody_authority, false),
            AccountMeta::new_readonly(self.core_bridge_config, false),
            AccountMeta::new(self.core_message, true),
            AccountMeta::new_readonly(self.core_emitter, false),
            AccountMeta::new(self.core_emitter_sequence, false),
            AccountMeta::new(self.core_fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(self.sender_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
            AccountMeta::new_readonly(self.core_bridge_program, false),
        ]
    }
}

pub struct TransferTokensWithPayloadWrapped {
    pub payer: Pubkey,
    pub src_token: Pubkey,
    pub wrapped_mint: Pubkey,
    pub wrapped_asset: Pubkey,
    pub transfer_authority: Pubkey,
    pub core_bridge_config: Pubkey,
    pub core_message: Pubkey,
    pub core_emitter: Pubkey,
    pub core_emitter_sequence: Pubkey,
    pub core_fee_collector: Option<Pubkey>,
    pub sender_authority: Pubkey,
    pub system_program: Pubkey,
    pub token_program: Pubkey,
    pub core_bridge_program: Pubkey,
}

impl ToAccountMetas for TransferTokensWithPayloadWrapped {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.payer, true),
            AccountMeta::new_readonly(crate::ID, false), // _config
            AccountMeta::new(self.src_token, false),
            AccountMeta::new_readonly(crate::ID, false), // _src_owner
            AccountMeta::new(self.wrapped_mint, false),
            AccountMeta::new_readonly(self.wrapped_asset, false),
            AccountMeta::new_readonly(self.transfer_authority, false),
            AccountMeta::new_readonly(self.core_bridge_config, false),
            AccountMeta::new(self.core_message, true),
            AccountMeta::new_readonly(self.core_emitter, false),
            AccountMeta::new(self.core_emitter_sequence, false),
            AccountMeta::new(self.core_fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(self.sender_authority, true),
            AccountMeta::new_readonly(crate::ID, false), // _rent
            AccountMeta::new_readonly(self.system_program, false),
            AccountMeta::new_readonly(self.token_program, false),
            AccountMeta::new_readonly(self.core_bridge_program, false),
        ]
    }
}
