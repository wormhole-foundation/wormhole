// #![cfg(all(target_arch = "bpf", not(feature = "no-entrypoint")))]

// Salt contains the framework definition, single file for now but to be extracted into a cargo
// package as soon as possible.
mod api;
mod types;

use api::{initialize, Initialize};
use types::BridgeConfig;

use solitaire::*;

solitaire! {
    Initialize(config: BridgeConfig) => initialize,
}
