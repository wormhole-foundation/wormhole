use solana_program::{
    instruction::Instruction,
    pubkey::Pubkey,
};
use std::str::FromStr;

use crate::vaa::{
    DeserializePayload,
    SignatureItem,
    VAA,
};
use borsh::BorshDeserialize;
use byteorder::WriteBytesExt;
use sha3::Digest;
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use std::io::Write;

use crate::{
    accounts::{
        Bridge,
        BridgeData,
        FeeCollector,
        GuardianSet,
        GuardianSetData,
        GuardianSetDerivationData,
        PostedVAA,
        PostedVAAData,
        PostedVAADerivationData,
    },
    instructions::{
        hash_vaa,
        post_message,
        post_message_unreliable,
        post_vaa,
        set_fees,
        transfer_fees,
        upgrade_contract,
        upgrade_guardian_set,
        verify_signatures,
    },
    types::{
        ConsistencyLevel,
        GovernancePayloadGuardianSetChange,
        GovernancePayloadTransferFees,
        GovernancePayloadUpgrade,
    },
    Claim,
    ClaimDerivationData,
    PostVAAData,
    VerifySignaturesData,
};
use byteorder::LittleEndian;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub fn post_message_ix(
    program_id: String,
    payer: String,
    emitter: String,
    message: String,
    nonce: u32,
    msg: Vec<u8>,
    consistency: String,
) -> JsValue {
    let consistency_level = match consistency.as_str() {
        "CONFIRMED" => ConsistencyLevel::Confirmed,
        "FINALIZED" => ConsistencyLevel::Finalized,
        _ => panic!("invalid consistency level"),
    };
    let ix = post_message(
        Pubkey::from_str(program_id.as_str()).unwrap(),
        Pubkey::from_str(payer.as_str()).unwrap(),
        Pubkey::from_str(emitter.as_str()).unwrap(),
        Pubkey::from_str(message.as_str()).unwrap(),
        nonce,
        msg,
        consistency_level,
    )
    .unwrap();
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn post_message_unreliable_ix(
    program_id: String,
    payer: String,
    emitter: String,
    message: String,
    nonce: u32,
    msg: Vec<u8>,
    consistency: String,
) -> JsValue {
    let consistency_level = match consistency.as_str() {
        "CONFIRMED" => ConsistencyLevel::Confirmed,
        "FINALIZED" => ConsistencyLevel::Finalized,
        _ => panic!("invalid consistency level"),
    };
    let ix = post_message_unreliable(
        Pubkey::from_str(program_id.as_str()).unwrap(),
        Pubkey::from_str(payer.as_str()).unwrap(),
        Pubkey::from_str(emitter.as_str()).unwrap(),
        Pubkey::from_str(message.as_str()).unwrap(),
        nonce,
        msg,
        consistency_level,
    )
    .unwrap();
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn post_vaa_ix(
    program_id: String,
    payer: String,
    signature_set: String,
    vaa: Vec<u8>,
) -> JsValue {
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let vaa = PostVAAData {
        version: vaa.version,
        guardian_set_index: vaa.guardian_set_index,
        timestamp: vaa.timestamp,
        nonce: vaa.nonce,
        emitter_chain: vaa.emitter_chain,
        emitter_address: vaa.emitter_address,
        sequence: vaa.sequence,
        consistency_level: vaa.consistency_level,
        payload: vaa.payload,
    };
    let ix = post_vaa(
        Pubkey::from_str(program_id.as_str()).unwrap(),
        Pubkey::from_str(payer.as_str()).unwrap(),
        Pubkey::from_str(signature_set.as_str()).unwrap(),
        vaa,
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn update_guardian_set_ix(program_id: String, payer: String, vaa: Vec<u8>) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload =
        GovernancePayloadGuardianSetChange::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &program_id,
    );
    let ix = upgrade_guardian_set(
        program_id,
        Pubkey::from_str(payer.as_str()).unwrap(),
        message_key,
        Pubkey::new(&vaa.emitter_address),
        payload.new_guardian_set_index - 1,
        payload.new_guardian_set_index,
        vaa.sequence,
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn set_fees_ix(program_id: String, payer: String, vaa: Vec<u8>) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let message_key = PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &program_id,
    );
    let ix = set_fees(
        program_id,
        Pubkey::from_str(payer.as_str()).unwrap(),
        message_key,
        Pubkey::new(&vaa.emitter_address),
        vaa.sequence,
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn transfer_fees_ix(program_id: String, payer: String, vaa: Vec<u8>) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = GovernancePayloadTransferFees::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &program_id,
    );
    let ix = transfer_fees(
        program_id,
        Pubkey::from_str(payer.as_str()).unwrap(),
        message_key,
        Pubkey::new(&vaa.emitter_address),
        vaa.sequence,
        Pubkey::new(&payload.to[..]),
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn upgrade_contract_ix(
    program_id: String,
    payer: String,
    spill: String,
    vaa: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let spill = Pubkey::from_str(spill.as_str()).unwrap();
    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let payload = GovernancePayloadUpgrade::deserialize(&mut vaa.payload.as_slice()).unwrap();
    let message_key = PostedVAA::<'_, { AccountState::Uninitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa.clone().into()).to_vec(),
        },
        &program_id,
    );
    let ix = upgrade_contract(
        program_id,
        Pubkey::from_str(payer.as_str()).unwrap(),
        message_key,
        Pubkey::new(&vaa.emitter_address),
        payload.new_contract,
        spill,
        vaa.sequence,
    );
    return JsValue::from_serde(&ix).unwrap();
}

#[wasm_bindgen]
pub fn verify_signatures_ix(
    program_id: String,
    payer: String,
    guardian_set_index: u32,
    guardian_set: JsValue,
    signature_set: String,
    vaa_data: Vec<u8>,
) -> JsValue {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let payer = Pubkey::from_str(payer.as_str()).unwrap();
    let signature_set = Pubkey::from_str(signature_set.as_str()).unwrap();

    let guardian_set: GuardianSetData = guardian_set.into_serde().unwrap();
    let vaa = VAA::deserialize(vaa_data.as_slice()).unwrap();

    // Map signatures to guardian set
    let mut signature_items: Vec<SignatureItem> = Vec::new();
    for s in vaa.signatures.iter() {
        let mut item = SignatureItem {
            signature: s.signature.clone(),
            key: [0; 20],
            index: s.guardian_index as u8,
        };
        item.key = guardian_set.keys[s.guardian_index as usize];

        signature_items.push(item);
    }

    let vaa_body = &vaa_data[VAA::HEADER_LEN + VAA::SIGNATURE_LEN * vaa.signatures.len()..];
    let body_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write(vaa_body).unwrap();
        h.finalize().into()
    };

    let mut verify_txs: Vec<Vec<Instruction>> = Vec::new();
    for (_tx_index, chunk) in signature_items.chunks(7).enumerate() {
        let mut secp_payload = Vec::new();
        let mut signature_status = [-1i8; 19];

        let data_offset = 1 + chunk.len() * 11;
        let message_offset = data_offset + chunk.len() * 85;

        // 1 number of signatures
        secp_payload.write_u8(chunk.len() as u8).unwrap();

        // Secp signature info description (11 bytes * n)
        for (i, s) in chunk.iter().enumerate() {
            secp_payload
                .write_u16::<LittleEndian>((data_offset + 85 * i) as u16)
                .unwrap();
            secp_payload.write_u8(0).unwrap();
            secp_payload
                .write_u16::<LittleEndian>((data_offset + 85 * i + 65) as u16)
                .unwrap();
            secp_payload.write_u8(0).unwrap();
            secp_payload
                .write_u16::<LittleEndian>(message_offset as u16)
                .unwrap();
            secp_payload
                .write_u16::<LittleEndian>(body_hash.len() as u16)
                .unwrap();
            secp_payload.write_u8(0).unwrap();
            signature_status[s.index as usize] = i as i8;
        }

        // Write signatures and addresses
        for s in chunk.iter() {
            secp_payload.write(&s.signature).unwrap();
            secp_payload.write(&s.key).unwrap();
        }

        // Write body
        secp_payload.write(&body_hash).unwrap();

        let secp_ix = Instruction {
            program_id: solana_program::secp256k1_program::id(),
            data: secp_payload,
            accounts: vec![],
        };

        let payload = VerifySignaturesData {
            signers: signature_status,
        };

        let verify_ix = match verify_signatures(
            program_id,
            payer,
            guardian_set_index,
            signature_set,
            payload,
        ) {
            Ok(v) => v,
            Err(e) => panic!("{:?}", e),
        };

        verify_txs.push(vec![secp_ix, verify_ix])
    }

    JsValue::from_serde(&verify_txs).unwrap()
}

#[wasm_bindgen]
pub fn guardian_set_address(bridge: String, index: u32) -> Vec<u8> {
    let program_id = Pubkey::from_str(bridge.as_str()).unwrap();
    let guardian_key = GuardianSet::<'_, { AccountState::Initialized }>::key(
        &GuardianSetDerivationData { index: index },
        &program_id,
    );

    guardian_key.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_guardian_set(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&GuardianSetData::try_from_slice(data.as_slice()).unwrap()).unwrap()
}

#[wasm_bindgen]
pub fn state_address(bridge: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(bridge.as_str()).unwrap();
    let bridge_key = Bridge::<'_, { AccountState::Initialized }>::key(None, &program_id);

    bridge_key.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_state(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&BridgeData::try_from_slice(data.as_slice()).unwrap()).unwrap()
}

#[wasm_bindgen]
pub fn fee_collector_address(bridge: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(bridge.as_str()).unwrap();
    let bridge_key = FeeCollector::key(None, &program_id);

    bridge_key.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn claim_address(program_id: String, vaa: Vec<u8>) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();

    let vaa = VAA::deserialize(vaa.as_slice()).unwrap();
    let claim_key = Claim::<'_>::key(
        &ClaimDerivationData {
            emitter_address: vaa.emitter_address,
            emitter_chain: vaa.emitter_chain,
            sequence: vaa.sequence,
        },
        &program_id,
    );
    claim_key.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_posted_message(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(
        &PostedVAAData::try_from_slice(data.as_slice())
            .unwrap()
            .message,
    )
    .unwrap()
}

#[wasm_bindgen]
pub fn parse_vaa(data: Vec<u8>) -> JsValue {
    JsValue::from_serde(&VAA::deserialize(data.as_slice()).unwrap()).unwrap()
}
