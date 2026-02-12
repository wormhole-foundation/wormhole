//! GuardianSet represents an account containing information about the current active guardians
//! responsible for signing wormhole VAAs.

use crate::types::GuardianPublicKey;
use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solitaire::{
    processors::seeded::{
        Seeded,
        SingleOwned,
    },
    AccountOwner,
    AccountState,
    Data,
    Owned,
};
use std::io::{
    Error,
    ErrorKind::InvalidData,
    Write,
};

pub type GuardianSet<'b, const State: AccountState> = Data<'b, GuardianSetData, { State }>;

/// Account discriminator for GuardianSetData — prevents type confusion attacks.
pub const GUARDIAN_SET_DISCRIMINATOR: &[u8] = b"gset";

#[derive(Default, Serialize, Deserialize)]
pub struct GuardianSetData {
    /// Index representing an incrementing version number for this guardian set.
    pub index: u32,

    /// ETH style public keys
    pub keys: Vec<GuardianPublicKey>,

    /// Timestamp representing the time this guardian became active.
    pub creation_time: u32,

    /// Expiration time when VAAs issued by this set are no longer valid.
    pub expiration_time: u32,
}

impl BorshSerialize for GuardianSetData {
    fn serialize<W: Write>(&self, writer: &mut W) -> std::io::Result<()> {
        writer.write_all(GUARDIAN_SET_DISCRIMINATOR)?;
        BorshSerialize::serialize(&self.index, writer)?;
        BorshSerialize::serialize(&self.keys, writer)?;
        BorshSerialize::serialize(&self.creation_time, writer)?;
        BorshSerialize::serialize(&self.expiration_time, writer)
    }
}

impl BorshDeserialize for GuardianSetData {
    fn deserialize(buf: &mut &[u8]) -> std::io::Result<Self> {
        if buf.len() < GUARDIAN_SET_DISCRIMINATOR.len() {
            return Err(Error::new(InvalidData, "Not enough bytes for GuardianSetData discriminator"));
        }
        let magic = &buf[..GUARDIAN_SET_DISCRIMINATOR.len()];
        if magic != GUARDIAN_SET_DISCRIMINATOR {
            return Err(Error::new(
                InvalidData,
                format!("GuardianSetData discriminator mismatch. Expected {:?} but got {:?}", GUARDIAN_SET_DISCRIMINATOR, magic),
            ));
        }
        *buf = &buf[GUARDIAN_SET_DISCRIMINATOR.len()..];
        Ok(GuardianSetData {
            index: BorshDeserialize::deserialize(buf)?,
            keys: BorshDeserialize::deserialize(buf)?,
            creation_time: BorshDeserialize::deserialize(buf)?,
            expiration_time: BorshDeserialize::deserialize(buf)?,
        })
    }
}

/// GuardianSet account PDAs are indexed by their version number.
pub struct GuardianSetDerivationData {
    pub index: u32,
}

impl<'a, const State: AccountState> Seeded<&GuardianSetDerivationData>
    for GuardianSet<'a, { State }>
{
    fn seeds(data: &GuardianSetDerivationData) -> Vec<Vec<u8>> {
        vec![
            "GuardianSet".as_bytes().to_vec(),
            data.index.to_be_bytes().to_vec(),
        ]
    }
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

impl SingleOwned for GuardianSetData {
}
