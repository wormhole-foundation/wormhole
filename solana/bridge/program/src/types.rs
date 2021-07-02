use crate::{
    api::ForeignAddress,
    vaa::{
        DeserializeGovernancePayload,
        DeserializePayload,
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
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::{
        AccountOwner,
        Owned,
    },
    trace,
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

#[derive(Default, BorshSerialize, BorshDeserialize)]
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

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct BridgeConfig {
    /// Period for how long a guardian set is valid after it has been replaced by a new one.  This
    /// guarantees that VAAs issued by that set can still be submitted for a certain period.  In
    /// this period we still trust the old guardian set.
    pub guardian_set_expiration_time: u32,

    /// Amount of lamports that needs to be paid to the protocol to post a message
    pub fee: u64,
}

#[derive(Default, BorshSerialize, BorshDeserialize)]
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

#[repr(transparent)]
pub struct PostedMessage(pub PostedMessageData);

impl BorshSerialize for PostedMessage {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write(b"msg")?;
        BorshSerialize::serialize(&self.0, writer)
    }
}

impl BorshDeserialize for PostedMessage {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        *buf = &buf[3..];
        Ok(PostedMessage(
            <PostedMessageData as BorshDeserialize>::deserialize(buf)?,
        ))
    }
}

impl Deref for PostedMessage {
    type Target = PostedMessageData;

    fn deref(&self) -> &Self::Target {
        unsafe { std::mem::transmute(&self.0) }
    }
}

impl DerefMut for PostedMessage {
    fn deref_mut(&mut self) -> &mut Self::Target {
        unsafe { std::mem::transmute(&mut self.0) }
    }
}

impl Default for PostedMessage {
    fn default() -> Self {
        PostedMessage(PostedMessageData::default())
    }
}

impl Clone for PostedMessage {
    fn clone(&self) -> Self {
        PostedMessage(self.0.clone())
    }
}

#[derive(Default, BorshSerialize, BorshDeserialize, Clone)]
pub struct PostedMessageData {
    /// Header of the posted VAA
    pub vaa_version: u8,

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

impl Owned for PostedMessage {
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

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize)]
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

impl DeserializePayload for GovernancePayloadUpgrade
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);

        Self::check_governance_header(&mut c)?;

        let mut addr = [0u8; 32];
        c.read_exact(&mut addr)?;

        Ok(GovernancePayloadUpgrade {
            new_contract: Pubkey::new(&addr[..]),
        })
    }
}

impl DeserializeGovernancePayload for GovernancePayloadUpgrade {
    const MODULE: &'static str = "CORE";
    const ACTION: u8 = 2;
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
            v.write(key);
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

        let new_index = c.read_u32::<BigEndian>()?;

        let keys_len = c.read_u8()?;
        let mut keys = Vec::with_capacity(keys_len as usize);
        for _ in 0..keys_len {
            let mut key: [u8; 20] = [0; 20];
            c.read(&mut key)?;
            keys.push(key);
        }

        Ok(GovernancePayloadGuardianSetChange {
            new_guardian_set_index: new_index,
            new_guardian_set: keys,
        })
    }
}

impl DeserializeGovernancePayload for GovernancePayloadGuardianSetChange {
    const MODULE: &'static str = "CORE";
    const ACTION: u8 = 1;
}

pub struct GovernancePayloadSetMessageFee {
    // New fee in lamports
    pub fee: u64,
}

impl DeserializePayload for GovernancePayloadSetMessageFee
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);

        let fee = c.read_u64::<BigEndian>()?;
        Ok(GovernancePayloadSetMessageFee { fee })
    }
}

impl DeserializeGovernancePayload for GovernancePayloadSetMessageFee {
    const MODULE: &'static str = "CORE";
    const ACTION: u8 = 3;
}

pub struct GovernancePayloadTransferFees {
    // Amount to be transferred
    pub amount: U256,
    // Recipient
    pub to: ForeignAddress,
}

impl DeserializePayload for GovernancePayloadTransferFees
where
    Self: DeserializeGovernancePayload,
{
    fn deserialize(buf: &mut &[u8]) -> Result<Self, SolitaireError> {
        let mut c = Cursor::new(buf);

        let mut amount_data: [u8; 32] = [0; 32];
        c.read_exact(&mut amount_data)?;
        let amount = U256::from_big_endian(&amount_data);

        let mut to = ForeignAddress::default();
        c.read_exact(&mut to)?;

        Ok(GovernancePayloadTransferFees { amount, to })
    }
}

impl DeserializeGovernancePayload for GovernancePayloadTransferFees {
    const MODULE: &'static str = "CORE";
    const ACTION: u8 = 4;
}
