use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use sha3::{Digest, Keccak256};
use std::str;

use cosmwasm_schema::cw_serde;
use cosmwasm_std::{Binary, StdResult, Storage};
use cosmwasm_storage::{
    bucket, bucket_read, singleton, singleton_read, Bucket, ReadonlyBucket, ReadonlySingleton,
    Singleton,
};

use cw_wormhole::{
    byte_utils::{get_string_from_32, ByteUtils},
    ContractError,
};

use cw_token_bridge::msg::TransferInfoResponse as TokenBridgeTransferInfoResponse;

type HumanAddr = String;
static CONFIG_KEY: &[u8] = b"config";
static CHAIN_CHANNELS: &[u8] = b"chain_channels";
static CURRENT_TRANSFER_KEY: &[u8] = b"current_transfer_tmp";

// pub const CHAIN_CHANNELS: Map<u16, String> = Map::new("chain_channels");

/// Information about this contract's general parameters.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct ConfigInfo {
    /// Address of the core bridge contract
    pub wormhole_contract: HumanAddr,

    /// Address of the token bridge contract
    pub token_bridge_contract: HumanAddr,

    /// The wormhole id of the current chain.
    pub chain_id: u16,
}
// Validator Action Approval(VAA) data
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct ParsedVAA {
    pub version: u8,
    pub guardian_set_index: u32,
    pub timestamp: u32,
    pub nonce: u32,
    pub len_signers: u8,

    pub emitter_chain: u16,
    pub emitter_address: Vec<u8>,
    pub sequence: u64,
    pub consistency_level: u8,
    pub payload: Vec<u8>,

    pub hash: Vec<u8>,
}

impl ParsedVAA {
    /* VAA format:

    header (length 6):
    0   uint8   version (0x01)
    1   uint32  guardian set index
    5   uint8   len signatures

    per signature (length 66):
    0   uint8       index of the signer (in guardian keys)
    1   [65]uint8   signature

    body:
    0   uint32      timestamp (unix in seconds)
    4   uint32      nonce
    8   uint16      emitter_chain
    10  [32]uint8   emitter_address
    42  uint64      sequence
    50  uint8       consistency_level
    51  []uint8     payload
    */

    pub const HEADER_LEN: usize = 6;
    pub const SIGNATURE_LEN: usize = 66;

    pub const GUARDIAN_SET_INDEX_POS: usize = 1;
    pub const LEN_SIGNER_POS: usize = 5;

    pub const VAA_NONCE_POS: usize = 4;
    pub const VAA_EMITTER_CHAIN_POS: usize = 8;
    pub const VAA_EMITTER_ADDRESS_POS: usize = 10;
    pub const VAA_SEQUENCE_POS: usize = 42;
    pub const VAA_CONSISTENCY_LEVEL_POS: usize = 50;
    pub const VAA_PAYLOAD_POS: usize = 51;

    // Signature data offsets in the signature block
    pub const SIG_DATA_POS: usize = 1;
    // Signature length minus recovery id at the end
    pub const SIG_DATA_LEN: usize = 64;
    // Recovery byte is last after the main signature
    pub const SIG_RECOVERY_POS: usize = Self::SIG_DATA_POS + Self::SIG_DATA_LEN;

    pub fn deserialize(data: &[u8]) -> StdResult<Self> {
        let version = data.get_u8(0);

        // Load 4 bytes starting from index 1
        let guardian_set_index: u32 = data.get_u32(Self::GUARDIAN_SET_INDEX_POS);
        let len_signers = data.get_u8(Self::LEN_SIGNER_POS) as usize;
        let body_offset: usize = Self::HEADER_LEN + Self::SIGNATURE_LEN * len_signers;

        // Hash the body
        if body_offset >= data.len() {
            return ContractError::InvalidVAA.std_err();
        }
        let body = &data[body_offset..];
        let mut hasher = Keccak256::new();
        hasher.update(body);
        let hash = hasher.finalize().to_vec();

        // Rehash the hash
        let mut hasher = Keccak256::new();
        hasher.update(hash);
        let hash = hasher.finalize().to_vec();

        // Signatures valid, apply VAA
        if body_offset + Self::VAA_PAYLOAD_POS > data.len() {
            return ContractError::InvalidVAA.std_err();
        }

        let timestamp = data.get_u32(body_offset);
        let nonce = data.get_u32(body_offset + Self::VAA_NONCE_POS);
        let emitter_chain = data.get_u16(body_offset + Self::VAA_EMITTER_CHAIN_POS);
        let emitter_address = data
            .get_bytes32(body_offset + Self::VAA_EMITTER_ADDRESS_POS)
            .to_vec();
        let sequence = data.get_u64(body_offset + Self::VAA_SEQUENCE_POS);
        let consistency_level = data.get_u8(body_offset + Self::VAA_CONSISTENCY_LEVEL_POS);
        let payload = data[body_offset + Self::VAA_PAYLOAD_POS..].to_vec();

        Ok(ParsedVAA {
            version,
            guardian_set_index,
            timestamp,
            nonce,
            len_signers: len_signers as u8,
            emitter_chain,
            emitter_address,
            sequence,
            consistency_level,
            payload,
            hash,
        })
    }
}

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn chain_channels(storage: &mut dyn Storage) -> Bucket<String> {
    bucket(storage, CHAIN_CHANNELS)
}

pub fn chain_channels_read(storage: &dyn Storage) -> ReadonlyBucket<String> {
    bucket_read(storage, CHAIN_CHANNELS)
}

pub struct UpgradeContract {
    pub new_contract: u64,
}

impl UpgradeContract {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let new_contract = data.get_u64(24);
        Ok(UpgradeContract { new_contract })
    }
}

pub struct RegisterChainChannel {
    pub chain_id: u16,
    pub channel_id: String,
}

impl RegisterChainChannel {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let chain_id = data.get_u16(0);
        // Note that get_string_from_32 actually handles longer strings.
        let channel_id = get_string_from_32(&data[2..]);

        Ok(RegisterChainChannel {
            chain_id,
            channel_id,
        })
    }
}

#[cw_serde]
pub enum TransferPayload {
    BasicTransfer { chain_id: u16, recipient: Binary },
    BasicDeposit { amount: u64 },
}

/// Structure to keep track of the current transfer. Required to pass state through to the reply handler for submessages during a transfer.
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, Eq, JsonSchema)]
pub struct TransferState {
    pub transfer_info: TokenBridgeTransferInfoResponse,
    pub target_chain_id: u16,
    pub target_channel_id: String,
    pub target_recipient: Binary,
}

pub fn current_transfer(storage: &mut dyn Storage) -> Singleton<TransferState> {
    singleton(storage, CURRENT_TRANSFER_KEY)
}
