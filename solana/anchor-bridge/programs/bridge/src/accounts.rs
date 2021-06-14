use crate::{
    types,
    types::{
        BridgeData,
        PostedMessage,
        SequenceTracker,
    },
};
use solana_program::pubkey::Pubkey;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
    Data,
    Derive,
};

pub type Bridge<'a, const State: AccountState> = Derive<Data<'a, BridgeData, { State }>, "Bridge">;

pub type GuardianSet<'b, const State: AccountState> = Data<'b, types::GuardianSetData, { State }>;

pub struct GuardianSetDerivationData {
    pub index: u32,
}

impl<'b, const State: AccountState> Seeded<&GuardianSetDerivationData>
    for GuardianSet<'b, { State }>
{
    fn seeds(data: &GuardianSetDerivationData) -> Vec<Vec<u8>> {
        vec![data.index.to_be_bytes().to_vec()]
    }
}

pub type SignatureSet<'b, const State: AccountState> = Data<'b, types::SignatureSet, { State }>;

pub struct SignaturesSetDerivationData {
    pub hash: [u8; 32],
}

impl<'b, const State: AccountState> Seeded<&SignaturesSetDerivationData>
    for SignatureSet<'b, { State }>
{
    fn seeds(data: &SignaturesSetDerivationData) -> Vec<Vec<u8>> {
        vec![data.hash.to_vec()]
    }
}

pub type Message<'b, const State: AccountState> = Data<'b, PostedMessage, { State }>;

pub struct MessageDerivationData {
    pub emitter_key: [u8; 32],
    pub sequence: u64,
}

impl<'b, const State: AccountState> Seeded<&MessageDerivationData> for Message<'b, { State }> {
    fn seeds(data: &MessageDerivationData) -> Vec<Vec<u8>> {
        vec![
            data.emitter_key.to_vec(),
            data.sequence.to_be_bytes().to_vec(),
        ]
    }
}

pub type Sequence<'b> = Data<'b, SequenceTracker, { AccountState::MaybeInitialized }>;

pub struct SequenceDerivationData<'a> {
    pub emitter_key: &'a Pubkey,
}

impl<'b> Seeded<&SequenceDerivationData<'b>> for Sequence<'b> {
    fn seeds(data: &SequenceDerivationData) -> Vec<Vec<u8>> {
        vec![data.emitter_key.to_bytes().to_vec()]
    }
}
