use schemars::{
    JsonSchema,
};
use serde::{
    Deserialize,
    Serialize,
};

use cosmwasm_std::{
    Binary,
    CanonicalAddr,
    Coin,
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

use crate::{
    byte_utils::ByteUtils,
    error::ContractError,
};

use sha3::{
    Digest,
    Keccak256,
};

type HumanAddr = String;

pub static CONFIG_KEY: &[u8] = b"config";
pub static GUARDIAN_SET_KEY: &[u8] = b"guardian_set";
pub static SEQUENCE_KEY: &[u8] = b"sequence";
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

    // Message sending fee
    pub fee: Coin,
}

// Validator Action Approval(VAA) data
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
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
        let body_offset: usize = Self::HEADER_LEN + Self::SIGNATURE_LEN * len_signers as usize;

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

// Guardian address
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct GuardianAddress {
    pub bytes: Binary, // 20-byte addresses
}

use crate::contract::FEE_DENOMINATION;
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
    pub addresses: Vec<GuardianAddress>,
    // List of guardian addresses
    pub expiration_time: u64, // Guardian set expiration time
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

pub fn config(storage: &mut dyn Storage) -> Singleton<ConfigInfo> {
    singleton(storage, CONFIG_KEY)
}

pub fn config_read(storage: &dyn Storage) -> ReadonlySingleton<ConfigInfo> {
    singleton_read(storage, CONFIG_KEY)
}

pub fn guardian_set_set(
    storage: &mut dyn Storage,
    index: u32,
    data: &GuardianSetInfo,
) -> StdResult<()> {
    bucket(storage, GUARDIAN_SET_KEY).save(&index.to_be_bytes(), data)
}

pub fn guardian_set_get(storage: &dyn Storage, index: u32) -> StdResult<GuardianSetInfo> {
    bucket_read(storage, GUARDIAN_SET_KEY).load(&index.to_be_bytes())
}

pub fn sequence_set(storage: &mut dyn Storage, emitter: &[u8], sequence: u64) -> StdResult<()> {
    bucket(storage, SEQUENCE_KEY).save(emitter, &sequence)
}

pub fn sequence_read(storage: &dyn Storage, emitter: &[u8]) -> u64 {
    bucket_read(storage, SEQUENCE_KEY)
        .load(&emitter)
        .or::<u64>(Ok(0))
        .unwrap()
}

pub fn vaa_archive_add(storage: &mut dyn Storage, hash: &[u8]) -> StdResult<()> {
    bucket(storage, GUARDIAN_SET_KEY).save(hash, &true)
}

pub fn vaa_archive_check(storage: &dyn Storage, hash: &[u8]) -> bool {
    bucket_read(storage, GUARDIAN_SET_KEY)
        .load(&hash)
        .or::<bool>(Ok(false))
        .unwrap()
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

pub struct GovernancePacket {
    pub module: Vec<u8>,
    pub action: u8,
    pub chain: u16,
    pub payload: Vec<u8>,
}

impl GovernancePacket {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let module = data.get_bytes32(0).to_vec();
        let action = data.get_u8(32);
        let chain = data.get_u16(33);
        let payload = data[35..].to_vec();

        Ok(GovernancePacket {
            module,
            action,
            chain,
            payload,
        })
    }
}

// action 1
pub struct ContractUpgrade  {
    pub new_contract: u64,
}

// action 2
pub struct GuardianSetUpgrade {
    pub new_guardian_set_index: u32,
    pub new_guardian_set: GuardianSetInfo,
}

impl ContractUpgrade {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let new_contract = data.get_u64(24);
        Ok(ContractUpgrade {
            new_contract,
        })
    }
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
            expiration_time: 0,
        };

        return Ok(GuardianSetUpgrade {
            new_guardian_set_index,
            new_guardian_set,
        });
    }
}

// action 3
pub struct SetFee {
    pub fee: Coin,
}

impl SetFee {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();

        let (_, amount) = data.get_u256(0);
        let fee = Coin {
            denom: String::from(FEE_DENOMINATION),
            amount: Uint128::new(amount),
        };
        Ok(SetFee { fee })
    }
}

// action 4
pub struct TransferFee {
    pub amount: Coin,
    pub recipient: CanonicalAddr,
}

impl TransferFee {
    pub fn deserialize(data: &Vec<u8>) -> StdResult<Self> {
        let data = data.as_slice();
        let recipient = data.get_address(0);

        let (_, amount) = data.get_u256(32);
        let amount = Coin {
            denom: String::from(FEE_DENOMINATION),
            amount: Uint128::new(amount),
        };
        Ok(TransferFee { amount, recipient })
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
        let x = hex::decode("080000000901007bfa71192f886ab6819fa4862e34b4d178962958d9b2e3d9437338c9e5fde1443b809d2886eaa69e0f0158ea517675d96243c9209c3fe1d94d5b19866654c6980000000b150000000500020001020304000000000000000000000000000000000000000000000000000000000000000000000a0261626364").unwrap();
        let v = ParsedVAA::deserialize(x.as_slice()).unwrap();
        assert_eq!(
            v,
            ParsedVAA {
                version: 8,
                guardian_set_index: 9,
                timestamp: 2837,
                nonce: 5,
                len_signers: 1,
                emitter_chain: 2,
                emitter_address: vec![
                    0, 1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
                    0, 0, 0, 0, 0, 0
                ],
                sequence: 10,
                consistency_level: 2,
                payload: vec![97, 98, 99, 100],
                hash: vec![
                    195, 10, 19, 96, 8, 61, 218, 69, 160, 238, 165, 142, 105, 119, 139, 121, 212,
                    73, 238, 179, 13, 80, 245, 224, 75, 110, 163, 8, 185, 132, 55, 34
                ]
            }
        );
    }
}
