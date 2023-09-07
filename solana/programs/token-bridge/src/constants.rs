//! Constants used by the Token Bridge Program. For integrators, necessary constants are re-exported
//! in the [sdk](crate::sdk) module.

use anchor_lang::prelude::constant;

/// Seed for upgrade authority.
#[constant]
pub const UPGRADE_SEED_PREFIX: &[u8] = b"upgrade";

/// Seed for Core Bridge emitter.
#[constant]
pub const EMITTER_SEED_PREFIX: &[u8] = b"emitter";

/// Seed for token transfer authority, who is delegated authority to transfer (or burn) tokens from
/// another (for outbound transfers).
#[constant]
pub const TRANSFER_AUTHORITY_SEED_PREFIX: &[u8] = b"authority_signer";

/// Seed for token custody authority, who is the owner of the Token Bridge's token accounts
/// warehousing native (not minted by Token Bridge) assets.
#[constant]
pub const CUSTODY_AUTHORITY_SEED_PREFIX: &[u8] = b"custody_signer";

/// Seed for mint authority, who mints the Token Bridge's wrapped assets.
#[constant]
pub const MINT_AUTHORITY_SEED_PREFIX: &[u8] = b"mint_signer";

/// Seed for Token Bridge's wrapped assets (mints).
#[constant]
pub const WRAPPED_MINT_SEED_PREFIX: &[u8] = b"wrapped";

/// Seed for program sender if an integrator wants to use his program ID as the from address.
#[constant]
pub const PROGRAM_SENDER_SEED_PREFIX: &[u8] = b"sender";

/// Seed for program redeemer if an integrator redeems a token transfer using his program ID as the
/// redeemer address.
#[constant]
pub const PROGRAM_REDEEMER_SEED_PREFIX: &[u8] = b"redeemer";

/// Maximum decimals allowed for a wrapped asset. This constant is also used to convert >8 native
/// asset amounts to a normalized amount to be encoded in Token Bridge transfers.
#[constant]
pub const MAX_DECIMALS: u8 = 8;

pub(crate) const GOVERNANCE_CHAIN: u16 = 1;

pub(crate) const GOVERNANCE_EMITTER: [u8; 32] = [
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4,
];
