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
    Info,
};

pub type FeeCollector<'a> = Derive<Info<'a>, "fee_collector">;
pub type Bridge<'a, const State: AccountState> = Derive<Data<'a, BridgeData, { State }>, "Bridge">;

pub type GuardianSet<'b, const State: AccountState> = Data<'b, types::GuardianSetData, { State }>;

pub struct GuardianSetDerivationData {
    pub index: u32,
}

impl<'b, const State: AccountState> Seeded<&GuardianSetDerivationData>
    for GuardianSet<'b, { State }>
{
    fn seeds(data: &GuardianSetDerivationData) -> Vec<Vec<u8>> {
        vec![
            "GuardianSet".as_bytes().to_vec(),
            data.index.to_be_bytes().to_vec(),
        ]
    }
}

pub type Claim<'b, const State: AccountState> = Data<'b, types::ClaimData, { State }>;

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

pub type SignatureSet<'b, const State: AccountState> = Data<'b, types::SignatureSet, { State }>;

pub struct SignatureSetDerivationData {
    pub hash: [u8; 32],
}

impl<'b, const State: AccountState> Seeded<&SignatureSetDerivationData>
    for SignatureSet<'b, { State }>
{
    fn seeds(data: &SignatureSetDerivationData) -> Vec<Vec<u8>> {
        vec![data.hash.to_vec()]
    }
}

pub type Message<'b, const State: AccountState> = Data<'b, PostedMessage, { State }>;

pub struct MessageDerivationData {
    pub emitter_key: [u8; 32],
    pub emitter_chain: u16,
    pub nonce: u32,
    pub payload: Vec<u8>,
}

impl<'b, const State: AccountState> Seeded<&MessageDerivationData> for Message<'b, { State }> {
    fn seeds(data: &MessageDerivationData) -> Vec<Vec<u8>> {
        let mut seeds = vec![
            data.emitter_key.to_vec(),
            data.emitter_chain.to_be_bytes().to_vec(),
            data.nonce.to_be_bytes().to_vec(),
        ];
        seeds.append(&mut data.payload.chunks(32).map(|v| v.to_vec()).collect());
        seeds
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
