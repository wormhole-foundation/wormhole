//! On-chain state access helpers.
//!
//! - `pending`: per-(chain, emitter, sequence, digest) signature-accumulation
//!   bucket.
//! - `chain_registration`: per-chain Token Bridge emitter registration,
//!   cross-checked on the submit paths.
//!
//! Further state modules (`account`, `digest`, `modification`) land with the
//! instructions that consume them.

pub mod chain_registration;
pub mod pending;
