use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

use cosmwasm_std::{Binary, CanonicalAddr, HumanAddr, StdResult, Storage, Coin, Uint128};
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

    // governance contract details
    pub gov_chain: u16,
    pub gov_address: Vec<u8>,

    // Asset locking fee
    pub fee: Coin,
}

// Validator Action Approval(VAA) data
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct ParsedVAA {

    pub version: u8,
    pub guardian_set_index: u32,
    pub timestamp: u64,
    pub nonce: u32,
    pub len_signers: u8,

    pub emitter_chain: u16,
    pub emitter_address: Vec<u8>,
    pub payload: Vec<u8>,

    pub hash: Vec<u8>
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
    0   uint64      timestamp (unix in seconds)
    8   uint32      nonce
    12  uint16      emitter_chain
    14  [32]uint8   emitter_address
    46  []uint8     payload
    */

    pub const HEADER_LEN: usize = 6;
    pub const SIGNATURE_LEN: usize = 66;

    pub const GUARDIAN_SET_INDEX_POS: usize = 1;
    pub const LEN_SIGNER_POS: usize = 5;

    pub const VAA_NONCE_POS: usize = 8;
    pub const VAA_EMITTER_CHAIN_POS: usize = 12;
    pub const VAA_EMITTER_ADDRESS_POS: usize = 14;
    pub const VAA_PAYLOAD_POS: usize = 46;

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

        let timestamp = data.get_u64(body_offset);
        let nonce = data.get_u32(body_offset + Self::VAA_NONCE_POS);
        let emitter_chain = data.get_u16(body_offset + Self::VAA_EMITTER_CHAIN_POS);
        let emitter_address = data.get_bytes32(body_offset + Self::VAA_EMITTER_ADDRESS_POS).to_vec();
        let payload = data[body_offset + Self::VAA_PAYLOAD_POS..].to_vec();

        Ok(ParsedVAA {
            version,
            guardian_set_index,
            timestamp,
            nonce,
            len_signers: len_signers as u8,
            emitter_chain,
            emitter_address,
            payload,
            hash,
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
        // allow quorum of 0 for testing purposes...
        if self.addresses.len() == 0 {
            return 0;
        }
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
    bucket(GUARDIAN_SET_KEY, storage).save(&index.to_be_bytes(), data)
}

pub fn guardian_set_get<S: Storage>(storage: &S, index: u32) -> StdResult<GuardianSetInfo> {
    bucket_read(GUARDIAN_SET_KEY, storage).load(&index.to_be_bytes())
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


pub struct GovernancePacket {
    pub module: Vec<u8>,
    pub chain: u16,
    pub action: u8,
    pub payload: Vec<u8>,
}

impl GovernancePacket {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let module = data.get_bytes32(0).to_vec();
        let chain = data.get_u16(32);
        let action = data.get_u8(34);
        let payload = data[35..].to_vec();

        Ok(GovernancePacket {
            module,
            chain,
            action,
            payload
        })
    }
}

// action 1
pub struct GuardianSetUpgrade {
    pub new_guardian_set_index: u32,
    pub new_guardian_set: GuardianSetInfo,
}

impl GuardianSetUpgrade {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {

        const ADDRESS_LEN: usize = 20;

        let data = data.as_slice();
        let new_guardian_set_index = data.get_u32(0);

        let n_guardians = data.get_u8(4);

        let mut addresses = vec![];

        for i in 0..n_guardians {
            let pos = 5 + (i as usize) * ADDRESS_LEN;
            if pos + ADDRESS_LEN > data.len() {
                return ContractError::InvalidVAA.std_err();
            }

            addresses.push(GuardianAddress {
                bytes: data[pos..pos + ADDRESS_LEN].to_vec().into(),
            });
        }

        let new_guardian_set = GuardianSetInfo {
            addresses,
            expiration_time: 0
        };

        return Ok(
            GuardianSetUpgrade {
                new_guardian_set_index,
                new_guardian_set
            }
        )
    }
}

// action 2
pub struct TransferFee {
    pub amount: Coin,
    pub recipient: CanonicalAddr,
}

impl TransferFee {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let recipient = data.get_address(0);

        let amount = Uint128(data.get_u128_be(32));
        let denom = match String::from_utf8(data[48..].to_vec()) {
            Ok(s) => s,
            Err(_) => return ContractError::InvalidVAA.std_err()
        };
        let amount = Coin {
            denom,
            amount,
        };
        Ok(TransferFee {
            amount,
            recipient
        })
    }
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

    #[test]
    fn test_deserialize() {
        let x = vec![1u8,0u8,0u8,0u8,1u8,0u8,0u8,0u8,0u8,0u8,96u8,180u8,80u8,111u8,0u8,0u8,0u8,1u8,0u8,3u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,0u8,120u8,73u8,153u8,19u8,90u8,170u8,138u8,60u8,165u8,145u8,68u8,104u8,133u8,47u8,221u8,219u8,221u8,216u8,120u8,157u8,0u8,91u8,48u8,44u8,48u8,44u8,51u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,53u8,54u8,44u8,50u8,51u8,51u8,44u8,49u8,44u8,49u8,49u8,49u8,44u8,49u8,54u8,55u8,44u8,49u8,57u8,48u8,44u8,50u8,48u8,51u8,44u8,49u8,54u8,44u8,49u8,55u8,54u8,44u8,50u8,49u8,56u8,44u8,50u8,53u8,49u8,44u8,49u8,51u8,49u8,44u8,51u8,57u8,44u8,49u8,54u8,44u8,49u8,57u8,53u8,44u8,50u8,50u8,55u8,44u8,49u8,52u8,57u8,44u8,50u8,51u8,54u8,44u8,49u8,57u8,48u8,44u8,50u8,49u8,50u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,51u8,44u8,50u8,51u8,50u8,44u8,48u8,44u8,51u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,48u8,44u8,53u8,51u8,44u8,49u8,49u8,54u8,44u8,52u8,56u8,44u8,49u8,49u8,54u8,44u8,49u8,52u8,57u8,44u8,49u8,48u8,56u8,44u8,49u8,49u8,51u8,44u8,56u8,44u8,48u8,44u8,50u8,51u8,50u8,44u8,52u8,57u8,44u8,49u8,53u8,50u8,44u8,49u8,44u8,50u8,56u8,44u8,50u8,48u8,51u8,44u8,50u8,49u8,50u8,44u8,50u8,50u8,49u8,44u8,50u8,52u8,49u8,44u8,56u8,53u8,44u8,49u8,48u8,57u8,93u8];
        ParsedVAA::deserialize(x.as_slice()).unwrap();
    }
}
