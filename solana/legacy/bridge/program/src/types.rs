use crate::{
    api::ForeignAddress,
    vaa::{
        DeserializeGovernancePayload,
        DeserializePayload,
        SerializeGovernancePayload,
        SerializePayload,
    },
};
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use byteorder::{
    BigEndian,
    ReadBytesExt,
};
use primitive_types::U256;
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::{
    program_error::ProgramError::InvalidAccountData,
    pubkey::Pubkey,
};
use solitaire::SolitaireError;
use std::{
    self,
    io::{
        Cursor,
        Read,
        Write,
    },
};

/// Type representing an Ethereum style public key for Guardians.
pub type GuardianPublicKey = [u8; 20];

#[repr(u8)]
#[derive(BorshSerialize, BorshDeserialize, Clone, Serialize, Deserialize)]
pub enum ConsistencyLevel {
    Confirmed,
    Finalized,
}

pub struct GovernancePayloadUpgrade {
    // Address of the new Implementation
    pub new_contract: Pubkey,
}

impl SerializePayload for GovernancePayloadUpgrade {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
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
    const MODULE: &'static str = "Core";
    const ACTION: u8 = 1;
}

impl DeserializeGovernancePayload for GovernancePayloadUpgrade {
}

pub struct GovernancePayloadGuardianSetChange {
    // New GuardianSetIndex
    pub new_guardian_set_index: u32,

    // New GuardianSet
    pub new_guardian_set: Vec<[u8; 20]>,
}

impl SerializePayload for GovernancePayloadGuardianSetChange {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
        use byteorder::WriteBytesExt;
        v.write_u32::<BigEndian>(self.new_guardian_set_index)?;
        v.write_u8(self.new_guardian_set.len() as u8)?;
        for key in self.new_guardian_set.iter() {
            v.write_all(key)?;
        }
        Ok(())
    }
}

impl DeserializePayload for GovernancePayloadGuardianSetChange
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);
        Self::check_governance_header(&mut c)?;

        let new_index = c.read_u32::<BigEndian>()?;

        let keys_len = c.read_u8()?;
        let mut keys = Vec::with_capacity(keys_len as usize);
        for _ in 0..keys_len {
            let mut key: [u8; 20] = [0; 20];
            c.read_exact(&mut key)?;
            keys.push(key);
        }

        if c.position() != c.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(GovernancePayloadGuardianSetChange {
            new_guardian_set_index: new_index,
            new_guardian_set: keys,
        })
    }
}

impl SerializeGovernancePayload for GovernancePayloadGuardianSetChange {
    const MODULE: &'static str = "Core";
    const ACTION: u8 = 2;
}

impl DeserializeGovernancePayload for GovernancePayloadGuardianSetChange {
}

pub struct GovernancePayloadSetMessageFee {
    // New fee in lamports
    pub fee: U256,
}

impl SerializePayload for GovernancePayloadSetMessageFee {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
        let mut fee_data = [0u8; 32];
        self.fee.to_big_endian(&mut fee_data);
        v.write_all(&fee_data[..])?;

        Ok(())
    }
}

impl DeserializePayload for GovernancePayloadSetMessageFee
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);
        Self::check_governance_header(&mut c)?;

        let mut fee_data: [u8; 32] = [0; 32];
        c.read_exact(&mut fee_data)?;
        let fee = U256::from_big_endian(&fee_data);

        if c.position() != c.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(GovernancePayloadSetMessageFee { fee })
    }
}

impl SerializeGovernancePayload for GovernancePayloadSetMessageFee {
    const MODULE: &'static str = "Core";
    const ACTION: u8 = 3;
}

impl DeserializeGovernancePayload for GovernancePayloadSetMessageFee {
}

pub struct GovernancePayloadTransferFees {
    // Amount to be transferred
    pub amount: U256,

    // Recipient
    pub to: ForeignAddress,
}

impl SerializePayload for GovernancePayloadTransferFees {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
        let mut amount_data = [0u8; 32];
        self.amount.to_big_endian(&mut amount_data);
        v.write_all(&amount_data)?;
        v.write_all(&self.to)?;
        Ok(())
    }
}

impl DeserializePayload for GovernancePayloadTransferFees
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);
        Self::check_governance_header(&mut c)?;

        let mut amount_data: [u8; 32] = [0; 32];
        c.read_exact(&mut amount_data)?;
        let amount = U256::from_big_endian(&amount_data);

        let mut to = ForeignAddress::default();
        c.read_exact(&mut to)?;

        if c.position() != c.into_inner().len() as u64 {
            return Err(InvalidAccountData.into());
        }

        Ok(GovernancePayloadTransferFees { amount, to })
    }
}

impl SerializeGovernancePayload for GovernancePayloadTransferFees {
    const MODULE: &'static str = "Core";
    const ACTION: u8 = 4;
}

impl DeserializeGovernancePayload for GovernancePayloadTransferFees {
}
