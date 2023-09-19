//! Constants used by the Core Bridge Program. For integrators, necessary constants are re-exported
//! in the [sdk](crate::sdk) module.

use anchor_lang::prelude::constant;

/// Wormhole Chain (Network) ID for Solana.
#[constant]
pub const SOLANA_CHAIN: u16 = 1;

/// Seed for fee collector (Core Bridge's system account).
#[constant]
pub const FEE_COLLECTOR_SEED_PREFIX: &[u8] = b"fee_collector";

#[constant]
/// Seed for upgrade authority.
pub const UPGRADE_SEED_PREFIX: &[u8] = b"upgrade";

/// Seed for program emitters.
#[constant]
pub const PROGRAM_EMITTER_SEED_PREFIX: &[u8] = b"emitter";

/// The max payload size allowed for outbound messages is 30KB. Any messages outbound larger than
/// this size will be disallowed.
#[constant]
pub const MAX_MESSAGE_PAYLOAD_SIZE: usize = 30 * 1_024;

pub(crate) const GOVERNANCE_CHAIN: u16 = 1;

pub(crate) const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];
