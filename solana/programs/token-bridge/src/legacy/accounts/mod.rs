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
    pub system_program: Pubkey,
    pub token_program: Pubkey,
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
    pub system_program: Pubkey,
    pub token_program: Pubkey,
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
