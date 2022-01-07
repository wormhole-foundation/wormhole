use std::convert::TryInto;

use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

use cosmwasm_std::{
    StdError,
    StdResult,
    Storage,
};
use cosmwasm_storage::{
    bucket,
    bucket_read,
    singleton,
    singleton_read,
    Bucket,
    ReadonlyBucket,
    ReadonlySingleton,
    Singleton,
};

use wormhole::byte_utils::ByteUtils;

type HumanAddr = String;

pub static CONFIG_KEY: &[u8] = b"config";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";
pub static WRAPPED_ASSET_ADDRESS_KEY: &[u8] = b"wrapped_asset_address";
pub static BRIDGE_CONTRACTS_KEY: &[u8] = b"bridge_contracts";
pub static TOKEN_ID_HASHES_KEY: &[u8] = b"token_id_hashes";
pub static SPL_CACHE_KEY: &[u8] = b"spl_cache";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // governance contract details
    pub gov_chain: u16,
    pub gov_address: Vec<u8>,

    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct SplCacheItem {
    pub name: [u8; 32],
    pub symbol: [u8; 32],
}

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn bridge_contracts(storage: &mut dyn Storage) -> Bucket<Vec<u8>> {
    bucket(storage, BRIDGE_CONTRACTS_KEY)
}

pub fn bridge_contracts_read(storage: &dyn Storage) -> ReadonlyBucket<Vec<u8>> {
    bucket_read(storage, BRIDGE_CONTRACTS_KEY)
}

pub fn wrapped_asset(storage: &mut dyn Storage) -> Bucket<HumanAddr> {
    bucket(storage, WRAPPED_ASSET_KEY)
}

pub fn wrapped_asset_read(storage: &dyn Storage) -> ReadonlyBucket<HumanAddr> {
    bucket_read(storage, WRAPPED_ASSET_KEY)
}

pub fn wrapped_asset_address(storage: &mut dyn Storage) -> Bucket<Vec<u8>> {
    bucket(storage, WRAPPED_ASSET_ADDRESS_KEY)
}

pub fn wrapped_asset_address_read(storage: &dyn Storage) -> ReadonlyBucket<Vec<u8>> {
    bucket_read(storage, WRAPPED_ASSET_ADDRESS_KEY)
}

pub fn spl_cache(storage: &mut dyn Storage) -> Bucket<SplCacheItem> {
    bucket(storage, SPL_CACHE_KEY)
}

pub fn spl_cache_read(storage: &dyn Storage) -> ReadonlyBucket<SplCacheItem> {
    bucket_read(storage, SPL_CACHE_KEY)
}

pub fn token_id_hashes(storage: &mut dyn Storage, chain: u16, address: [u8; 32]) -> Bucket<String> {
    Bucket::multilevel(
        storage,
        &[TOKEN_ID_HASHES_KEY, &chain.to_be_bytes(), &address],
    )
}

pub fn token_id_hashes_read(
    storage: &mut dyn Storage,
    chain: u16,
    address: [u8; 32],
) -> ReadonlyBucket<String> {
    ReadonlyBucket::multilevel(
        storage,
        &[TOKEN_ID_HASHES_KEY, &chain.to_be_bytes(), &address],
    )
}

pub struct Action;

impl Action {
    pub const TRANSFER: u8 = 1;
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

    pub fn serialize(&self) -> Vec<u8> {
        [self.action.to_be_bytes().to_vec(), self.payload.clone()].concat()
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[repr(transparent)]
pub struct BoundedVec<T, const N: usize> {
    vec: Vec<T>,
}

impl<T, const N: usize> BoundedVec<T, N> {
    pub fn new(vec: Vec<T>) -> StdResult<Self> {
        if vec.len() > N {
            return Result::Err(StdError::GenericErr {
                msg: format!("vector length exceeds {}", N),
            });
        };
        Ok(Self { vec })
    }

    #[inline]
    pub fn to_vec(&self) -> Vec<T>
    where
        T: Clone,
    {
        self.vec.clone()
    }

    #[inline]
    pub fn len(&self) -> usize {
        self.vec.len()
    }
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TransferInfo {
    pub nft_address: [u8; 32],
    pub nft_chain: u16,
    pub symbol: [u8; 32],
    pub name: [u8; 32],
    pub token_id: [u8; 32],
    pub uri: BoundedVec<u8, 200>, // max 200 bytes due to Solana
    pub recipient: [u8; 32],
    pub recipient_chain: u16,
}

impl TransferInfo {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let mut offset: usize = 0; // offset into data in bytes
        let nft_address = data.get_const_bytes::<32>(offset);
        offset += 32;
        let nft_chain = data.get_u16(offset);
        offset += 2;
        let symbol = data.get_const_bytes::<32>(offset);
        offset += 32;
        let name = data.get_const_bytes::<32>(offset);
        offset += 32;
        let token_id = data.get_const_bytes::<32>(offset);
        offset += 32;
        let uri_length: usize = data.get_u8(offset).into();
        offset += 1;
        let uri = data.get_bytes(offset, uri_length).to_vec();
        offset += uri_length;
        let recipient = data.get_const_bytes::<32>(offset);
        offset += 32;
        let recipient_chain = data.get_u16(offset);
        offset += 2;

        if data.len() != offset {
            return Result::Err(StdError::GenericErr {
                msg: format!(
                    "Invalid transfer length, expected {}, but got {}",
                    offset,
                    data.len()
                ),
            });
        }

        Ok(TransferInfo {
            nft_address,
            nft_chain,
            symbol,
            name,
            token_id,
            uri: BoundedVec::new(uri.to_vec())?,
            recipient,
            recipient_chain,
        })
    }
    pub fn serialize(&self) -> Vec<u8> {
        [
            self.nft_address.to_vec(),
            self.nft_chain.to_be_bytes().to_vec(),
            self.symbol.to_vec(),
            self.name.to_vec(),
            self.token_id.to_vec(),
            vec![self.uri.len().try_into().unwrap()], // won't panic, because uri.len() is less than 200
            self.uri.to_vec(),
            self.recipient.to_vec(),
            self.recipient_chain.to_be_bytes().to_vec(),
        ]
        .concat()
    }
}

pub struct UpgradeContract {
    pub new_contract: u64,
}

pub struct RegisterChain {
    pub chain_id: u16,
    pub chain_address: Vec<u8>,
}

impl UpgradeContract {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let new_contract = data.get_u64(24);
        Ok(UpgradeContract { new_contract })
    }
}

impl RegisterChain {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let chain_id = data.get_u16(0);
        let chain_address = data[2..].to_vec();

        Ok(RegisterChain {
            chain_id,
            chain_address,
        })
    }
}
