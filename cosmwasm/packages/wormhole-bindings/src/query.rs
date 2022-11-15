use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{Binary, CustomQuery, Empty};

#[cw_serde]
pub struct Signature {
    /// The index of the guardian in the guardian set.
    pub index: u8,

    /// The signature, which should be exactly 65 bytes with the following layout:
    ///
    /// ```markdown
    /// 0  .. 64: Signature   (ECDSA)
    /// 64 .. 65: Recovery ID (ECDSA)
    /// ```
    pub signature: Binary,
}

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
