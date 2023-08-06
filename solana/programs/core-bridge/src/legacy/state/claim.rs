use anchor_lang::prelude::*;

/// Account used to reflect a consumed VAA. This account is intended to be created once per VAA so
/// it can provide a protection against replay attacks for instructions that redeem VAAs that are
/// meant to only be consumed once.
///
/// NOTE: This account's PDA seeds are inconsistent with how other Core Bridges save consumed VAAs.
/// This account uses a tuple of (emitter_chain, emitter_address, sequence) whereas other Core
/// Bridge implementations use the message digest (double keccak).

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct Claim {
    /// This member is not necessary, but we must preserve it since the legacy bridge assumes this
    /// serialization for consumed VAAs (it is set to true when a VAA has been claimed). The fact
    /// that this account exists at all should be enough to protect against a replay attack.
    pub is_complete: bool,
}

impl crate::legacy::utils::LegacyAccount<0> for Claim {
    const DISCRIMINATOR: [u8; 0] = [];

    fn program_id() -> Pubkey {
        crate::ID
    }
}
