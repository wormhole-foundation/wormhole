use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{HumanAddr, StdResult, Storage};
use cosmwasm_storage::{
    bucket, bucket_read, singleton, singleton_read, Bucket, ReadonlyBucket, ReadonlySingleton,
    Singleton,
};

use wormhole::byte_utils::ByteUtils;


pub static CONFIG_KEY: &[u8] = b"config";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";
pub static WRAPPED_ASSET_ADDRESS_KEY: &[u8] = b"wrapped_asset_address";
pub static BRIDGE_CONTRACTS: &[u8] = b"bridge_contracts";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // Current active guardian set
    pub owner: HumanAddr,
    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,
}

pub fn config<S: Storage>(storage: &mut S) -> Singleton<S, ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read<S: Storage>(storage: &S) -> ReadonlySingleton<S, ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn bridge_contracts<S: Storage>(storage: &mut S) -> Bucket<S, Vec<u8>> {
    bucket(BRIDGE_CONTRACTS, storage)
}

pub fn bridge_contracts_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, Vec<u8>> {
    bucket_read(BRIDGE_CONTRACTS, storage)
}

pub fn wrapped_asset<S: Storage>(storage: &mut S) -> Bucket<S, HumanAddr> {
    bucket(WRAPPED_ASSET_KEY, storage)
}

pub fn wrapped_asset_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, HumanAddr> {
    bucket_read(WRAPPED_ASSET_KEY, storage)
}

pub fn wrapped_asset_address<S: Storage>(storage: &mut S) -> Bucket<S, Vec<u8>> {
    bucket(WRAPPED_ASSET_ADDRESS_KEY, storage)
}

pub fn wrapped_asset_address_read<S: Storage>(storage: &S) -> ReadonlyBucket<S, Vec<u8>> {
    bucket_read(WRAPPED_ASSET_ADDRESS_KEY, storage)
}




pub struct Action;

impl Action {
    pub const TRANSFER: u8 = 0;
    pub const ATTEST_META: u8 = 1;
}

// 0 u8 action
// 1 [u8] payload

pub struct TokenBridgeMessage {
    pub action: u8,
    pub payload: Vec<u8>,
}

impl TokenBridgeMessage {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let action = data.get_u8(0);
        let payload = &data[1..];

        Ok(TokenBridgeMessage {
            action,
            payload: payload.to_vec(),
        })
    }

    pub fn serialize(&self) ->Vec<u8> {
        [self.action.to_be_bytes().to_vec(), self.payload.clone()].concat()
    }
}

//     0   u16      token_chain
//     2   [u8; 32] token_address
//     34  u256     amount
//     66  u16      recipient_chain
//     68  [u8; 32] recipient

pub struct TransferInfo {
    pub token_chain: u16,
    pub token_address: Vec<u8>,
    pub amount: (u128, u128),
    pub recipient_chain: u16,
    pub recipient: Vec<u8>,
}

impl TransferInfo {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let token_chain = data.get_u16(0);
        let token_address = data.get_bytes32(2).to_vec();
        let amount = data.get_u256(34);
        let recipient_chain = data.get_u16(66);
        let recipient = data.get_bytes32(68).to_vec();

        Ok(TransferInfo {
            token_chain,
            token_address,
            amount,
            recipient_chain,
            recipient,
        })
    }
    pub fn serialize(&self) -> Vec<u8> {
        [
            self.token_chain.to_be_bytes().to_vec(),
            self.token_address.clone(),
            self.amount.0.to_be_bytes().to_vec(),
            self.amount.1.to_be_bytes().to_vec(),
            self.recipient_chain.to_be_bytes().to_vec(),
            self.recipient.to_vec(),
        ]
        .concat()
    }
}

//PayloadID uint8 = 2
// // Address of the token. Left-zero-padded if shorter than 32 bytes
// TokenAddress [32]uint8
// // Chain ID of the token
// TokenChain uint16
// // Number of decimals of the token (big-endian uint256)
// Decimals [32]uint8
// // Symbol of the token (UTF-8)
// Symbol [32]uint8
// // Name of the token (UTF-8)
// Name [32]uint8

pub struct AssetMeta {
    pub token_chain: u16,
    pub token_address: Vec<u8>,
    pub decimals: u8,
    pub symbol: Vec<u8>,
    pub name: Vec<u8>,
}

impl AssetMeta {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let token_chain = data.get_u16(0);
        let token_address = data.get_bytes32(2).to_vec();
        let decimals = data.get_u8(34);
        let symbol = data.get_bytes32(35).to_vec();
        let name = data.get_bytes32(67).to_vec();

        Ok(AssetMeta {
            token_chain,
            token_address,
            decimals,
            symbol,
            name,
        })
    }

    pub fn serialize(&self) -> Vec<u8> {
        [
            self.token_chain.to_be_bytes().to_vec(),
            self.token_address.clone(),
            self.decimals.to_be_bytes().to_vec(),
            self.symbol.clone(),
            self.name.clone(),
        ]
        .concat()
    }
}
