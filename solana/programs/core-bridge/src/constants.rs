use anchor_lang::prelude::constant;

#[constant]
pub const SOLANA_CHAIN: u16 = 1;

#[constant]
/// Seed for upgrade authority (A.K.A. "upgrade").
pub const UPGRADE_SEED_PREFIX: &[u8] = b"upgrade";

// Wormhole Messages (inbound and outbound)

/// The max payload size allowed for outbound messages is 30KB. Any messages outbound larger than
/// this size will be disallowed. And VAAs with payload sizes larger than this amount cannot be
/// posted.
#[constant]
pub const MAX_MESSAGE_PAYLOAD_SIZE: usize = 30 * 1_024;
