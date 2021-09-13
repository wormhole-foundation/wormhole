use crate::{
    types::{
        Address,
        ChainID,
    },
    TokenBridgeError,
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use bridge::{
    vaa::{
        DeserializePayload,
        SerializePayload,
    },
    DeserializeGovernancePayload,
    SerializeGovernancePayload,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
    WriteBytesExt,
};
use primitive_types::U256;
use solana_program::{
    native_token::Sol,
    program_error::{
        ProgramError,
        ProgramError::InvalidAccountData,
    },
    pubkey::Pubkey,
};
use solitaire::SolitaireError;
use std::{
    error::Error,
    io::{
        Cursor,
        Read,
        Write,
    },
    str::Utf8Error,
    string::FromUtf8Error,
};

#[derive(PartialEq, Debug, Clone)]
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

        if v.position() != v.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

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

#[derive(PartialEq, Debug)]
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
        use bstr::ByteSlice;

        let mut v = Cursor::new(buf);

        if v.read_u8()? != 2 {
            return Err(SolitaireError::Custom(0));
        };

        let mut token_address = Address::default();
        v.read_exact(&mut token_address)?;

        let token_chain = v.read_u16::<BigEndian>()?;
        let decimals = v.read_u8()?;

        let mut symbol_data = vec![0u8; 32];
        v.read_exact(&mut symbol_data)?;
        symbol_data.retain(|&c| c != 0);
        let mut symbol: Vec<char> = symbol_data.chars().collect();
        symbol.retain(|&c| c != '\u{FFFD}');
        let symbol: String = symbol.iter().collect();

        let mut name_data = vec![0u8; 32];
        v.read_exact(&mut name_data)?;
        name_data.retain(|&c| c != 0);
        let mut name: Vec<char> = name_data.chars().collect();
        name.retain(|&c| c != '\u{FFFD}');
        let name: String = name.iter().collect();

        if v.position() != v.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

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
            symbol[i] = self.symbol.as_bytes()[i];
        }
        writer.write(&symbol)?;

        let mut name: [u8; 32] = [0; 32];
        for i in 0..self.name.len() {
            name[i] = self.name.as_bytes()[i];
        }
        writer.write(&name)?;

        Ok(())
    }
}

#[derive(PartialEq, Debug)]
pub struct PayloadGovernanceRegisterChain {
    // Chain ID of the chain to be registered
    pub chain: ChainID,
    // Address of the endpoint on the chain
    pub endpoint_address: Address,
}

impl SerializeGovernancePayload for PayloadGovernanceRegisterChain {
    const MODULE: &'static str = "TokenBridge";
    const ACTION: u8 = 1;
}

impl DeserializeGovernancePayload for PayloadGovernanceRegisterChain {
}

impl DeserializePayload for PayloadGovernanceRegisterChain
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut v = Cursor::new(buf);
        Self::check_governance_header(&mut v)?;

        let chain = v.read_u16::<BigEndian>()?;
        let mut endpoint_address = [0u8; 32];
        v.read_exact(&mut endpoint_address)?;

        if v.position() != v.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(PayloadGovernanceRegisterChain {
            chain,
            endpoint_address,
        })
    }
}

impl SerializePayload for PayloadGovernanceRegisterChain
where
    Self: SerializeGovernancePayload,
{
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        self.write_governance_header(writer)?;
        // Payload ID
        writer.write_u16::<BigEndian>(self.chain)?;
        writer.write(&self.endpoint_address[..])?;

        Ok(())
    }
}

#[derive(PartialEq, Debug)]
pub struct GovernancePayloadUpgrade {
    // Address of the new Implementation
    pub new_contract: Pubkey,
}

impl SerializePayload for GovernancePayloadUpgrade {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
        self.write_governance_header(v)?;
        v.write(&self.new_contract.to_bytes())?;
        Ok(())
    }
}

impl DeserializePayload for GovernancePayloadUpgrade
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);
        Self::check_governance_header(&mut c)?;

        let mut addr = [0u8; 32];
        c.read_exact(&mut addr)?;

        if c.position() != c.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(GovernancePayloadUpgrade {
            new_contract: Pubkey::new(&addr[..]),
        })
    }
}

impl SerializeGovernancePayload for GovernancePayloadUpgrade {
    const MODULE: &'static str = "TokenBridge";
    const ACTION: u8 = 2;
}

impl DeserializeGovernancePayload for GovernancePayloadUpgrade {
}

#[cfg(feature = "no-entrypoint")]
mod tests {
    use crate::messages::{
        GovernancePayloadUpgrade,
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    };
    use bridge::{
        DeserializePayload,
        SerializePayload,
    };
    use primitive_types::U256;
    use rand::RngCore;
    use solana_program::pubkey::Pubkey;

    #[test]
    pub fn test_serde_transfer() {
        let mut token_address = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut token_address);
        let mut to = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut to);

        let transfer_original = PayloadTransfer {
            amount: U256::from(1003),
            token_address,
            token_chain: 8,
            to,
            to_chain: 1,
            fee: U256::from(1139),
        };

        let mut data = transfer_original.try_to_vec().unwrap();
        let transfer_deser = PayloadTransfer::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(transfer_original, transfer_deser);
    }

    #[test]
    pub fn test_serde_asset_meta() {
        let mut token_address = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut token_address);

        let am_original = PayloadAssetMeta {
            token_address,
            token_chain: 9,
            decimals: 13,
            symbol: "ABKK".to_string(),
            name: "ZAC".to_string(),
        };

        let mut data = am_original.try_to_vec().unwrap();
        let am_deser = PayloadAssetMeta::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(am_original, am_deser);
    }

    #[test]
    pub fn test_serde_gov_upgrade() {
        let original = GovernancePayloadUpgrade {
            new_contract: Pubkey::new_unique(),
        };

        let mut data = original.try_to_vec().unwrap();
        let deser = GovernancePayloadUpgrade::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(original, deser);
    }

    #[test]
    pub fn test_serde_gov_register_chain() {
        let mut endpoint_address = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut endpoint_address);

        let original = PayloadGovernanceRegisterChain {
            chain: 8,
            endpoint_address,
        };

        let mut data = original.try_to_vec().unwrap();
        let deser = PayloadGovernanceRegisterChain::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(original, deser);
    }
}
