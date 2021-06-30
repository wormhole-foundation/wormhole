#![feature(const_generics)]
pub mod config;
pub mod forward;
pub mod initialize;
mod set_config;

use solitaire::{
    solitaire
};

use config::Pyth2WormholeConfig;
use forward::{forward_price, Forward, ForwardData};
use initialize::{initialize, Initialize};
use set_config::{set_config, SetConfig};

solitaire! {
    Forward(ForwardData) => forward_price,
    Initialize(Pyth2WormholeConfig) => initialize,
    SetConfig(Pyth2WormholeConfig) => set_config,
}
