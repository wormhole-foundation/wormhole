//! ClaimData accounts are one off markers that can be combined with other accounts to represent
//! data that can only be used once.

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

pub type Claim<'a, const State: AccountState> = Data<'a, ClaimData, { State }>;

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct ClaimData {
    pub claimed: bool,
}

impl Owned for ClaimData {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub struct ClaimDerivationData {
    pub emitter_address: [u8; 32],
    pub emitter_chain: u16,
    pub sequence: u64,
}

impl<'b, const State: AccountState> Seeded<&ClaimDerivationData> for Claim<'b, { State }> {
    fn seeds(data: &ClaimDerivationData) -> Vec<Vec<u8>> {
        return vec![
            data.emitter_address.to_vec(),
            data.emitter_chain.to_be_bytes().to_vec(),
            data.sequence.to_be_bytes().to_vec(),
        ];
    }
}
