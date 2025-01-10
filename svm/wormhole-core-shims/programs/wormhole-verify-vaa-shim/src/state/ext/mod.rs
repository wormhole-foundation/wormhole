pub use anchor_lang::prelude::*;
use wormhole_solana_consts::CORE_BRIDGE_PROGRAM_ID;

#[derive(Debug, AnchorSerialize, AnchorDeserialize, Clone)]
pub struct WormholeGuardianSet {
    /// Index representing an incrementing version number for this guardian set.
    pub index: u32,

    /// Ethereum-style public keys.
    pub keys: Vec<[u8; 20]>,

    /// Timestamp representing the time this guardian became active.
    pub creation_time: u32,

    /// Expiration time when VAAs issued by this set are no longer valid.
    pub expiration_time: u32,
}

// TODO: does there need to be some sort of seed check as well?
impl Owner for WormholeGuardianSet {
    fn owner() -> Pubkey {
        CORE_BRIDGE_PROGRAM_ID
    }
}

// workaround for anchor 0.30.1
// https://github.com/coral-xyz/anchor/blob/e6d7dafe12da661a36ad1b4f3b5970e8986e5321/spl/src/idl_build.rs#L11
impl anchor_lang::Discriminator for WormholeGuardianSet {
    const DISCRIMINATOR: [u8; 8] = [0; 8];
}

impl AccountSerialize for WormholeGuardianSet {}

impl AccountDeserialize for WormholeGuardianSet {
    fn try_deserialize_unchecked(buf: &mut &[u8]) -> Result<Self> {
        Self::deserialize(buf).map_err(Into::into)
    }
}

impl WormholeGuardianSet {
    pub const SEED_PREFIX: &'static [u8] = b"GuardianSet";

    pub fn is_active(&self, timestamp: &u32) -> bool {
        // Note: This is a fix for Wormhole on mainnet.  The initial guardian set was never expired
        // so we block it here.
        if self.index == 0 && self.creation_time == 1628099186 {
            false
        } else {
            self.expiration_time == 0 || self.expiration_time >= *timestamp
        }
    }
}
