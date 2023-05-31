use std::io;

use crate::types::NormalizedAmount;
use core_bridge_program::{
    types::{ChainId, ExternalAddress},
    WormDecode, WormEncode,
};

use super::TokenBridgeMessage;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TokenTransferWithPayload {
    /// Amount being transferred (big-endian uint256)
    pub normalized_amount: NormalizedAmount,
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: ExternalAddress,
    /// Chain ID of the token
    pub token_chain: ChainId,
    /// Address of the redeemer. Left-zero-padded if shorter than 32 bytes
    pub redeemer: ExternalAddress,
    /// Chain ID of the redeemer
    pub redeemer_chain: ChainId,
    /// Sender of the transaction
    pub sender_address: ExternalAddress,
    /// Arbitrary payload
    pub payload: Vec<u8>,
}

impl TokenBridgeMessage for TokenTransferWithPayload {}

impl WormDecode for TokenTransferWithPayload {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let normalized_amount = NormalizedAmount::decode_reader(reader)?;
        let token_address = ExternalAddress::decode_reader(reader)?;
        let token_chain = ChainId::decode_reader(reader)?;
        let redeemer = ExternalAddress::decode_reader(reader)?;
        let redeemer_chain = ChainId::decode_reader(reader)?;
        let sender_address = ExternalAddress::decode_reader(reader)?;
        let mut payload = Vec::new();
        reader.read_to_end(&mut payload)?;

        Ok(Self {
            normalized_amount,
            token_address,
            token_chain,
            redeemer,
            redeemer_chain,
            sender_address,
            payload,
        })
    }
}

impl WormEncode for TokenTransferWithPayload {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.normalized_amount.encode(writer)?;
        self.token_address.encode(writer)?;
        self.token_chain.encode(writer)?;
        self.redeemer.encode(writer)?;
        self.redeemer_chain.encode(writer)?;
        self.sender_address.encode(writer)?;
        writer.write_all(&self.payload)
    }
}
