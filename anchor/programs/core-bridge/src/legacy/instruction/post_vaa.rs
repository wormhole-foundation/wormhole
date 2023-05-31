use crate::types::{ChainId, ExternalAddress, Finality, Timestamp, VaaVersion};
use anchor_lang::prelude::{borsh, AnchorDeserialize, AnchorSerialize};

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct LegacyPostVaaArgs {
    /// The VAA version passed into this instruction does not do anything functional because someone
    /// can just pass in the value '1' and it will be accepted.
    pub _version: VaaVersion,
    /// The guardian set index passed into this instruction does not do anything functional because
    /// this guardian set index is checked in Anchor's account context.
    pub _guardian_set_index: u32,
    pub timestamp: Timestamp,
    pub nonce: u32,
    pub emitter_chain: ChainId,
    pub emitter_address: ExternalAddress,
    pub sequence: u64,
    pub finality: Finality,
    pub payload: Vec<u8>,
}
