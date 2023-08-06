use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

/// Arguments to post new VAA data after signature verification.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct PostVaaArgs {
    /// Unused data.
    pub _gap_0: [u8; 5],
    /// Time the message was submitted.
    pub timestamp: u32,
    /// Unique ID for this message.
    pub nonce: u32,
    /// The Wormhole chain ID denoting the origin of this message.
    pub emitter_chain: u16,
    /// Emitter of the message.
    pub emitter_address: [u8; 32],
    /// Sequence number of this message.
    pub sequence: u64,
    /// Level of consistency requested by the emitter.
    pub consistency_level: u8,
    /// Message payload.
    pub payload: Vec<u8>,
}
