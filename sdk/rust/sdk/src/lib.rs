//! This SDK provides API's for implementing cross-chain message passing via the Wormhole protocol.
//! This package aims to provide a consistent API regardless of the underlying runtime, but some
//! types will differ depending on which implementation is being targeted.
//!
//! Each implementation can be toggled using feature flags, which will switch out the underlying
//! depenencies to pull in the depenendices for the corresponding runtimes.
//!
//! Implementations:
//!
//! Runtime   | Feature Flag            | Version
//! ----------|-------------------------|---------------------------------------------------- 
//! Solana    | --feature=solana        | solana-sdk 1.7.1 
//! Terra     | --feature=terra         | cosmos-sdk 0.16.0 
//!
//! Docs specific to each blockchain's runtime can be found in submodules within the chains module
//! at the root of this package.

pub mod chains;

pub use wormhole_core::*;
pub use chains::*;
