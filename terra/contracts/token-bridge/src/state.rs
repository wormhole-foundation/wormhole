use schemars::JsonSchema;
use serde::{
    Deserialize,
    Serialize,
};

use cosmwasm_std::{
    CanonicalAddr,
    StdError,
    StdResult,
    Storage,
    Uint128,
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
pub static TRANSFER_TMP_KEY: &[u8] = b"transfer_tmp";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";
pub static WRAPPED_ASSET_SEQ_KEY: &[u8] = b"wrapped_seq_asset";
pub static WRAPPED_ASSET_ADDRESS_KEY: &[u8] = b"wrapped_asset_address";
pub static BRIDGE_CONTRACTS: &[u8] = b"bridge_contracts";
pub static BRIDGE_DEPOSITS: &[u8] = b"bridge_deposits";
pub static NATIVE_COUNTER: &[u8] = b"native_counter";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // governance contract details
    pub gov_chain: u16,
    pub gov_address: Vec<u8>,

    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64,
}

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn bridge_deposit(storage: &mut dyn Storage) -> Bucket<Uint128> {
    bucket(storage, BRIDGE_DEPOSITS)
}

pub fn bridge_deposit_read(storage: &dyn Storage) -> ReadonlyBucket<Uint128> {
    bucket_read(storage, BRIDGE_DEPOSITS)
}

pub fn bridge_contracts(storage: &mut dyn Storage) -> Bucket<Vec<u8>> {
    bucket(storage, BRIDGE_CONTRACTS)
}

pub fn bridge_contracts_read(storage: &dyn Storage) -> ReadonlyBucket<Vec<u8>> {
    bucket_read(storage, BRIDGE_CONTRACTS)
}

pub fn wrapped_asset(storage: &mut dyn Storage) -> Bucket<HumanAddr> {
    bucket(storage, WRAPPED_ASSET_KEY)
}

pub fn wrapped_asset_read(storage: &dyn Storage) -> ReadonlyBucket<HumanAddr> {
    bucket_read(storage, WRAPPED_ASSET_KEY)
}

pub fn wrapped_asset_seq(storage: &mut dyn Storage) -> Bucket<u64> {
    bucket(storage, WRAPPED_ASSET_SEQ_KEY)
}

pub fn wrapped_asset_seq_read(storage: &mut dyn Storage) -> ReadonlyBucket<u64> {
    bucket_read(storage, WRAPPED_ASSET_SEQ_KEY)
}

pub fn wrapped_asset_address(storage: &mut dyn Storage) -> Bucket<Vec<u8>> {
    bucket(storage, WRAPPED_ASSET_ADDRESS_KEY)
}

pub fn wrapped_asset_address_read(storage: &dyn Storage) -> ReadonlyBucket<Vec<u8>> {
    bucket_read(storage, WRAPPED_ASSET_ADDRESS_KEY)
}

type Serialized128 = String;

/// Structure to keep track of an active CW20 transfer, required to pass state through to the reply
/// handler for submessages during a transfer.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TransferState {
    pub account: String,
    pub message: Vec<u8>,
    pub multiplier: Serialized128,
    pub nonce: u32,
    pub previous_balance: Serialized128,
    pub token_address: HumanAddr,
    pub token_canonical: CanonicalAddr,
}

pub fn wrapped_transfer_tmp(storage: &mut dyn Storage) -> Singleton<TransferState> {
    singleton(storage, TRANSFER_TMP_KEY)
}

pub fn send_native(
    storage: &mut dyn Storage,
    asset_address: &CanonicalAddr,
    amount: Uint128,
) -> StdResult<()> {
    let mut counter_bucket = bucket(storage, NATIVE_COUNTER);
    let new_total = amount
        + counter_bucket
            .load(asset_address.as_slice())
            .unwrap_or(Uint128::zero());
    if new_total > Uint128::new(u64::MAX as u128) {
        return Err(StdError::generic_err(
            "transfer exceeds max outstanding bridged token amount",
        ));
    }
    counter_bucket.save(asset_address.as_slice(), &new_total)
}

pub fn receive_native(
    storage: &mut dyn Storage,
    asset_address: &CanonicalAddr,
    amount: Uint128,
) -> StdResult<()> {
    let mut counter_bucket = bucket(storage, NATIVE_COUNTER);
    let total: Uint128 = counter_bucket.load(asset_address.as_slice())?;
    let result = total.checked_sub(amount)?;
    counter_bucket.save(asset_address.as_slice(), &result)
}

pub struct Action;

impl Action {
    pub const TRANSFER: u8 = 1;
    pub const ATTEST_META: u8 = 2;
    pub const TRANSFER_WITH_PAYLOAD: u8 = 3;
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

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TransferInfo {
    pub amount: (u128, u128),
    pub token_address: Vec<u8>,
    pub token_chain: u16,
    pub recipient: Vec<u8>,
    pub recipient_chain: u16,
    pub fee: (u128, u128),
}

impl TransferInfo {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let amount = data.get_u256(0);
        let token_address = data.get_bytes32(32).to_vec();
        let token_chain = data.get_u16(64);
        let recipient = data.get_bytes32(66).to_vec();
        let recipient_chain = data.get_u16(98);
        let fee = data.get_u256(100);

        Ok(TransferInfo {
            amount,
            token_address,
            token_chain,
            recipient,
            recipient_chain,
            fee,
        })
    }
    pub fn serialize(&self) -> Vec<u8> {
        [
            self.amount.0.to_be_bytes().to_vec(),
            self.amount.1.to_be_bytes().to_vec(),
            self.token_address.clone(),
            self.token_chain.to_be_bytes().to_vec(),
            self.recipient.to_vec(),
            self.recipient_chain.to_be_bytes().to_vec(),
            self.fee.0.to_be_bytes().to_vec(),
            self.fee.1.to_be_bytes().to_vec(),
        ]
        .concat()
    }
}

//     0   u256     amount
//     32  [u8; 32] token_address
//     64  u16      token_chain
//     66  [u8; 32] recipient
//     98  u16      recipient_chain
//     100 u256     fee
//     132 [u8]     payload

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct TransferWithPayloadInfo {
    pub transfer_info: TransferInfo,
    pub payload: Vec<u8>,
}

impl TransferWithPayloadInfo {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let transfer_info = TransferInfo::deserialize(data)?;
        let payload = TransferWithPayloadInfo::get_payload(data);

        Ok(TransferWithPayloadInfo {
            transfer_info,
            payload,
        })
    }
    pub fn serialize(&self) -> Vec<u8> {
        [self.transfer_info.serialize(), self.payload.clone()].concat()
    }
    pub fn get_payload(data: &Vec<u8>) -> Vec<u8> {
        return data[132..].to_vec();
    }
}

// 0  [32]uint8  TokenAddress
// 32 uint16     TokenChain
// 34 uint8      Decimals
// 35 [32]uint8  Symbol
// 67 [32]uint8  Name

pub struct AssetMeta {
    pub token_address: Vec<u8>,
    pub token_chain: u16,
    pub decimals: u8,
    pub symbol: Vec<u8>,
    pub name: Vec<u8>,
}

impl AssetMeta {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let token_address = data.get_bytes32(0).to_vec();
        let token_chain = data.get_u16(32);
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
            self.token_address.clone(),
            self.token_chain.to_be_bytes().to_vec(),
            self.decimals.to_be_bytes().to_vec(),
            self.symbol.clone(),
            self.name.clone(),
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
