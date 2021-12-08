#![no_main]
use libfuzzer_sys::fuzz_target;
use wormhole_sdk::VAA;

fuzz_target!(|data: &[u8]| {
    VAA::from_bytes(data);
});

