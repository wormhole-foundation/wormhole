use anchor_lang::prelude::*;
use wormhole_solana_common::{legacy_account, LegacyDiscriminator};

/// NOTE: This account's PDA seeds are inconsistent with how other Core Bridges save consumed VAAs.
/// This account uses a tuple of (emitter_chain, emitter_address, sequence) whereas other Core
/// Bridge implementations use the message hash.
#[legacy_account]
#[derive(Debug, PartialEq, Eq, InitSpace)]
pub struct Claim {
    /// This member is not necessary, but we must preserve it since the legacy bridge assumes this
    /// serialization for consumed VAAs (it is set to true when a VAA has been claimed). The fact
    /// that this account exists at all should be enough to protect against a replay attack.
    pub is_complete: bool,
}

impl LegacyDiscriminator<0> for Claim {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}
