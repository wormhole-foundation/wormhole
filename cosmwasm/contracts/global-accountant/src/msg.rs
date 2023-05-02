use accountant::state::{account, transfer, Account, Modification, Transfer};
use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Binary;
use serde_wormhole::RawMessage;
use wormhole_sdk::{
    vaa::{Body, Signature},
    Address,
};

use crate::state::{self, PendingTransfer};

pub const SUBMITTED_OBSERVATIONS_PREFIX: &[u8; 35] = b"acct_sub_obsfig_000000000000000000|";

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
    #[serde(with = "hex")]
    #[schemars(with = "String")]
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
        let body = Body {
            timestamp: self.timestamp,
            nonce: self.nonce,
            emitter_chain: self.emitter_chain.into(),
            emitter_address: Address(self.emitter_address),
            sequence: self.sequence,
            consistency_level: self.consistency_level,
            payload: RawMessage::new(&self.payload),
        };

        let digest = body.digest()?;

        Ok(digest.secp256k_hash.to_vec().into())
    }
}

// The default externally-tagged serde representation of enums is awkward in JSON when the
// enum contains unit variants mixed with newtype variants.  We can't use the internally-tagged
// representation because it only supports newtype variants that contain structs or maps.  So use
// the adjacently tagged variant representation here: the enum is always encoded as an object with
// a "type" field that indicates the variant and an optional "data" field that contains the data for
// the variant, if any.
#[cw_serde]
#[serde(tag = "type", content = "data")]
pub enum ObservationStatus {
    Pending,
    Committed,
    Error(String),
}

#[cw_serde]
pub struct SubmitObservationResponse {
    pub key: transfer::Key,
    pub status: ObservationStatus,
}

#[cw_serde]
pub struct ObservationError {
    pub key: transfer::Key,
    pub error: String,
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

    /// Submit one or more signed VAAs to update the on-chain state.  If processing any of the VAAs
    /// returns an error, the entire transaction is aborted and none of the VAAs are committed.
    SubmitVaas {
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
    #[returns(AllTransfersResponse)]
    AllTransfers {
        start_after: Option<transfer::Key>,
        limit: Option<u32>,
    },
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
    #[returns(TransferStatus)]
    TransferStatus(transfer::Key),
    #[returns(BatchTransferStatusResponse)]
    BatchTransferStatus(Vec<transfer::Key>),
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
pub struct MissingObservationsResponse {
    pub missing: Vec<MissingObservation>,
}

#[cw_serde]
pub struct MissingObservation {
    pub chain_id: u16,
    pub tx_hash: Binary,
}

#[cw_serde]
pub enum TransferStatus {
    Pending(Vec<state::Data>),
    Committed {
        data: transfer::Data,
        digest: Binary,
    },
}

#[cw_serde]
pub struct TransferDetails {
    // The key for the transfer.
    pub key: transfer::Key,
    // The status of the transfer.  If `status` is `None`, then there is no transfer associated
    // with `key`.
    pub status: Option<TransferStatus>,
}

#[cw_serde]
pub struct BatchTransferStatusResponse {
    pub details: Vec<TransferDetails>,
}
