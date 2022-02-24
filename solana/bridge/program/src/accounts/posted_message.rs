use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    AccountOwner,
    AccountState,
    Data,
    Owned,
};
use std::{
    io::Write,
    ops::{
        Deref,
        DerefMut,
    },
};

pub type PostedMessage<'a, const State: AccountState> = Data<'a, PostedMessageData, { State }>;

// This is using the same payload as the PostedVAA for backwards compatibility.
// This will be deprecated in a future release.
#[repr(transparent)]
pub struct PostedMessageData(pub MessageData);

#[derive(Debug, Default, BorshSerialize, BorshDeserialize, Clone, Serialize, Deserialize)]
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
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("BRIDGE_ADDRESS")).unwrap())
    }
}
