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
    pubkey::{
        Pubkey,
        PUBKEY_BYTES,
    },
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

#[derive(PartialEq, Debug, Clone)]
pub struct PayloadTransfer {
    /// Amount being transferred (big-endian uint256)
    pub amount: U256,
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    /// Chain ID of the token
    pub token_chain: ChainID,
    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: Address,
    /// Chain ID of the recipient
    pub to_chain: ChainID,
    /// Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
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
        writer.write_all(&am_data)?;

        writer.write_all(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;
        writer.write_all(&self.to)?;
        writer.write_u16::<BigEndian>(self.to_chain)?;

        let mut fee_data: [u8; 32] = [0; 32];
        self.fee.to_big_endian(&mut fee_data);
        writer.write_all(&fee_data)?;

        Ok(())
    }
}

impl DeserializePayload for PayloadTransferWithPayload {
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut v = Cursor::new(buf);

        if v.read_u8()? != 3 {
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

        let mut from_address = Address::default();
        v.read_exact(&mut from_address)?;

        let mut payload = vec![];
        v.read_to_end(&mut payload)?;

        Ok(PayloadTransferWithPayload {
            amount,
            token_address,
            token_chain,
            to,
            to_chain,
            from_address,
            payload,
        })
    }
}

impl SerializePayload for PayloadTransferWithPayload {
    fn serialize<W: Write>(&self, writer: &mut W) -> Result<(), SolitaireError> {
        // Payload ID
        writer.write_u8(3)?;

        let mut am_data: [u8; 32] = [0; 32];
        self.amount.to_big_endian(&mut am_data);
        writer.write_all(&am_data)?;

        writer.write_all(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;
        writer.write_all(&self.to)?;
        writer.write_u16::<BigEndian>(self.to_chain)?;

        writer.write_all(&self.from_address)?;

        writer.write_all(self.payload.as_slice())?;

        Ok(())
    }
}

#[derive(PartialEq, Debug, Clone)]
pub struct PayloadTransferWithPayload {
    /// Amount being transferred (big-endian uint256)
    pub amount: U256,
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    /// Chain ID of the token
    pub token_chain: ChainID,
    /// Address of the recipient. Left-zero-padded if shorter than 32 bytes
    pub to: Address,
    /// Chain ID of the recipient
    pub to_chain: ChainID,
    /// Sender of the transaction
    pub from_address: Address,
    /// Arbitrary payload
    pub payload: Vec<u8>,
}

#[derive(PartialEq, Debug)]
pub struct PayloadAssetMeta {
    /// Address of the token. Left-zero-padded if shorter than 32 bytes
    pub token_address: Address,
    /// Chain ID of the token
    pub token_chain: ChainID,
    /// Number of decimals of the token
    pub decimals: u8,
    /// Symbol of the token
    pub symbol: String,
    /// Name of the token
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

        writer.write_all(&self.token_address)?;
        writer.write_u16::<BigEndian>(self.token_chain)?;

        writer.write_u8(self.decimals)?;

        let mut symbol: [u8; 32] = [0; 32];
        let count = cmp::min(symbol.len(), self.symbol.len());
        symbol[..count].copy_from_slice(self.symbol[..count].as_bytes());

        writer.write_all(&symbol)?;

        let mut name: [u8; 32] = [0; 32];
        let count = cmp::min(name.len(), self.name.len());
        name[..count].copy_from_slice(self.name[..count].as_bytes());

        writer.write_all(&name)?;

        Ok(())
    }
}

#[derive(PartialEq, Debug)]
pub struct PayloadGovernanceRegisterChain {
    /// Chain ID of the chain to be registered
    pub chain: ChainID,
    /// Address of the endpoint on the chain
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
        writer.write_all(&self.endpoint_address[..])?;

        Ok(())
    }
}

#[derive(PartialEq, Debug)]
pub struct GovernancePayloadUpgrade {
    /// Address of the new Implementation
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
    const MODULE: &'static str = "TokenBridge";
    const ACTION: u8 = 2;
}

impl DeserializeGovernancePayload for GovernancePayloadUpgrade {
}

/// Body of the `SetPauserAddresses` (action 4) governance message — see the "Pausing" section of
/// whitepapers/0003_token_bridge.md.
///
/// Wire format after the governance header (three length-prefixed addresses, in order):
///   `pauser_len(u8) | pauser[..] | freezer_len(u8) | freezer[..] | unpauser_len(u8) | unpauser[..]`
///
/// On Solana each length must be either 32 (the native address size) or 0 (the role is left
/// unassigned). Any other length is rejected. An all-zero 32-byte address is treated as equivalent
/// to a zero-length encoding — both decode to `Pubkey::default()`, which `api::pause` rejects via
/// the `PauserNotConfigured` check before comparing the caller.
#[derive(PartialEq, Debug)]
pub struct PayloadSetPauserAddresses {
    pub pauser: Pubkey,
    pub freezer: Pubkey,
    pub unpauser: Pubkey,
}

impl SerializePayload for PayloadSetPauserAddresses {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
        self.write_governance_header(v)?;
        // Canonical SVM encoding: always emit the full 32-byte length, even for an unassigned
        // role (in which case the body is 32 bytes of zeros). The wire format also accepts a
        // zero-length encoding on the receive side; either form decodes to `Pubkey::default()`.
        // `PUBKEY_BYTES` is 32 and `as u8` is safe for that range.
        v.write_u8(PUBKEY_BYTES as u8)?;
        v.write_all(&self.pauser.to_bytes())?;
        v.write_u8(PUBKEY_BYTES as u8)?;
        v.write_all(&self.freezer.to_bytes())?;
        v.write_u8(PUBKEY_BYTES as u8)?;
        v.write_all(&self.unpauser.to_bytes())?;
        Ok(())
    }
}

impl DeserializePayload for PayloadSetPauserAddresses
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);
        Self::check_governance_header(&mut c)?;

        let pauser = read_length_prefixed_pubkey(&mut c)?;
        let freezer = read_length_prefixed_pubkey(&mut c)?;
        let unpauser = read_length_prefixed_pubkey(&mut c)?;

        if c.position() != c.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(PayloadSetPauserAddresses {
            pauser,
            freezer,
            unpauser,
        })
    }
}

impl SerializeGovernancePayload for PayloadSetPauserAddresses {
    const MODULE: &'static str = "TokenBridge";
    const ACTION: u8 = 4;
}

impl DeserializeGovernancePayload for PayloadSetPauserAddresses {
}

/// Decode a single length-prefixed address from a `SetPauserAddresses` payload. Accepts either
/// length 0 (role unassigned) or length 32 (Solana native address size); any other length is
/// rejected. Both `len == 0` and `len == 32` with all-zero bytes decode to `Pubkey::default()`.
fn read_length_prefixed_pubkey(
    c: &mut Cursor<&mut &[u8]>,
) -> std::result::Result<Pubkey, SolitaireError> {
    let len = c.read_u8()?;
    if len == 0 {
        return Ok(Pubkey::default());
    }
    if usize::from(len) != PUBKEY_BYTES {
        return Err(InvalidAccountData.into());
    }
    let mut bytes = [0u8; PUBKEY_BYTES];
    c.read_exact(&mut bytes)?;
    Ok(Pubkey::new(&bytes[..]))
}

#[cfg(test)]
#[allow(unused_imports)]
mod tests {
    use crate::messages::{
        GovernancePayloadUpgrade,
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadSetPauserAddresses,
        PayloadTransfer,
        PayloadTransferWithPayload,
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

        let data = transfer_original.try_to_vec().unwrap();
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

        let data = am_original.try_to_vec().unwrap();
        let am_deser = PayloadAssetMeta::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(am_original, am_deser);
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

    #[test]
    pub fn test_serde_gov_set_pauser_addresses() {
        let original = PayloadSetPauserAddresses {
            pauser: Pubkey::new_unique(),
            freezer: Pubkey::new_unique(),
            unpauser: Pubkey::new_unique(),
        };

        let data = original.try_to_vec().unwrap();
        let deser = PayloadSetPauserAddresses::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(original, deser);
    }

    #[test]
    pub fn test_serde_transfer_with_payload() {
        let mut token_address = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut token_address);
        let mut from_address = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut from_address);
        let mut to = [0u8; 32];
        rand::thread_rng().fill_bytes(&mut to);
        let payload = vec![0u8; 10];

        let transfer_original = PayloadTransferWithPayload {
            amount: U256::from(1003),
            token_address,
            token_chain: 8,
            to,
            to_chain: 1,
            from_address,
            payload,
        };

        let data = transfer_original.try_to_vec().unwrap();
        let transfer_deser = PayloadTransferWithPayload::deserialize(&mut data.as_slice()).unwrap();

        assert_eq!(transfer_original, transfer_deser);
    }
}
