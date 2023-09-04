use anchor_lang::prelude::*;

/// NOTE: This account's PDA seeds are inconsistent with how other Core Bridges save consumed VAAs.
/// This account uses a tuple of (emitter_chain, emitter_address, sequence) whereas other Core
/// Bridge implementations use the message hash.
#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Claim {
    /// This member is not necessary, but we must preserve it since the legacy bridge assumes this
    /// serialization for consumed VAAs (it is set to true when a VAA has been claimed). The fact
    /// that this account exists at all should be enough to protect against a replay attack.
    pub is_complete: bool,
}

impl Owner for Claim {
    fn owner() -> Pubkey {
        crate::ID
    }
}

impl core_bridge_program::legacy::utils::LegacyDiscriminator<0> for Claim {
    const LEGACY_DISCRIMINATOR: [u8; 0] = [];
}
