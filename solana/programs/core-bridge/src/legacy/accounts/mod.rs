//! A set of structs mirroring the structs deriving [Accounts](anchor_lang::prelude::Accounts),
//! where each field is a [Pubkey]. This is useful for specifying self for a client.
//!
//! NOTE: This is similar to how [accounts](mod@crate::accounts) is generated via Anchor's
//! [program][anchor_lang::prelude::program] macro.

use anchor_lang::{prelude::Pubkey, ToAccountMetas};
use solana_program::instruction::AccountMeta;

/// Context to post a new Core Bridge message.
pub struct PostMessage {
    /// Core Bridge Program Data (mut, seeds = \["Bridge"\]).
    pub config: Pubkey,
    /// Core Bridge Message (mut).
    pub message: Pubkey,
    /// Core Bridge Emitter (optional, read-only signer).
    pub emitter: Option<Pubkey>,
    /// Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
    pub emitter_sequence: Pubkey,
    /// Transaction payer (mut signer).
    pub payer: Pubkey,
    /// Core Bridge Fee Collector (optional, read-only, seeds = \["fee_collector"\]).
    pub fee_collector: Option<Pubkey>,
    /// System Program.
    pub system_program: Pubkey,
}

impl ToAccountMetas for PostMessage {
    fn to_account_metas(&self, message_is_signer: Option<bool>) -> Vec<AccountMeta> {
        // Using `message_is_signer` above is a hack. But because we do not want to return a result,
        // we assume that the caller of this is passing in whether a message is a signer, which is
        // the case when a new message account is created when someone invokes this instruction.
        // Otherwise the message was already prepared so it does not have to be a signer.

        // If the emitter is None, we do not require it to be a signer.
        let (emitter, emitter_is_signer) = match self.emitter {
            Some(emitter) => (emitter, true),
            None => (crate::ID, false),
        };

        vec![
            AccountMeta::new(self.config, false),
            AccountMeta::new(self.message, message_is_signer.unwrap_or(true)),
            AccountMeta::new_readonly(emitter, emitter_is_signer),
            AccountMeta::new(self.emitter_sequence, false),
            AccountMeta::new(self.payer, true),
            AccountMeta::new(self.fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(self.system_program, false),
        ]
    }
}

/// Context to post a new or reuse an existing Core Bridge message.
pub struct PostMessageUnreliable {
    /// Core Bridge Program Data (mut, seeds = \["Bridge"\]).
    pub config: Pubkey,
    /// Core Bridge Message (mut).
    pub message: Pubkey,
    /// Core Bridge Emitter (read-only signer).
    pub emitter: Pubkey,
    /// Core Bridge Emitter Sequence (mut, seeds = \["Sequence", emitter.key\]).
    pub emitter_sequence: Pubkey,
    /// Transaction payer (mut signer).
    pub payer: Pubkey,
    /// Core Bridge Fee Collector (optional, read-only, seeds = \["fee_collector"\]).
    pub fee_collector: Option<Pubkey>,
    /// System Program.
    pub system_program: Pubkey,
}

impl ToAccountMetas for PostMessageUnreliable {
    fn to_account_metas(&self, _is_signer: Option<bool>) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.config, false),
            AccountMeta::new(self.message, true),
            AccountMeta::new_readonly(self.emitter, true),
            AccountMeta::new(self.emitter_sequence, false),
            AccountMeta::new(self.payer, true),
            AccountMeta::new(self.fee_collector.unwrap_or(crate::ID), false),
            AccountMeta::new_readonly(crate::ID, false), // _clock
            AccountMeta::new_readonly(self.system_program, false),
        ]
    }
}
