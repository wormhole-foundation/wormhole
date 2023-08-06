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
    processors::seeded::Seeded,
    AccountOwner,
    AccountState,
    Data,
    Owned,
};

pub type GuardianSet<'b, const State: AccountState> = Data<'b, GuardianSetData, { State }>;

#[derive(Default, BorshSerialize, BorshDeserialize, Serialize, Deserialize)]
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
