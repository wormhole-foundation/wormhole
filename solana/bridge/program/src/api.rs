pub mod close_posted_message;
pub mod close_signature_set_and_posted_vaa;
pub mod governance;
pub mod initialize;
pub mod post_message;
pub mod post_vaa;
pub mod verify_signature;

pub use close_posted_message::*;
pub use close_signature_set_and_posted_vaa::*;
pub use governance::*;
pub use initialize::*;
pub use post_message::*;
pub use post_vaa::*;
pub use verify_signature::*;

/// Retention window (in seconds) for closing `PostedMessage` / `PostedVAA`
/// accounts. 30 days on mainnet / devnet / Fogo; 1 day on Solana testnet so
/// we can exercise the reclaim flow end-to-end without waiting a month.
///
/// Dispatched at compile time via `crate::solana_env`, so no runtime branch.
/// The classifier itself is covered by unit tests in `lib.rs`, which is why
/// we don't need a separate test for this match arm.
pub(crate) const RETENTION_PERIOD: i64 = match crate::solana_env(env!("BRIDGE_ADDRESS")) {
    Some(crate::SolanaEnv::Testnet) => 24 * 60 * 60,
    _ => 30 * 24 * 60 * 60,
};
