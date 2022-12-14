use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, CustomQuery, Empty};
use wormhole::vaa::Signature;

#[cw_serde]
#[derive(QueryResponses)]
pub enum WormholeQuery {
    /// Verifies that `data` has been signed by a quorum of guardians from `guardian_set_index`.
    #[returns(Empty)]
    VerifyQuorum {
        data: Binary,
        guardian_set_index: u32,
        signatures: Vec<Signature>,
    },

    /// Verifies that `data` has been signed by a guardian from `guardian_set_index`.
    #[returns(Empty)]
    VerifySignature {
        data: Binary,
        guardian_set_index: u32,
        signature: Signature,
    },

    /// Returns the number of signatures necessary for quorum for the given guardian set index.
    #[returns(u32)]
    CalculateQuorum { guardian_set_index: u32 },
}

impl CustomQuery for WormholeQuery {}
