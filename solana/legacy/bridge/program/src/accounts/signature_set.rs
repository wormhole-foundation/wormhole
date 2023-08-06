//! PostedMessage

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solitaire::{
    AccountOwner,
    AccountState,
    Data,
    Owned,
};

pub type SignatureSet<'b, const State: AccountState> = Data<'b, SignatureSetData, { State }>;

#[derive(Default, BorshSerialize, BorshDeserialize)]
pub struct SignatureSetData {
    /// Signatures of validators
    pub signatures: Vec<bool>,

    /// Hash of the data
    pub hash: [u8; 32],

    /// Index of the guardian set
    pub guardian_set_index: u32,
}

impl Owned for SignatureSetData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}
