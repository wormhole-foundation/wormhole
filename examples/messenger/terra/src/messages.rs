//! CosmWasm defines Message types to its various entrypoints. These are JSON encoded and decoded
//! by the runtime before being passed into the various entrypoints.

use cosmwasm_std::Binary;
use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

/// InstantiateMsg is passed into the contract initialiser when the contract is first deployed,
/// this is a one off message.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {
    pub version: String,
}

/// ExecuteMsg is passed into the execute contract handler whenever a user submits a transaction
/// targetting our contract, this is our "main" entrypoint.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum ExecuteMsg {
    RecvMessage {
        vaa: Binary,
    },

    SendMessage {
        nonce: u32,
        nick:  String,
        text:  String,
    },
}
