use anchor_lang::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
struct MetadataUri {
    wormhole_chain_id: u16,
    canonical_address: String,
    native_decimals: u8,
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct LegacyWrappedAsset {
    pub token_chain: u16,
    pub token_address: [u8; 32],
    pub native_decimals: u8,
}

impl core_bridge_program::sdk::legacy::LegacyAccount for LegacyWrappedAsset {
    const DISCRIMINATOR: &'static [u8] = &[];

    fn program_id() -> Pubkey {
        crate::ID
    }
}

impl LegacyWrappedAsset {
    pub const SEED_PREFIX: &'static [u8] = b"meta";
}

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, PartialEq, Eq, InitSpace)]
pub struct WrappedAsset {
    pub legacy: LegacyWrappedAsset,
    pub last_updated_sequence: u64,
}

impl std::ops::Deref for WrappedAsset {
    type Target = LegacyWrappedAsset;

    fn deref(&self) -> &Self::Target {
        &self.legacy
    }
}

impl WrappedAsset {
    pub const SEED_PREFIX: &'static [u8] = LegacyWrappedAsset::SEED_PREFIX;
}

impl core_bridge_program::sdk::legacy::LegacyAccount for WrappedAsset {
    const DISCRIMINATOR: &'static [u8] = LegacyWrappedAsset::DISCRIMINATOR;

    fn program_id() -> Pubkey {
        crate::ID
    }
}
