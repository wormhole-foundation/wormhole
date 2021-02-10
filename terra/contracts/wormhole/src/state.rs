use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{Binary, CanonicalAddr, HumanAddr, StdResult, Storage};
use cosmwasm_storage::{
    bucket, bucket_read, singleton, singleton_read, Bucket, ReadonlyBucket, ReadonlySingleton,
    Singleton,
};

use crate::byte_utils::ByteUtils;
use crate::error::ContractError;

use sha3::{Digest, Keccak256};

pub static CONFIG_KEY: &[u8] = b"config";
pub static GUARDIAN_SET_KEY: &[u8] = b"guardian_set";
pub static WRAPPED_ASSET_KEY: &[u8] = b"wrapped_asset";
pub static WRAPPED_ASSET_ADDRESS_KEY: &[u8] = b"wrapped_asset_address";

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ConfigInfo {
    // Current active guardian set
    pub guardian_set_index: u32,

    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,

    // Code id for wrapped asset contract
    pub wrapped_asset_code_id: u64,

    // Contract owner address, it can make contract active/inactive
    pub owner: CanonicalAddr,

    // If true the contract is active and functioning
    pub is_active: bool,
}

// Validator Action Approval(VAA) data
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ParsedVAA {
    pub version: u8,
    pub guardian_set_index: u32,
    pub len_signers: usize,
    pub hash: Vec<u8>,
    pub action: u8,
    pub payload: Vec<u8>,
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
    0   uint32  unix seconds
    4   uint8   action
    5   [payload_size]uint8 payload */

    pub const HEADER_LEN: usize = 6;
    pub const SIGNATURE_LEN: usize = 66;

    pub const GUARDIAN_SET_INDEX_POS: usize = 1;
    pub const LEN_SIGNER_POS: usize = 5;

    pub const VAA_ACTION_POS: usize = 4;
    pub const VAA_PAYLOAD_POS: usize = 5;

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
        let body_offset: usize = Self::HEADER_LEN + Self::SIGNATURE_LEN * len_signers as usize;

        // Hash the body
        if body_offset >= data.len() {
            return ContractError::InvalidVAA.std_err();
        }
        let body = &data[body_offset..];
        let mut hasher = Keccak256::new();
        hasher.update(body);
        let hash = hasher.finalize().to_vec();

        // Signatures valid, apply VAA
        if body_offset + Self::VAA_PAYLOAD_POS > data.len() {
            return ContractError::InvalidVAA.std_err();
        }
        let action = data.get_u8(body_offset + Self::VAA_ACTION_POS);
        let payload = &data[body_offset + Self::VAA_PAYLOAD_POS..];

        Ok(ParsedVAA {
            version,
            guardian_set_index,
            len_signers,
            hash,
            action,
            payload: payload.to_vec(),
        })
    }
}

// Guardian address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianAddress {
    pub bytes: Binary, // 20-byte addresses
}

#[cfg(test)]
use hex;
#[cfg(test)]
impl GuardianAddress {
    pub fn from(string: &str) -> GuardianAddress {
        GuardianAddress {
            bytes: hex::decode(string).expect("Decoding failed").into(),
        }
    }
}

// Guardian set information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianSetInfo {
    pub addresses: Vec<GuardianAddress>, // List of guardian addresses
    pub expiration_time: u64,            // Guardian set expiration time
}

impl GuardianSetInfo {
    pub fn quorum(&self) -> usize {
        ((self.addresses.len() * 10 / 3) * 2) / 10 + 1
    }
}

// Wormhole contract generic information
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct WormholeInfo {
    // Period for which a guardian set stays active after it has been replaced
    pub guardian_set_expirity: u64,
}

pub fn config<S: Storage>(storage: &mut S) -> Singleton<S, ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read<S: Storage>(storage: &S) -> ReadonlySingleton<S, ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn guardian_set_set<S: Storage>(
    storage: &mut S,
    index: u32,
    data: &GuardianSetInfo,
) -> StdResult<()> {
    bucket(GUARDIAN_SET_KEY, storage).save(&index.to_le_bytes(), data)
}

pub fn guardian_set_get<S: Storage>(storage: &S, index: u32) -> StdResult<GuardianSetInfo> {
    bucket_read(GUARDIAN_SET_KEY, storage).load(&index.to_le_bytes())
}

pub fn vaa_archive_add<S: Storage>(storage: &mut S, hash: &[u8]) -> StdResult<()> {
    bucket(GUARDIAN_SET_KEY, storage).save(hash, &true)
}

pub fn vaa_archive_check<S: Storage>(storage: &S, hash: &[u8]) -> bool {
    bucket_read(GUARDIAN_SET_KEY, storage)
        .load(&hash)
        .or::<bool>(Ok(false))
        .unwrap()
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

#[cfg(test)]
mod tests {
    use super::*;

    fn build_guardian_set(length: usize) -> GuardianSetInfo {
        let mut addresses: Vec<GuardianAddress> = Vec::with_capacity(length);
        for _ in 0..length {
            addresses.push(GuardianAddress {
                bytes: vec![].into(),
            });
        }

        GuardianSetInfo {
            addresses,
            expiration_time: 0,
        }
    }

    #[test]
    fn quardian_set_quorum() {
        assert_eq!(build_guardian_set(1).quorum(), 1);
        assert_eq!(build_guardian_set(2).quorum(), 2);
        assert_eq!(build_guardian_set(3).quorum(), 3);
        assert_eq!(build_guardian_set(4).quorum(), 3);
        assert_eq!(build_guardian_set(5).quorum(), 4);
        assert_eq!(build_guardian_set(6).quorum(), 5);
        assert_eq!(build_guardian_set(7).quorum(), 5);
        assert_eq!(build_guardian_set(8).quorum(), 6);
        assert_eq!(build_guardian_set(9).quorum(), 7);
        assert_eq!(build_guardian_set(10).quorum(), 7);
        assert_eq!(build_guardian_set(11).quorum(), 8);
        assert_eq!(build_guardian_set(12).quorum(), 9);
        assert_eq!(build_guardian_set(20).quorum(), 14);
        assert_eq!(build_guardian_set(25).quorum(), 17);
        assert_eq!(build_guardian_set(100).quorum(), 67);
    }
}
