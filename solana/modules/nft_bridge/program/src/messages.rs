use crate::types::{
    Address,
    ChainID,
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
    program_error::ProgramError::InvalidAccountData,
    pubkey::Pubkey,
};
use solitaire::SolitaireError;
use std::{
    cmp,
    io::{
        Cursor,
        Read,
        Write,
    },
};

pub const MODULE: &str = "NFTBridge";

#[derive(PartialEq, Debug, Clone)]
pub struct PayloadTransfer {
    // Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    // Chain ID of the token
    pub token_chain: ChainID,
    // Symbol of the token
    pub symbol: String,
    // Name of the token
    pub name: String,
    // TokenID of the token (big-endian uint256)
    pub token_id: U256,
    // URI of the token metadata
    pub uri: String,
    // Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: Address,
    // Chain ID of the recipient
    pub to_chain: ChainID,
}

impl DeserializePayload for PayloadTransfer {
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        use bstr::ByteSlice;
        let mut v = Cursor::new(buf);

        if v.read_u8()? != 1 {
            return Err(SolitaireError::Custom(0));
        };

        let mut token_address = Address::default();
        v.read_exact(&mut token_address)?;

        let token_chain = v.read_u16::<BigEndian>()?;

        // We may receive invalid UTF-8 over the bridge, especially if truncated. To compensate for
        // this we rely on the bstr libraries ability to parse invalid UTF-8, and strip out the
        // "Invalid Unicode Codepoint" (FFFD) characters. This becomes the canonical representation
        // on Solana.
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

        let mut id_data: [u8; 32] = [0; 32];
        v.read_exact(&mut id_data)?;
        let token_id = U256::from_big_endian(&id_data);

        let uri_len = v.read_u8()?;
        let mut uri_bytes = vec![0u8; uri_len as usize];
        v.read_exact(uri_bytes.as_mut_slice())?;
        let uri = String::from_utf8(uri_bytes).unwrap();

        let mut to = Address::default();
        v.read_exact(&mut to)?;

        let to_chain = v.read_u16::<BigEndian>()?;

        if v.position() != v.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(PayloadTransfer {
            token_address,
            token_chain,
            to,
            to_chain,
            symbol,
            name,
            token_id,
            uri,
        })
    }
}

impl SerializePayload for PayloadTransfer {
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        // Payload ID
        writer.write_u8(1)?;

        writer.write_all(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;

        let mut symbol: [u8; 32] = [0; 32];
        let count = cmp::min(symbol.len(), self.symbol.len());
        symbol[..count].copy_from_slice(self.symbol[..count].as_bytes());

        writer.write_all(&symbol)?;

        let mut name: [u8; 32] = [0; 32];
        let count = cmp::min(name.len(), self.name.len());
        name[..count].copy_from_slice(self.name[..count].as_bytes());
        writer.write_all(&name)?;

        let mut id_data: [u8; 32] = [0; 32];
        self.token_id.to_big_endian(&mut id_data);
        writer.write_all(&id_data)?;

        writer.write_u8(self.uri.len() as u8)?;
        writer.write_all(self.uri.as_bytes())?;

        writer.write_all(&self.to)?;
        writer.write_u16::<BigEndian>(self.to_chain)?;

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
    const MODULE: &'static str = MODULE;
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
        writer.write_all(&self.endpoint_address[..])?;

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
        v.write_all(&self.new_contract.to_bytes())?;
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
    const MODULE: &'static str = MODULE;
    const ACTION: u8 = 2;
}

impl DeserializeGovernancePayload for GovernancePayloadUpgrade {
}

#[cfg(test)]
#[allow(unused_imports)]
mod tests {
    use crate::messages::{
        GovernancePayloadUpgrade,
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
            token_address,
            token_chain: 8,
            to,
            to_chain: 1,
            name: String::from("Token Token"),
            symbol: String::from("TEST"),
            uri: String::from("https://abc.abc.abc.com"),
            token_id: U256::from(1234),
        };

        let data = transfer_original.try_to_vec().unwrap();
        let transfer_deser = PayloadTransfer::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(transfer_original, transfer_deser);
    }

    #[test]
    pub fn test_serde_gov_upgrade() {
        let original = GovernancePayloadUpgrade {
            new_contract: Pubkey::new_unique(),
        };

        let data = original.try_to_vec().unwrap();
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

        let data = original.try_to_vec().unwrap();
        let deser = PayloadGovernanceRegisterChain::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(original, deser);
    }
}
