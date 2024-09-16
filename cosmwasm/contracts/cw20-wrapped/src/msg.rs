#![allow(clippy::field_reassign_with_default)]
use cosmwasm_schema::{cw_serde, QueryResponses};

use cosmwasm_std::{Addr, Binary, Uint128};
use cw20::{AllowanceResponse, BalanceResponse, Expiration, TokenInfoResponse};

type HumanAddr = String;

#[cw_serde]
pub struct InstantiateMsg {
    pub name: String,
    pub symbol: String,
    pub asset_chain: u16,
    pub asset_address: Binary,
    pub decimals: u8,
    pub mint: Option<InitMint>,
    pub init_hook: Option<InitHook>,
}

#[cw_serde]
pub struct InitHook {
    pub msg: Binary,
    pub contract_addr: HumanAddr,
}

#[cw_serde]
pub struct InitMint {
    pub recipient: HumanAddr,
    pub amount: Uint128,
}

#[cw_serde]
pub struct MigrateMsg {}

#[cw_serde]
pub enum ExecuteMsg {
    /// Implements CW20. Transfer is a base message to move tokens to another account without triggering actions
    Transfer {
        recipient: HumanAddr,
        amount: Uint128,
    },
    /// Slightly different than CW20. Burn is a base message to destroy tokens forever
    Burn { account: HumanAddr, amount: Uint128 },
    /// Implements CW20. Send is a base message to transfer tokens to a contract and trigger an action
    /// on the receiving contract.
    Send {
        contract: HumanAddr,
        amount: Uint128,
        msg: Binary,
    },
    /// Implements CW20 "mintable" extension. If authorized, creates amount new tokens
    /// and adds to the recipient balance.
    Mint {
        recipient: HumanAddr,
        amount: Uint128,
    },
    /// Implements CW20 "approval" extension. Allows spender to access an additional amount tokens
    /// from the owner's (env.sender) account. If expires is Some(), overwrites current allowance
    /// expiration with this one.
    IncreaseAllowance {
        spender: HumanAddr,
        amount: Uint128,
        expires: Option<Expiration>,
    },
    /// Implements CW20 "approval" extension. Lowers the spender's access of tokens
    /// from the owner's (env.sender) account by amount. If expires is Some(), overwrites current
    /// allowance expiration with this one.
    DecreaseAllowance {
        spender: HumanAddr,
        amount: Uint128,
        expires: Option<Expiration>,
    },
    /// Implements CW20 "approval" extension. Transfers amount tokens from owner -> recipient
    /// if `env.sender` has sufficient pre-approval.
    TransferFrom {
        owner: HumanAddr,
        recipient: HumanAddr,
        amount: Uint128,
    },
    /// Implements CW20 "approval" extension. Sends amount tokens from owner -> contract
    /// if `env.sender` has sufficient pre-approval.
    SendFrom {
        owner: HumanAddr,
        contract: HumanAddr,
        amount: Uint128,
        msg: Binary,
    },
    /// Implements CW20 "approval" extension. Destroys tokens forever
    BurnFrom { owner: HumanAddr, amount: Uint128 },
    /// Extend Interface with the ability to update token metadata.
    UpdateMetadata { name: String, symbol: String },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(WrappedAssetInfoResponse)]
    /// Generic information about the wrapped asset
    WrappedAssetInfo {},

    #[returns(BalanceResponse)]
    /// Implements CW20. Returns the current balance of the given address, 0 if unset.
    Balance {
        address: HumanAddr,
    },

    #[returns(TokenInfoResponse)]
    /// Implements CW20. Returns metadata on the contract - name, decimals, supply, etc.
    TokenInfo {},

    #[returns(AllowanceResponse)]
    /// Implements CW20 "allowance" extension.
    /// Returns how much spender can use from owner account, 0 if unset.
    Allowance {
        owner: HumanAddr,
        spender: HumanAddr,
    },
}

#[cw_serde]
pub struct WrappedAssetInfoResponse {
    pub asset_chain: u16,      // Asset chain id
    pub asset_address: Binary, // Asset smart contract address in the original chain
    pub bridge: Addr,          // Bridge address, authorized to mint and burn wrapped tokens
}
