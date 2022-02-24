use solitaire::Seeded;
use solana_program::pubkey::Pubkey;
use wasm_bindgen::prelude::*;

use std::str::FromStr;

use crate::{attest::P2WEmitter, types};

#[wasm_bindgen]
pub fn get_emitter_address(program_id: String) -> Vec<u8> {
    let program_id = Pubkey::from_str(program_id.as_str()).unwrap();
    let emitter = P2WEmitter::key(None, &program_id);

    emitter.to_bytes().to_vec()
}

#[wasm_bindgen]
pub fn parse_attestation(bytes: Vec<u8>) -> JsValue {
    let a = types::PriceAttestation::deserialize(bytes.as_slice()).unwrap();
    
    JsValue::from_serde(&a).unwrap()
}

#[wasm_bindgen]
pub fn parse_batch_attestation(bytes: Vec<u8>) -> JsValue {
    let a = types::batch_deserialize(bytes.as_slice()).unwrap();
    

    JsValue::from_serde(&a).unwrap()
}
