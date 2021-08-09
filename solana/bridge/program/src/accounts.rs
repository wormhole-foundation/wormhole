use crate::{
    types,
    types::{
        BridgeData,
        PostedMessageData,
        PostedVAAData,
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

pub type PostedMessage<'b, const State: AccountState> = Data<'b, PostedMessageData, { State }>;

pub type PostedVAA<'b, const State: AccountState> = Data<'b, PostedVAAData, { State }>;

pub struct PostedVAADerivationData {
    pub payload_hash: Vec<u8>,
}

impl<'b, const State: AccountState> Seeded<&PostedVAADerivationData> for PostedVAA<'b, { State }> {
    fn seeds(data: &PostedVAADerivationData) -> Vec<Vec<u8>> {
        vec!["PostedVAA".as_bytes().to_vec(), data.payload_hash.to_vec()]
    }
}

pub type Sequence<'b> = Data<'b, SequenceTracker, { AccountState::MaybeInitialized }>;

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
