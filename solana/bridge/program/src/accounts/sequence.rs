use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    AccountOwner,
    AccountState,
    Data,
    Owned,
};

pub type Sequence<'b> = Data<'b, SequenceTracker, { AccountState::MaybeInitialized }>;

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize)]
pub struct SequenceTracker {
    pub sequence: u64,
}

pub struct SequenceDerivationData<'a> {
    pub emitter_key: &'a Pubkey,
}

impl<'b> Seeded<&SequenceDerivationData<'b>> for Sequence<'b> {
    fn seeds(data: &SequenceDerivationData) -> Vec<Vec<u8>> {
        vec![
            "Sequence".as_bytes().to_vec(),
            data.emitter_key.to_bytes().to_vec(),
        ]
    }
}

impl Owned for SequenceTracker {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}
