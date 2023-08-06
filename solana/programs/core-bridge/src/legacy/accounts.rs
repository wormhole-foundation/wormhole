//! A set of structs mirroring the structs deriving Accounts, where each field is a Pubkey. This is
//! useful for specifying accounts for a client.

use anchor_lang::prelude::Pubkey;

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
    /// Core Bridge Fee Collector (optional, mut, seeds = \["fee_collector"\]).
    pub fee_collector: Option<Pubkey>,
    /// System Program.
    pub system_program: Pubkey,
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
    /// Core Bridge Fee Collector (optional, mut, seeds = \["fee_collector"\]).
    pub fee_collector: Option<Pubkey>,
    /// System Program.
    pub system_program: Pubkey,
}
