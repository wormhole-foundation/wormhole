use anchor_lang::prelude::*;
use wormhole_solana_common::SeedPrefix;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone, Eq, PartialEq, InitSpace)]
pub struct FeeCollector {}

impl SeedPrefix for FeeCollector {
    #[inline]
    fn seed_prefix() -> &'static [u8] {
        b"fee_collector"
    }
}

impl AccountDeserialize for FeeCollector {
    fn try_deserialize_unchecked(buf: &mut &[u8]) -> Result<Self> {
        Self::deserialize(buf).map_err(Into::into)
    }
}

impl AccountSerialize for FeeCollector {}

impl Owner for FeeCollector {
    fn owner() -> Pubkey {
        solana_program::system_program::ID
    }
}
