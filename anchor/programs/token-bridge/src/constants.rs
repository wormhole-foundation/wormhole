use anchor_lang::prelude::constant;

#[constant]
/// Seed for upgrade authority (A.K.A. "upgrade").
pub const UPGRADE_SEED_PREFIX: &[u8] = b"upgrade";

#[constant]
/// Seed for Core Bridge emitter (A.K.A. "emitter").
pub const EMITTER_SEED_PREFIX: &[u8] = b"emitter";

#[constant]
/// Seed for token transfer authority (A.K.A. "authority_signer").
pub const TRANSFER_AUTHORITY_SEED_PREFIX: &[u8] = b"authority_signer";

#[constant]
/// Seed for token custody authority (A.K.A. "custody_signer").
pub const CUSTODY_AUTHORITY_SEED_PREFIX: &[u8] = b"custody_signer";

#[constant]
/// Seed for mint authority (A.K.A. "mint_signer").
pub const MINT_AUTHORITY_SEED_PREFIX: &[u8] = b"mint_signer";

#[constant]
/// Seed for wrapped mint (A.K.A. "wrapped").
pub const WRAPPED_MINT_SEED_PREFIX: &[u8] = b"wrapped";

#[constant]
pub const MAX_DECIMALS: u8 = 8;
