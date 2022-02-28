use wasm_bindgen::prelude::*;

use crate::{
    BatchPriceAttestation,
    PriceAttestation,
};

#[wasm_bindgen]
pub fn parse_attestation(bytes: Vec<u8>) -> JsValue {
    let a = PriceAttestation::deserialize(bytes.as_slice()).unwrap();

    JsValue::from_serde(&a).unwrap()
}

#[wasm_bindgen]
pub fn parse_batch_attestation(bytes: Vec<u8>) -> JsValue {
    let a = BatchPriceAttestation::deserialize(bytes.as_slice()).unwrap();

    JsValue::from_serde(&a).unwrap()
}
