use dep::aztec::protocol_types::traits::{Deserialize, Packable, Serialize};
use std::meta::derive;

#[derive(Deserialize, Packable, Serialize)]
pub struct Provider {
    chain_id: u16,
    governance_chain_id: u16,
    governance_contract: Field // bytes
}

#[derive(Deserialize, Packable, Serialize)]
pub struct GuardianSet {
    keys: Field, // In Noir, we need to use fixed-size arrays - assuming max 10 keys
    expiration_time: u32
}

pub struct Signature {
    r: [u8; 32],
    s: [u8; 32],
    v: u8,
    guardian_index: u8
}

pub struct VAA {
    version: u8,
    timestamp: u32,
    nonce: u32,
    emitter_chain_id: u16,
    emitter_address: [u8; 32],
    sequence: u64,
    consistency_level: u8,
    payload: [u8; 1024], // Fixed size for byte arrays in Noir, adjust as needed
    guardian_set_index: u32,
    signatures: [Signature; 13], // Assuming max 13 signatures
    hash: [u8; 32]
}

pub struct ContractUpgrade {
    module: [u8; 32],
    action: u8,
    chain: u16,
    new_contract: Field // Addresses in Noir are typically represented as Field elements
}

pub struct GuardianSetUpgrade {
    module: [u8; 32],
    action: u8,
    chain: u16,
    new_guardian_set: GuardianSet,
    new_guardian_set_index: u32
}

pub struct SetMessageFee {
    module: [u8; 32],
    action: u8,
    chain: u16,
    message_fee: Field
}

pub struct TransferFees {
    module: [u8; 32],
    action: u8,
    chain: u16,
    amount: Field,
    recipient: [u8; 32]
}

pub struct RecoverChainId {
    module: [u8; 32],
    action: u8,
    evm_chain_id: Field,
    new_chain_id: u16
}
