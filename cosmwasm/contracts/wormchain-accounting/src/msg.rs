use accounting::state::{account, transfer, Account, Modification, Transfer};
use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::Binary;
use wormhole_bindings::Signature;

use crate::state::{self, PendingTransfer};

#[cw_serde]
pub struct Instantiate {
    pub tokenbridge_addr: String,
    pub accounts: Vec<Account>,
    pub transfers: Vec<Transfer>,
    pub modifications: Vec<Modification>,
}

impl From<Instantiate> for accounting::msg::Instantiate {
    fn from(i: Instantiate) -> Self {
        Self {
            accounts: i.accounts,
            transfers: i.transfers,
            modifications: i.modifications,
        }
    }
}

#[cw_serde]
pub struct InstantiateMsg {
    // A serialized `Instantiate` message.
    pub instantiate: Binary,
    // The index of the guardian set used to sign this message.
    pub guardian_set_index: u32,
    // A quorum of signatures for `instantiate`.
    pub signatures: Vec<Signature>,
}

#[cw_serde]
#[derive(Default)]
pub struct Observation {
    // The key that uniquely identifies the observation.
    pub key: transfer::Key,

    // The nonce for the transfer.
    pub nonce: u32,

    // The hash of the transaction on the emitter chain in which the transfer
    // was performed.
    pub tx_hash: Binary,

    // The serialized tokenbridge payload.
    pub payload: Binary,
}

#[cw_serde]
pub struct Upgrade {
    pub new_addr: [u8; 32],
}

#[cw_serde]
pub enum ExecuteMsg {
    SubmitObservations {
        // A serialized `Vec<Observation>`. Multiple observations can be submitted together to reduce
        // transaction overhead.
        observations: Binary,
        // The index of the guardian set used to sign the observations.
        guardian_set_index: u32,
        // A signature for `observations`.
        signature: Signature,
    },
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
    #[returns(transfer::Data)]
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
}

#[cw_serde]
pub struct AllAccountsResponse {
    pub accounts: Vec<Account>,
}

#[cw_serde]
pub struct AllTransfersResponse {
    pub transfers: Vec<Transfer>,
}

#[cw_serde]
pub struct AllPendingTransfersResponse {
    pub pending: Vec<PendingTransfer>,
}

#[cw_serde]
pub struct AllModificationsResponse {
    pub modifications: Vec<Modification>,
}
