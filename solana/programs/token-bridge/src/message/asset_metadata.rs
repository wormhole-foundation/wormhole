use std::io;

use crate::types::FixedString;
use core_bridge_program::{
    types::{ChainId, ExternalAddress},
    WormDecode, WormEncode,
};

use super::TokenBridgeMessage;

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AssetMetadata {
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: ExternalAddress,
    /// Chain ID of the token
    pub token_chain: ChainId,
    /// Number of decimals of the token
    pub decimals: u8,
    /// Symbol of the token
    pub symbol: FixedString<32>,
    /// Name of the token
    pub name: FixedString<32>,
}

impl TokenBridgeMessage for AssetMetadata {}

impl WormDecode for AssetMetadata {
    fn decode_reader<R: io::Read>(reader: &mut R) -> io::Result<Self> {
        let token_address = ExternalAddress::decode_reader(reader)?;
        let token_chain = ChainId::decode_reader(reader)?;
        let decimals = u8::decode_reader(reader)?;
        let symbol = FixedString::decode_reader(reader)?;
        let name = FixedString::decode_reader(reader)?;

        Ok(Self {
            token_address,
            token_chain,
            decimals,
            symbol,
            name,
        })
    }
}

impl WormEncode for AssetMetadata {
    fn encode<W: io::Write>(&self, writer: &mut W) -> io::Result<()> {
        self.token_address.encode(writer)?;
        self.token_chain.encode(writer)?;
        self.decimals.encode(writer)?;
        self.symbol.encode(writer)?;
        self.name.encode(writer)
    }
}
