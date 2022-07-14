use crate::MessageData;
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solitaire::{
    processors::seeded::Seeded,
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

pub type PostedVAA<'b, const State: AccountState> = Data<'b, PostedVAAData, { State }>;

pub struct PostedVAADerivationData {
    pub payload_hash: Vec<u8>,
}

impl<'a, const State: AccountState> Seeded<&PostedVAADerivationData> for PostedVAA<'a, { State }> {
    fn seeds(data: &PostedVAADerivationData) -> Vec<Vec<u8>> {
        vec![b"PostedVAA".to_vec(), data.payload_hash.to_vec()]
    }
}

#[repr(transparent)]
#[derive(Default)]
pub struct PostedVAAData {
    pub message: MessageData,
}

impl BorshSerialize for PostedVAAData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(b"vaa")?;
        BorshSerialize::serialize(&self.message, writer)
    }
}

impl BorshDeserialize for PostedVAAData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        if buf.len() < 3 {
            return Err(Error::new(InvalidData, "Not enough bytes"));
        }

        // We accept "vaa", "msg", or "msu" because it's convenient to read all of these as PostedVAAData
        let expected: [&[u8]; 3] = [b"vaa", b"msg", b"msu"];
        let magic: &[u8] = &buf[0..3];
        if !expected.contains(&magic) {
            return Err(Error::new(InvalidData, "Magic mismatch."));
        };
        *buf = &buf[3..];
        Ok(PostedVAAData {
            message: <MessageData as BorshDeserialize>::deserialize(buf)?,
        })
    }
}

impl Deref for PostedVAAData {
    type Target = MessageData;

    fn deref(&self) -> &Self::Target {
        &self.message
    }
}

impl DerefMut for PostedVAAData {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.message
    }
}

impl Clone for PostedVAAData {
    fn clone(&self) -> Self {
        PostedVAAData {
            message: self.message.clone(),
        }
    }
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
        use solana_program::pubkey::Pubkey;
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("BRIDGE_ADDRESS")).unwrap())
    }
}
