use std::io;

use crate::types::NormalizedAmount;
use core_bridge_program::{
    types::{ChainId, ExternalAddress},
    WormDecode, WormEncode,
};

use super::TokenBridgeMessage;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TokenTransfer {
    /// Amount being transferred (big-endian uint256)
    pub normalized_amount: NormalizedAmount,
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: ExternalAddress,
    /// Chain ID of the token
    pub token_chain: ChainId,
    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub recipient: ExternalAddress,
    /// Chain ID of the recipient
    pub recipient_chain: ChainId,
    /// Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
    pub normalized_relayer_fee: NormalizedAmount,
}

impl TokenBridgeMessage for TokenTransfer {}

impl WormDecode for TokenTransfer {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let normalized_amount = NormalizedAmount::decode_reader(reader)?;
        let token_address = ExternalAddress::decode_reader(reader)?;
        let token_chain = ChainId::decode_reader(reader)?;
        let recipient = ExternalAddress::decode_reader(reader)?;
        let recipient_chain = ChainId::decode_reader(reader)?;
        let normalized_relayer_fee = NormalizedAmount::decode_reader(reader)?;

        Ok(Self {
            normalized_amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            normalized_relayer_fee,
        })
    }
}

impl WormEncode for TokenTransfer {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.normalized_amount.encode(writer)?;
        self.token_address.encode(writer)?;
        self.token_chain.encode(writer)?;
        self.recipient.encode(writer)?;
        self.recipient_chain.encode(writer)?;
        self.normalized_relayer_fee.encode(writer)
    }
}
