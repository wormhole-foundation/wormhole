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
use solitaire::{
    processors::seeded::{
        AccountOwner,
        Owned,
    },
    SolitaireError,
};
use std::{
    io::{
        Cursor,
        Read,
        Write,
    },
    ops::{
        Deref,
        DerefMut,
    },
    str::FromStr,
};

#[derive(Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
pub struct GuardianSetData {
    /// Version number of this guardian set.
    pub index: u32,

    /// public key hashes of the guardian set
    pub keys: Vec<[u8; 20]>,

    /// creation time
    pub creation_time: u32,

    /// expiration time when VAAs issued by this set are no longer valid
    pub expiration_time: u32,
}

impl GuardianSetData {
    /// Number of guardians in the set
    pub fn num_guardians(&self) -> u8 {
        self.keys.iter().filter(|v| **v != [0u8; 20]).count() as u8
    }
}

impl Owned for GuardianSetData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
pub struct BridgeConfig {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_expiration_time: u32,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee: u64,
}

#[derive(Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
pub struct BridgeData {
    /// The current guardian set index, used to decide which signature sets to accept.
    pub guardian_set_index: u32,

    /// Lamports in the collection account
    pub last_lamports: u64,

    /// Bridge configuration, which is set once upon initialization.
    pub config: BridgeConfig,
}

impl Owned for BridgeData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct SignatureSet {
    /// Signatures of validators
    pub signatures: Vec<[u8; 65]>,

    /// Hash of the data
    pub hash: [u8; 32],

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl Owned for SignatureSet {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

// This is using the same payload as the PostedVAA for backwards compatibility.
// This will be deprecated in a future release.
#[repr(transparent)]
pub struct PostedMessageData(pub MessageData);

impl BorshSerialize for PostedMessageData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write(b"msg")?;
        BorshSerialize::serialize(&self.0, writer)
    }
}

impl BorshDeserialize for PostedMessageData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        *buf = &buf[3..];
        Ok(PostedMessageData(
            <MessageData as BorshDeserialize>::deserialize(buf)?,
        ))
    }
}

impl Deref for PostedMessageData {
    type Target = MessageData;

    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl DerefMut for PostedMessageData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl Default for PostedMessageData {
    fn default() -> Self {
        PostedMessageData(MessageData::default())
    }
}

impl Clone for PostedMessageData {
    fn clone(&self) -> Self {
        PostedMessageData(self.0.clone())
    }
}

#[cfg(not(feature = "cpi"))]
impl Owned for PostedMessageData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for PostedMessageData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::Other(
            Pubkey::from_str("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o").unwrap(),
        )
    }
}

#[repr(transparent)]
pub struct PostedVAAData(pub MessageData);

impl BorshSerialize for PostedVAAData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write(b"vaa")?;
        BorshSerialize::serialize(&self.0, writer)
    }
}

impl BorshDeserialize for PostedVAAData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        *buf = &buf[3..];
        Ok(PostedVAAData(
            <MessageData as BorshDeserialize>::deserialize(buf)?,
        ))
    }
}

impl Deref for PostedVAAData {
    type Target = MessageData;

    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl DerefMut for PostedVAAData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl Default for PostedVAAData {
    fn default() -> Self {
        PostedVAAData(MessageData::default())
    }
}

impl Clone for PostedVAAData {
    fn clone(&self) -> Self {
        PostedVAAData(self.0.clone())
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize, Clone, Serialize, Deserialize)]
pub struct MessageData {
    /// Header of the posted VAA
    pub vaa_version: u8,

    /// Level of consistency requested by the emitter
    pub consistency_level: u8,

    /// Time the vaa was submitted
    pub vaa_time: u32,

    /// Account where signatures are stored
    pub vaa_signature_account: Pubkey,

    /// Time the posted message was created
    pub submission_time: u32,

    /// Unique nonce for this message
    pub nonce: u32,

    /// Sequence number of this message
    pub sequence: u64,

    /// Emitter of the message
    pub emitter_chain: u16,

    /// Emitter of the message
    pub emitter_address: [u8; 32],

    /// Message payload
    pub payload: Vec<u8>,
}

#[cfg(not(feature = "cpi"))]
impl Owned for PostedVAAData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for PostedVAAData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::Other(
            Pubkey::from_str("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o").unwrap(),
        )
    }
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize)]
pub struct SequenceTracker {
    pub sequence: u64,
}

impl Owned for SequenceTracker {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct ClaimData {
    pub claimed: bool,
}

impl Owned for ClaimData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub struct GovernancePayloadUpgrade {
    // Address of the new Implementation
    pub new_contract: Pubkey,
}

impl SerializePayload for GovernancePayloadUpgrade {
    fn serialize<W: Write>(&self, v: &mut W) -> std::result::Result<(), SolitaireError> {
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
            v.write(key)?;
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
            c.read(&mut key)?;
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
        v.write(&fee_data[..])?;

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
        v.write(&amount_data)?;
        v.write(&self.to)?;
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

#[repr(u8)]
#[derive(BorshSerialize, BorshDeserialize, Clone, Serialize, Deserialize)]
pub enum ConsistencyLevel {
    Confirmed,
    Finalized,
}
