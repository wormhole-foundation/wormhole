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
    io::{
        Error,
        ErrorKind::InvalidData,
        Write,
    },
    ops::{
        Deref,
        DerefMut,
    },
};

pub type PostedMessage<'a, const State: AccountState> = Data<'a, PostedMessageData, { State }>;

#[repr(transparent)]
#[derive(Default)]
pub struct PostedMessageData {
    pub message: MessageData,
}

pub type PostedMessageUnreliable<'a, const State: AccountState> =
    Data<'a, PostedMessageUnreliableData, { State }>;

#[repr(transparent)]
#[derive(Default)]
pub struct PostedMessageUnreliableData {
    pub message: MessageData,
}

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

// PostedMessageData impls

impl BorshSerialize for PostedMessageData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(b"msg")?;
        BorshSerialize::serialize(&self.message, writer)
    }
}

impl BorshDeserialize for PostedMessageData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        if buf.len() < 3 {
            return Err(Error::new(InvalidData, "Not enough bytes"));
        }

        let expected = b"msg";
        let magic: &[u8] = &buf[0..3];
        if magic != expected {
            return Err(Error::new(
                InvalidData,
                format!(
                    "Magic mismatch. Expected {:?} but got {:?}",
                    expected, magic
                ),
            ));
        };
        *buf = &buf[3..];
        Ok(PostedMessageData {
            message: <MessageData as BorshDeserialize>::deserialize(buf)?,
        })
    }
}

impl Deref for PostedMessageData {
    type Target = MessageData;

    fn deref(&self) -> &Self::Target {
        &self.message
    }
}

impl DerefMut for PostedMessageData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.message
    }
}

impl Clone for PostedMessageData {
    fn clone(&self) -> Self {
        PostedMessageData {
            message: self.message.clone(),
        }
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

// PostedMessageUnreliableData impls

impl BorshSerialize for PostedMessageUnreliableData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(b"msu")?;
        BorshSerialize::serialize(&self.message, writer)
    }
}

impl BorshDeserialize for PostedMessageUnreliableData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        if buf.len() < 3 {
            return Err(Error::new(InvalidData, "Not enough bytes"));
        }

        let expected = b"msu";
        let magic: &[u8] = &buf[0..3];
        if magic != expected {
            return Err(Error::new(
                InvalidData,
                format!(
                    "Magic mismatch. Expected {:?} but got {:?}",
                    expected, magic
                ),
            ));
        };
        *buf = &buf[3..];
        Ok(PostedMessageUnreliableData {
            message: <MessageData as BorshDeserialize>::deserialize(buf)?,
        })
    }
}

impl Deref for PostedMessageUnreliableData {
    type Target = MessageData;

    fn deref(&self) -> &Self::Target {
        &self.message
    }
}

impl DerefMut for PostedMessageUnreliableData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.message
    }
}

impl Clone for PostedMessageUnreliableData {
    fn clone(&self) -> Self {
        PostedMessageUnreliableData {
            message: self.message.clone(),
        }
    }
}

#[cfg(not(feature = "cpi"))]
impl Owned for PostedMessageUnreliableData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for PostedMessageUnreliableData {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("BRIDGE_ADDRESS")).unwrap())
    }
}
