use crate::{
    types::{Address, ChainID},
    TokenBridgeError,
};
use borsh::{BorshDeserialize, BorshSerialize};
use bridge::vaa::{DeserializePayload, SerializePayload};
use byteorder::{BigEndian, ReadBytesExt, WriteBytesExt};
use primitive_types::U256;
use solana_program::{native_token::Sol, program_error::ProgramError};
use solitaire::SolitaireError;
use std::{
    error::Error,
    io::{Cursor, Read, Write},
    str::Utf8Error,
    string::FromUtf8Error,
};

pub struct PayloadTransfer {
    // Amount being transferred (big-endian uint256)
    pub amount: U256,
    // Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    // Chain ID of the token
    pub token_chain: ChainID,
    // Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: Address,
    // Chain ID of the recipient
    pub to_chain: ChainID,
    // Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
    pub fee: U256,
}

impl DeserializePayload for PayloadTransfer {
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut v = Cursor::new(buf);

        if v.read_u8()? != 1 {
            return Err(SolitaireError::Custom(0));
        };

        let mut am_data: [u8; 32] = [0; 32];
        v.read_exact(&mut am_data)?;
        let amount = U256::from_big_endian(&am_data);

        let mut token_address = Address::default();
        v.read_exact(&mut token_address)?;

        let token_chain = v.read_u16::<BigEndian>()?;

        let mut to = Address::default();
        v.read_exact(&mut to)?;

        let to_chain = v.read_u16::<BigEndian>()?;

        let mut fee_data: [u8; 32] = [0; 32];
        v.read_exact(&mut fee_data)?;
        let fee = U256::from_big_endian(&fee_data);

        Ok(PayloadTransfer {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            fee,
        })
    }
}

impl SerializePayload for PayloadTransfer {
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        // Payload ID
        writer.write_u8(1)?;

        let mut am_data: [u8; 32] = [0; 32];
        self.amount.to_big_endian(&mut am_data);
        writer.write(&am_data)?;

        writer.write(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;
        writer.write(&self.to)?;
        writer.write_u16::<BigEndian>(self.to_chain)?;

        let mut fee_data: [u8; 32] = [0; 32];
        self.fee.to_big_endian(&mut fee_data);
        writer.write(&fee_data)?;

        Ok(())
    }
}

pub struct PayloadAssetMeta {
    // Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    // Chain ID of the token
    pub token_chain: ChainID,
    // Number of decimals of the token
    pub decimals: u8,
    // Symbol of the token
    pub symbol: String,
    // Name of the token
    pub name: String,
}

impl DeserializePayload for PayloadAssetMeta {
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut v = Cursor::new(buf);

        if v.read_u8()? != 2 {
            return Err(SolitaireError::Custom(0));
        };

        let mut token_address = Address::default();
        v.read_exact(&mut token_address)?;

        let token_chain = v.read_u16::<BigEndian>()?;
        let decimals = v.read_u8()?;

        let mut symbol_data: [u8; 32] = [0; 32];
        v.read_exact(&mut symbol_data)?;
        let symbol = String::from_utf8(symbol_data.to_vec())
            .map_err::<SolitaireError, _>(|_| TokenBridgeError::InvalidUTF8String.into())?;

        let mut name_data: [u8; 32] = [0; 32];
        v.read_exact(&mut name_data)?;
        let name = String::from_utf8(name_data.to_vec())
            .map_err::<SolitaireError, _>(|_| TokenBridgeError::InvalidUTF8String.into())?;

        Ok(PayloadAssetMeta {
            token_address,
            token_chain,
            decimals,
            symbol,
            name,
        })
    }
}

impl SerializePayload for PayloadAssetMeta {
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        // Payload ID
        writer.write_u8(2)?;

        writer.write(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;

        writer.write_u8(self.decimals)?;

        let mut symbol: [u8; 32] = [0; 32];
        for i in 0..self.symbol.len() {
            symbol[(32 - self.symbol.len()) + i] = self.symbol.as_bytes()[i];
        }
        writer.write(&symbol);

        let mut name: [u8; 32] = [0; 32];
        for i in 0..self.name.len() {
            name[(32 - self.name.len()) + i] = self.name.as_bytes()[i];
        }
        writer.write(&name);

        Ok(())
    }
}

pub struct PayloadGovernanceRegisterChain {
    // Chain ID of the chain to be registered
    pub chain: ChainID,
    // Address of the endpoint on the chain
    pub endpoint_address: Address,
}

impl DeserializePayload for PayloadGovernanceRegisterChain {
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut v = Cursor::new(buf);

        if v.read_u8()? != 2 {
            return Err(SolitaireError::Custom(0));
        };
        Ok(PayloadGovernanceRegisterChain {
            chain: 0,
            endpoint_address: [0u8; 32],
        })
    }
}

impl SerializePayload for PayloadGovernanceRegisterChain {
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        // Payload ID
        writer.write_u8(2)?;

        Ok(())
    }
}
