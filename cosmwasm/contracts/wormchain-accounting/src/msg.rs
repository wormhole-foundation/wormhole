use accounting::state::{account, transfer, Account, Modification, Transfer};
use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Binary;
use wormhole::{
    vaa::{Body, Signature},
    Address,
};

use crate::state::{self, PendingTransfer};

#[cw_serde]
#[derive(Default)]
pub struct Observation {
    // The hash of the transaction on the emitter chain in which the transfer was performed.
    pub tx_hash: Binary,

    // Seconds since UNIX epoch.
    pub timestamp: u32,

    // The nonce for the transfer.
    pub nonce: u32,

    // The source chain from which this observation was created.
    pub emitter_chain: u16,

    // The address on the source chain that emitted this message.
    pub emitter_address: [u8; 32],

    // The sequence number of this observation.
    pub sequence: u64,

    // The consistency level requested by the emitter.
    pub consistency_level: u8,

    // The serialized tokenbridge payload.
    pub payload: Binary,
}

impl Observation {
    // Calculate a digest of the observation that can be used for de-duplication.
    pub fn digest(&self) -> anyhow::Result<Binary> {
        // We don't know the actual type of `self.payload` so we create a body with a 0-sized
        // payload and then just append it when calculating the digest.

        let body = Body {
            timestamp: self.timestamp,
            nonce: self.nonce,
            emitter_chain: self.emitter_chain.into(),
            emitter_address: Address(self.emitter_address),
            sequence: self.sequence,
            consistency_level: self.consistency_level,
            payload: (),
        };

        let digest = body.digest_with_payload(&self.payload)?;

        Ok(digest.secp256k_hash.to_vec().into())
    }
}

#[cw_serde]
pub struct Upgrade {
    pub new_addr: [u8; 32],
}

#[cw_serde]
pub enum ExecuteMsg {
    /// Submit a series of observations.  Once the contract has received a quorum of signatures
    /// for a particular observation, the transfer associated with the observation will be
    /// committed to the on-chain state.
    SubmitObservations {
        // A serialized `Vec<Observation>`. Multiple observations can be submitted together to reduce
        // transaction overhead.
        observations: Binary,
        // The index of the guardian set used to sign the observations.
        guardian_set_index: u32,
        // A signature for `observations`.
        signature: Signature,
    },

    /// Modifies the balance of a single account.  Used to manually override the balance.
    ModifyBalance {
        // A serialized `Modification` message.
        modification: Binary,

        // The index of the guardian set used to sign this modification.
        guardian_set_index: u32,

        // A quorum of signatures for `modification`.
        signatures: Vec<Signature>,
    },

    UpgradeContract {
        // A serialized `Upgrade` message.
        upgrade: Binary,

        // The index of the guardian set used to sign this request.
        guardian_set_index: u32,

        // A quorum of signatures for `key`.
        signatures: Vec<Signature>,
    },

    /// Submit one or more signed VAAs to update the on-chain state.  If processing any of the VAAs
    /// returns an error, the entire transaction is aborted and none of the VAAs are committed.
    SubmitVAAs {
        /// One or more VAAs to be submitted.  Each VAA should be encoded in the standard wormhole
        /// wire format.
        vaas: Vec<Binary>,
    },
}

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(account::Balance)]
    Balance(account::Key),
    #[returns(AllAccountsResponse)]
    AllAccounts {
        start_after: Option<account::Key>,
        limit: Option<u32>,
    },
    #[returns(TransferResponse)]
    Transfer(transfer::Key),
    #[returns(AllTransfersResponse)]
    AllTransfers {
        start_after: Option<transfer::Key>,
        limit: Option<u32>,
    },
    #[returns(state::Data)]
    PendingTransfer(transfer::Key),
    #[returns(AllPendingTransfersResponse)]
    AllPendingTransfers {
        start_after: Option<transfer::Key>,
        limit: Option<u32>,
    },
    #[returns(Modification)]
    Modification { sequence: u64 },
    #[returns(AllModificationsResponse)]
    AllModifications {
        start_after: Option<u64>,
        limit: Option<u32>,
    },
    #[returns(cosmwasm_std::Empty)]
    ValidateTransfer { transfer: Transfer },
    #[returns(ChainRegistrationResponse)]
    ChainRegistration { chain: u16 },
    #[returns(MissingObservationsResponse)]
    MissingObservations { guardian_set: u32, index: u8 },
}

#[cw_serde]
pub struct AllAccountsResponse {
    pub accounts: Vec<Account>,
}

#[cw_serde]
pub struct AllTransfersResponse {
    // A tuple of the transfer details and the digest.
    pub transfers: Vec<(Transfer, Binary)>,
}

#[cw_serde]
pub struct AllPendingTransfersResponse {
    pub pending: Vec<PendingTransfer>,
}

#[cw_serde]
pub struct AllModificationsResponse {
    pub modifications: Vec<Modification>,
}

#[cw_serde]
pub struct ChainRegistrationResponse {
    pub address: Binary,
}

#[cw_serde]
pub struct TransferResponse {
    pub data: transfer::Data,
    pub digest: Binary,
}

#[cw_serde]
pub struct MissingObservationsResponse {
    pub missing: Vec<MissingObservation>,
}

#[cw_serde]
pub struct MissingObservation {
    pub chain_id: u16,
    pub tx_hash: Binary,
}
