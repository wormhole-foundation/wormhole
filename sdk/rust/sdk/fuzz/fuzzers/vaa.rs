#![no_main]
use libfuzzer_sys::fuzz_target;
use wormhole_sdk::vaa::VAA;

fuzz_target!(|data: &[u8]| {
    VAA::from_bytes(data);
});

