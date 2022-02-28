#![feature(adt_const_params)]
pub mod attest;
pub mod config;
pub mod initialize;
pub mod set_config;

use solitaire::solitaire;

pub use attest::{
    attest,
    Attest,
    AttestData,
};
pub use config::Pyth2WormholeConfig;
pub use initialize::{
    initialize,
    Initialize,
};
pub use set_config::{
    set_config,
    SetConfig,
};

solitaire! {
    Attest(AttestData) => attest,
    Initialize(Pyth2WormholeConfig) => initialize,
    SetConfig(Pyth2WormholeConfig) => set_config,
}

#[cfg(feature = "wasm")]
#[cfg(all(target_arch = "wasm32", target_os = "unknown"))]
pub mod wasm;
