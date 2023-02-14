use std::fmt;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::Uint256;
use cw_storage_plus::Map;

pub mod account;
mod addr;
pub mod transfer;

pub use account::Account;
pub use addr::TokenAddress;
pub use transfer::Transfer;

pub const ACCOUNTS: Map<account::Key, account::Balance> = Map::new("accountant/accounts");

pub const TRANSFERS: Map<transfer::Key, transfer::Data> = Map::new("accountant/transfers");

#[cw_serde]
#[derive(Eq, PartialOrd, Ord)]
pub enum Kind {
    Add = 1,
    Sub = 2,
}

impl fmt::Display for Kind {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Kind::Add => f.write_str("ADD"),
            Kind::Sub => f.write_str("SUB"),
        }
    }
}

#[cw_serde]
#[derive(Eq, PartialOrd, Ord)]
pub struct Modification {
    // The sequence number of this modification.  Each modification must be
    // uniquely identifiable just by its sequnce number.
    pub sequence: u64,
    // The chain id of the account to be modified.
    pub chain_id: u16,
    // The chain id of the native chain for the token.
    pub token_chain: u16,
    // The address of the token on its native chain.
    pub token_address: TokenAddress,
    // The kind of modification to be made.
    pub kind: Kind,
    // The amount to be modified.
    pub amount: Uint256,
    // A human-readable reason for the modification.
    pub reason: String,
}

pub const MODIFICATIONS: Map<u64, Modification> = Map::new("accountant/modifications");
