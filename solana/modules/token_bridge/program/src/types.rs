use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use serde::{
    Deserialize,
    Serialize,
};
use solana_program::{
    program_error::ProgramError,
    program_pack::{
        IsInitialized,
        Pack,
    },
    pubkey::{
        Pubkey,
        PUBKEY_BYTES,
    },
};
use solitaire::{
    pack_type,
    processors::seeded::{
        AccountOwner,
        Owned,
        SingleOwned,
    },
};
use spl_token::state::{
    Account,
    Mint,
};
use std::io;

pub type Address = [u8; 32];
pub type ChainID = u16;

#[derive(Default, Clone, Copy, BorshSerialize, Serialize, Deserialize)]
pub struct Config {
    pub wormhole_bridge: Pubkey,
}

// Hand-rolled `BorshDeserialize` so `try_from_slice` tolerates the pauser tail (bytes
// CONFIG_BORSH_LEN..CONFIG_WITH_PAUSER_LEN, i.e. 32..137) that the realloc'd Config carries after
// the first `SetPauserAddresses` governance VAA.
// The derived `try_from_slice` rejects any trailing bytes, which would break every handler
// that peels `ConfigAccount` (transfer, complete, attest, …) on a migrated bridge.
impl BorshDeserialize for Config {
    fn deserialize(buf: &mut &[u8]) -> io::Result<Self> {
        Ok(Config {
            wormhole_bridge: <Pubkey as BorshDeserialize>::deserialize(buf)?,
        })
    }

    fn try_from_slice(v: &[u8]) -> io::Result<Self> {
        let mut cursor = v;
        <Self as BorshDeserialize>::deserialize(&mut cursor)
    }
}

#[cfg(not(feature = "cpi"))]
impl Owned for Config {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for Config {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

/// Pauser tail layout written at offset `CONFIG_BORSH_LEN` of the existing `Config` PDA.
///
/// Backwards-compatible extension: the `Config` Borsh struct itself stays 32 bytes, so existing
/// SDK transactions that pass the Config as writable continue to round-trip — solitaire's
/// `Persist::persist` calls Borsh `serialize` which writes only the first 32 bytes and leaves
/// the tail untouched.
///
/// Lazy migration: legacy 32-byte accounts are treated as "unpaused, no pauser configured".
/// The first `SetPauserAddresses` governance VAA realloc's the account to
/// `CONFIG_BORSH_LEN + PAUSER_TAIL_LEN` bytes and writes the tail. After migration, both
/// `pauser` and `unpauser` may still be set to `Pubkey::default()` via governance — this is
/// the canonical "role unassigned" encoding, and `api::pause` reverts before comparing the
/// caller in that case (see whitepapers/0003_token_bridge.md Pausing).
///
/// Tail layout (137-byte total Config account). The three roles are stored in the same order as
/// the `SetPauserAddresses` wire format — pauser, freezer, unpauser:
///   bytes  32 ..  33 : `paused`       (u8, exactly 0 or 1 — see `write_paused` / `paused()`)
///   bytes  33 ..  65 : `pauser`       (Pubkey)
///   bytes  65 ..  97 : `freezer`      (Pubkey)
///   bytes  97 .. 129 : `unpauser`     (Pubkey)
///   bytes 129 .. 137 : `pause_expiry` (i64 LE, unix seconds — matches Clock.unix_timestamp)
///
/// The first `SetPauserAddresses` realloc's straight to the full 137-byte size (see
/// `api::governance`). See whitepapers/0003_token_bridge.md Pausing.

// Account-size sentinels. A Config account is either CONFIG_BORSH_LEN (legacy / un-migrated, the
// size at which the bridge `initialize` creates it) or CONFIG_WITH_PAUSER_LEN (post-migration).
// Any other length on a successfully-deserialized account would indicate corruption.
pub const CONFIG_BORSH_LEN: usize = 32;
pub const PAUSER_TAIL_LEN: usize = 1 + PUBKEY_BYTES + PUBKEY_BYTES + PUBKEY_BYTES + 8;
pub const CONFIG_WITH_PAUSER_LEN: usize = CONFIG_BORSH_LEN + PAUSER_TAIL_LEN;

// Byte offsets inside the tail (relative to the start of the Config account). Roles are laid out
// in the same order as the `SetPauserAddresses` wire format: pauser, freezer, unpauser.
pub const PAUSED_OFFSET: usize = CONFIG_BORSH_LEN;
pub const PAUSER_OFFSET: usize = PAUSED_OFFSET + 1;
pub const FREEZER_OFFSET: usize = PAUSER_OFFSET + PUBKEY_BYTES;
pub const UNPAUSER_OFFSET: usize = FREEZER_OFFSET + PUBKEY_BYTES;
pub const PAUSE_EXPIRY_OFFSET: usize = UNPAUSER_OFFSET + PUBKEY_BYTES;

// Pin offset/length relationships at compile time so a future tweak to the constants can't
// silently corrupt the tail layout. (Written with `if ... { panic!() }` rather than `assert!`
// because clippy's `assertions_on_constants` lint flags compile-time-constant `assert!` calls.)
const _: () = {
    if PAUSED_OFFSET >= CONFIG_WITH_PAUSER_LEN {
        panic!("PAUSED_OFFSET must fall inside the tail");
    }
    if PAUSER_OFFSET + PUBKEY_BYTES > CONFIG_WITH_PAUSER_LEN {
        panic!("pauser slot must fit inside CONFIG_WITH_PAUSER_LEN");
    }
    if UNPAUSER_OFFSET + PUBKEY_BYTES > CONFIG_WITH_PAUSER_LEN {
        panic!("unpauser slot must fit inside CONFIG_WITH_PAUSER_LEN");
    }
    if FREEZER_OFFSET + PUBKEY_BYTES > CONFIG_WITH_PAUSER_LEN {
        panic!("freezer slot must fit inside CONFIG_WITH_PAUSER_LEN");
    }
    if PAUSE_EXPIRY_OFFSET + 8 != CONFIG_WITH_PAUSER_LEN {
        panic!("pause_expiry slot must end exactly at CONFIG_WITH_PAUSER_LEN");
    }
};

/// Read the `paused` flag. Returns `false` for legacy (un-migrated) accounts; otherwise the
/// stored byte is interpreted strictly — `0` is unpaused, `1` is paused, any other value is
/// considered corrupted and treated as paused (fail-closed). Writers (`write_paused`) only ever
/// store `0` or `1`, so a non-canonical byte should not occur in practice.
#[must_use]
pub fn paused(config_data: &[u8]) -> bool {
    if config_data.len() < CONFIG_WITH_PAUSER_LEN {
        return false;
    }
    match config_data[PAUSED_OFFSET] {
        0 => false,
        1 => true,
        // Fail-closed on corruption: any non-canonical byte is treated as paused.
        _ => true,
    }
}

/// Read the configured pauser. Returns `Pubkey::default()` for legacy (un-migrated) accounts
/// AND for accounts where governance explicitly set the role to the zero pubkey.
#[must_use]
pub fn pauser(config_data: &[u8]) -> Pubkey {
    if config_data.len() < CONFIG_WITH_PAUSER_LEN {
        return Pubkey::default();
    }
    Pubkey::new(&config_data[PAUSER_OFFSET..(PAUSER_OFFSET + PUBKEY_BYTES)])
}

/// Read the configured unpauser. Same legacy / unassigned semantics as [`pauser`].
#[must_use]
pub fn unpauser(config_data: &[u8]) -> Pubkey {
    if config_data.len() < CONFIG_WITH_PAUSER_LEN {
        return Pubkey::default();
    }
    Pubkey::new(&config_data[UNPAUSER_OFFSET..(UNPAUSER_OFFSET + PUBKEY_BYTES)])
}

/// Read the configured freezer. Same legacy / unassigned semantics as [`pauser`].
#[must_use]
pub fn freezer(config_data: &[u8]) -> Pubkey {
    if config_data.len() < CONFIG_WITH_PAUSER_LEN {
        return Pubkey::default();
    }
    Pubkey::new(&config_data[FREEZER_OFFSET..(FREEZER_OFFSET + PUBKEY_BYTES)])
}

/// Read the pause expiry (unix seconds): the point at which an active pause becomes eligible to
/// be lifted permissionlessly via `api::pause::unpause_expired`. Returns `0` for legacy
/// (un-migrated) accounts. See whitepapers/0003_token_bridge.md Pausing.
#[must_use]
pub fn pause_expiry(config_data: &[u8]) -> i64 {
    if config_data.len() < CONFIG_WITH_PAUSER_LEN {
        return 0;
    }
    let mut buf = [0u8; 8];
    buf.copy_from_slice(&config_data[PAUSE_EXPIRY_OFFSET..(PAUSE_EXPIRY_OFFSET + 8)]);
    i64::from_le_bytes(buf)
}

pub(crate) fn write_paused(config_data: &mut [u8], paused: bool) {
    config_data[PAUSED_OFFSET] = u8::from(paused);
}

pub(crate) fn write_pause_expiry(config_data: &mut [u8], expiry: i64) {
    config_data[PAUSE_EXPIRY_OFFSET..(PAUSE_EXPIRY_OFFSET + 8)]
        .copy_from_slice(&expiry.to_le_bytes());
}

/// Write all three pause-authority roles (pauser, freezer, unpauser) atomically. Leaves `paused`
/// and `pause_expiry` untouched so a governance rotation does not change the live pause state.
pub(crate) fn write_pause_authorities(
    config_data: &mut [u8],
    pauser: &Pubkey,
    freezer: &Pubkey,
    unpauser: &Pubkey,
) {
    config_data[PAUSER_OFFSET..(PAUSER_OFFSET + PUBKEY_BYTES)].copy_from_slice(&pauser.to_bytes());
    config_data[FREEZER_OFFSET..(FREEZER_OFFSET + PUBKEY_BYTES)]
        .copy_from_slice(&freezer.to_bytes());
    config_data[UNPAUSER_OFFSET..(UNPAUSER_OFFSET + PUBKEY_BYTES)]
        .copy_from_slice(&unpauser.to_bytes());
}

/// Returns `Err(Paused)` if the bridge is currently paused. Legacy (un-extended) Config
/// accounts are treated as unpaused. Call from every non-governance, non-`unpause` entry point.
pub fn require_not_paused(
    config_info: &solana_program::account_info::AccountInfo,
) -> solitaire::Result<()> {
    let data = config_info.data.borrow();
    if paused(&data) {
        return Err(crate::TokenBridgeError::Paused.into());
    }
    Ok(())
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct EndpointRegistration {
    pub chain: ChainID,
    pub contract: Address,
}

#[cfg(not(feature = "cpi"))]
impl Owned for EndpointRegistration {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

impl SingleOwned for EndpointRegistration {
}

#[cfg(feature = "cpi")]
impl Owned for EndpointRegistration {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

#[derive(Default, Clone, Copy, BorshDeserialize, BorshSerialize, Serialize, Deserialize)]
pub struct WrappedMeta {
    pub chain: ChainID,
    pub token_address: Address,
    pub original_decimals: u8,
}

impl SingleOwned for WrappedMeta {
}

#[cfg(not(feature = "cpi"))]
impl Owned for WrappedMeta {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

#[cfg(feature = "cpi")]
impl Owned for WrappedMeta {
    fn owner(&self) -> AccountOwner {
        use std::str::FromStr;
        AccountOwner::Other(Pubkey::from_str(env!("TOKEN_BRIDGE_ADDRESS")).unwrap())
    }
}

pub mod spl_token_2022 {
    use solana_program::pubkey::Pubkey;
    use std::str::FromStr;

    pub fn id() -> Pubkey {
        Pubkey::from_str("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb").unwrap()
    }
}

trait MintExtensionPack: Pack {
    fn unpack(input: &[u8]) -> Result<Self, ProgramError>;
}

// from: https://github.com/solana-program/token-2022/blob/9cbf7a1e9bab57aabb71b4b84fc84e3670108573/interface/src/extension/mod.rs#L262-L268
// Since there is no account discriminator in these accounts, it's possible to confuse a multisig account for a mint account.
// This check prevents that by ensuring the length is not equal to a multisig account length.
// Mint accounts that happen to be 355 bytes long are out of luck (but this won't concern us, if it's even possible).
fn check_min_len_and_not_multisig(input: &[u8], minimum_len: usize) -> Result<(), ProgramError> {
    const MULTISIG_LEN: usize = 355; // spl_token::state::Multisig::LEN;
    if input.len() == MULTISIG_LEN || input.len() < minimum_len {
        Err(ProgramError::InvalidAccountData)
    } else {
        Ok(())
    }
}

impl MintExtensionPack for Mint {
    // this implementation is almost identical to the default Pack::unpack,
    // except for the length check. Instead of requiring exact length, we require
    // a minimum length, to allow for extensions.
    fn unpack(input: &[u8]) -> Result<Self, ProgramError> {
        check_min_len_and_not_multisig(input, Self::LEN)?;
        let value: Mint = solana_program::program_pack::Pack::unpack_from_slice(input)?;
        if value.is_initialized() {
            Ok(value)
        } else {
            Err(ProgramError::UninitializedAccount)
        }
    }
}

pack_type!(
    SplMint,
    Mint,
    AccountOwner::OneOf(vec![spl_token::id(), spl_token_2022::id()]),
    MintExtensionPack
);
pack_type!(
    SplAccount,
    Account,
    AccountOwner::OneOf(vec![spl_token::id(), spl_token_2022::id()])
);

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mint_unpack_from_slice_old_token() {
        let src: [u8; 82] = [
            0x01, 0x00, 0x00, 0x00, 0x64, 0xf1, 0x33, 0x5f, 0xe8, 0x35, 0x98, 0x99, 0x99, 0xfb,
            0xd2, 0x84, 0x35, 0xc9, 0x0b, 0x89, 0x47, 0xfb, 0x25, 0x8f, 0x7a, 0xea, 0xcb, 0x19,
            0xc8, 0x8f, 0x9b, 0x09, 0x7a, 0xe2, 0xc7, 0xe7, 0x00, 0xc0, 0x57, 0x73, 0xa5, 0x7c,
            0x02, 0x00, 0x09, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        ];
        let mint = <Mint as MintExtensionPack>::unpack(&src).unwrap();
        assert!(mint.is_initialized);
    }

    #[test]
    fn test_mint_unpack_from_slice_new_token() {
        let src: [u8; 344] = [
            0x01, 0x00, 0x00, 0x00, 0x67, 0x94, 0x7e, 0xf1, 0x3a, 0x15, 0x8c, 0xb9, 0xbf, 0xca,
            0xbe, 0xa0, 0x18, 0xb3, 0xf8, 0xd2, 0xe5, 0x5b, 0x22, 0x81, 0xa7, 0x63, 0x62, 0x62,
            0x42, 0x73, 0x97, 0x1d, 0xba, 0xfa, 0x1e, 0x99, 0x00, 0xe8, 0x76, 0x48, 0x17, 0x00,
            0x00, 0x00, 0x09, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
            0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x12, 0x00,
            0x40, 0x00, 0x67, 0x94, 0x7e, 0xf1, 0x3a, 0x15, 0x8c, 0xb9, 0xbf, 0xca, 0xbe, 0xa0,
            0x18, 0xb3, 0xf8, 0xd2, 0xe5, 0x5b, 0x22, 0x81, 0xa7, 0x63, 0x62, 0x62, 0x42, 0x73,
            0x97, 0x1d, 0xba, 0xfa, 0x1e, 0x99, 0x94, 0xf5, 0xf0, 0x2e, 0xe1, 0x66, 0xcd, 0x5e,
            0x14, 0xa4, 0x1e, 0x22, 0xeb, 0x6d, 0x93, 0xda, 0x79, 0xd7, 0x50, 0xda, 0x9d, 0xbc,
            0xa4, 0x84, 0xf1, 0x88, 0x3d, 0x47, 0x55, 0xbc, 0x6f, 0x5b, 0x13, 0x00, 0x6a, 0x00,
            0x67, 0x94, 0x7e, 0xf1, 0x3a, 0x15, 0x8c, 0xb9, 0xbf, 0xca, 0xbe, 0xa0, 0x18, 0xb3,
            0xf8, 0xd2, 0xe5, 0x5b, 0x22, 0x81, 0xa7, 0x63, 0x62, 0x62, 0x42, 0x73, 0x97, 0x1d,
            0xba, 0xfa, 0x1e, 0x99, 0x94, 0xf5, 0xf0, 0x2e, 0xe1, 0x66, 0xcd, 0x5e, 0x14, 0xa4,
            0x1e, 0x22, 0xeb, 0x6d, 0x93, 0xda, 0x79, 0xd7, 0x50, 0xda, 0x9d, 0xbc, 0xa4, 0x84,
            0xf1, 0x88, 0x3d, 0x47, 0x55, 0xbc, 0x6f, 0x5b, 0x04, 0x00, 0x00, 0x00, 0x54, 0x65,
            0x73, 0x74, 0x04, 0x00, 0x00, 0x00, 0x54, 0x65, 0x73, 0x74, 0x12, 0x00, 0x00, 0x00,
            0x68, 0x74, 0x74, 0x70, 0x73, 0x3a, 0x2f, 0x2f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
            0x2e, 0x63, 0x6f, 0x6d, 0x00, 0x00, 0x00, 0x00,
        ];
        let mint = <Mint as MintExtensionPack>::unpack(&src).unwrap();
        assert!(mint.is_initialized);
    }
}
