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
    io::Write,
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
        vec!["PostedVAA".as_bytes().to_vec(), data.payload_hash.to_vec()]
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
#[cfg(not(feature = "cpi"))]
impl Owned for PostedVAAData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for PostedVAAData {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        use solana_program::pubkey::Pubkey;
        AccountOwner::Other(Pubkey::from_str(env!("BRIDGE_ADDRESS")).unwrap())
    }
}
