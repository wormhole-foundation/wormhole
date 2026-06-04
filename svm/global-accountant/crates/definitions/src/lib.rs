//! Shared types and constants for the Wormhole Global Accountant Solana program.
//!
//! No Solana dependency so the layouts can be re-used from on-chain code,
//! host-side tests, and client tooling.

#![no_std]

use bytemuck::{Pod, Zeroable};

/// 32-byte address, layout-compatible with `solana_address::Address` and
/// pinocchio's `Address`. Untyped to keep this crate Solana-SDK-free.
pub type Pubkey = [u8; 32];

/// Instruction discriminators. Single-byte prefix on the instruction data.
/// Discriminator `0` is reserved.
#[repr(u8)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Instruction {
    CloseDigest = 1,
    SubmitObservations = 2,
    ClosePending = 3,
    /// Permissionless signed-VAA backfill via the Verify VAA Shim CPI; applies
    /// balance effects directly, bypassing the quorum tracker.
    SubmitVaas = 4,
    /// Token Bridge `RegisterChain` governance handler: verifies the VAA and
    /// initialises/updates the canonical `ChainRegistration` PDA.
    RegisterChain = 5,
    /// Accountant `ModifyBalance` governance handler: verifies the VAA and
    /// applies an Add/Subtract delta to the canonical `BalanceAccount` PDA.
    ModifyBalance = 6,
}

impl Instruction {
    pub const fn from_u8(value: u8) -> Option<Self> {
        match value {
            1 => Some(Self::CloseDigest),
            2 => Some(Self::SubmitObservations),
            3 => Some(Self::ClosePending),
            4 => Some(Self::SubmitVaas),
            5 => Some(Self::RegisterChain),
            6 => Some(Self::ModifyBalance),
            _ => None,
        }
    }
}

/// PDA seed prefix for [`DigestAccountLayout`].
pub const DIGEST_SEED_PREFIX: &[u8] = b"digest";

/// PDA seed prefix for [`PendingObservationsLayout`]. Full tuple:
/// `(b"pending", chain_be, emitter, sequence_be, digest)`. The digest suffix
/// lets fork/reorg observations accumulate in sibling buckets and binds each
/// bucket to its digest, so no runtime digest-equality check is needed.
pub const PENDING_SEED_PREFIX: &[u8] = b"pending";

/// PDA seed prefix for [`BalanceAccountLayout`]. Full tuple:
/// `(b"account", chain_be, token_chain_be, token_address)`. Big-endian chain
/// fields match the VAA wire format and the other seed derivations.
pub const ACCOUNT_SEED_PREFIX: &[u8] = b"account";

/// PDA seed prefix for [`ChainRegistrationLayout`]. Full tuple:
/// `(b"chain_registration", chain_be)`.
pub const CHAIN_REGISTRATION_SEED_PREFIX: &[u8] = b"chain_registration";

/// PDA seed prefix for the global-accountant authority that signs all
/// `solana-noreplay` CPIs. Full tuple: `[b"noreplay-authority"]`. One global
/// authority suffices because the noreplay namespace (`chain_be ‖ emitter`)
/// already segregates per-emitter sequence spaces.
pub const NOREPLAY_AUTHORITY_SEED_PREFIX: &[u8] = b"noreplay-authority";

/// Canonical program ID for `solana-noreplay`
/// (`repMHgR5BEpGLeZvM5iGoNNDPw4eu2BS6sXJzaC8K4t`). Raw bytes to keep this
/// crate Solana-SDK-free.
pub const NOREPLAY_PROGRAM_ID: Pubkey = [
    0x0c, 0xb8, 0x38, 0x00, 0x73, 0xdf, 0x36, 0x25, 0xa1, 0x32, 0x11, 0x1f, 0xee, 0x67, 0x8d, 0xd0,
    0x6b, 0x7e, 0x3d, 0xf2, 0x90, 0xa2, 0xb1, 0xd5, 0x4a, 0x48, 0x5b, 0xdb, 0x72, 0x61, 0x82, 0x91,
];

/// Discriminator for `solana-noreplay`'s `MarkUsed`. Wire format:
/// `[disc: u8][namespace_len: u16 LE][namespace: ≤64 B][sequence: u64 LE]`.
pub const NOREPLAY_MARK_USED_DISCRIMINATOR: u8 = 1;

/// Bits per bitmap bucket. Bucket index is `sequence / BITS_PER_BUCKET`, bit
/// offset is `sequence % BITS_PER_BUCKET`.
pub const NOREPLAY_BITS_PER_BUCKET: u64 = 1024;

/// Bitmap payload size inside a noreplay PDA (account is 1-byte bump + bitmap).
pub const NOREPLAY_BITMAP_BYTES: usize = 128;

/// Byte offset of the bitmap payload (byte 0 is the stored canonical bump).
pub const NOREPLAY_BITMAP_OFFSET: usize = 1;

/// 256-bit unsigned integer stored on-disk as 32 **big-endian** bytes. BE
/// matches the VAA `amount` wire format so a payload's bytes copy in directly.
/// `#[repr(transparent)]` over `[u8; 32]` keeps it `Pod`; arithmetic round-trips
/// through `ruint`'s `U256` at the boundary.
#[repr(transparent)]
#[derive(Clone, Copy, Debug, Default, Eq, PartialEq, Pod, Zeroable)]
pub struct Uint256(pub [u8; 32]);

impl Uint256 {
    /// All-zero value.
    pub const ZERO: Self = Self([0u8; 32]);

    /// All-ones value (`2^256 - 1`).
    pub const MAX: Self = Self([0xffu8; 32]);

    /// Build a `Uint256` from a `u128`, big-endian (low 16 bytes carry the
    /// value).
    pub const fn from_u128(v: u128) -> Self {
        let v_be = v.to_be_bytes();
        let mut bytes = [0u8; 32];
        let mut i = 0;
        while i < 16 {
            bytes[16 + i] = v_be[i];
            i += 1;
        }
        Self(bytes)
    }

    /// Add, returning `None` on overflow.
    #[inline]
    pub fn checked_add(self, other: Self) -> Option<Self> {
        let a = ruint::aliases::U256::from_be_bytes::<32>(self.0);
        let b = ruint::aliases::U256::from_be_bytes::<32>(other.0);
        a.checked_add(b).map(|r| Self(r.to_be_bytes::<32>()))
    }

    /// Subtract, returning `None` on underflow.
    #[inline]
    pub fn checked_sub(self, other: Self) -> Option<Self> {
        let a = ruint::aliases::U256::from_be_bytes::<32>(self.0);
        let b = ruint::aliases::U256::from_be_bytes::<32>(other.0);
        a.checked_sub(b).map(|r| Self(r.to_be_bytes::<32>()))
    }

    /// Construct from 32 big-endian bytes.
    pub const fn from_be_bytes(bytes: [u8; 32]) -> Self {
        Self(bytes)
    }
}

impl PartialOrd for Uint256 {
    fn partial_cmp(&self, other: &Self) -> Option<core::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

impl Ord for Uint256 {
    fn cmp(&self, other: &Self) -> core::cmp::Ordering {
        // Lexicographic over big-endian bytes equals numerical order.
        self.0.cmp(&other.0)
    }
}

/// Zero-copy layout for a `DigestAccount` PDA. Field ordering keeps natural
/// alignment without padding.
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct DigestAccountLayout {
    pub emitter: Pubkey,
    pub digest: [u8; 32],
    pub payer: Pubkey,
    pub sequence: u64,
    pub quorum_at_slot: u64,
    pub guardian_set_index: u32,
    pub chain: u16,
    /// Reserved; zero-initialised on open. Crate-private so callers go through
    /// `Zeroable`.
    pub(crate) _padding: [u8; 2],
}

impl DigestAccountLayout {
    /// Byte length of the layout (also the rent-paying allocation size).
    pub const LEN: usize = core::mem::size_of::<Self>();
}

// Compile-time pins against layout drift.
const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(DigestAccountLayout, emitter) == 0);
    assert!(offset_of!(DigestAccountLayout, digest) == 32);
    assert!(offset_of!(DigestAccountLayout, payer) == 64);
    assert!(offset_of!(DigestAccountLayout, sequence) == 96);
    assert!(offset_of!(DigestAccountLayout, quorum_at_slot) == 104);
    assert!(offset_of!(DigestAccountLayout, guardian_set_index) == 112);
    assert!(offset_of!(DigestAccountLayout, chain) == 116);
    assert!(DigestAccountLayout::LEN == 120);
};

/// Zero-copy layout for a per-`(chain, emitter, sequence)` pending-quorum PDA.
/// On-disk size is **76 bytes** (4-byte alignment, explicit tail padding).
///
/// | offset | size | field              |
/// |--------|------|--------------------|
/// | 0      | 32   | digest             |
/// | 32     | 32   | payer              |
/// | 64     | 4    | guardian_set_index |
/// | 68     | 4    | signatures (u32 bitmap; bit N == guardian-index N signed) |
/// | 72     | 2    | chain              |
/// | 74     | 2    | _padding           |
///
/// The 32-bit bitmap covers 32 guardian indices. A protocol move to >32
/// guardians requires widening the field and bumping the layout version.
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct PendingObservationsLayout {
    pub digest: [u8; 32],
    pub payer: Pubkey,
    pub guardian_set_index: u32,
    pub signatures: u32,
    pub chain: u16,
    /// Explicit tail padding required by `Pod` (no implicit padding allowed).
    /// Crate-private so callers go through `Zeroable`.
    pub(crate) _padding: [u8; 2],
}

impl PendingObservationsLayout {
    /// Byte length of the layout (also the rent-paying allocation size).
    pub const LEN: usize = core::mem::size_of::<Self>();

    /// Quorum threshold: 13 of 19 guardians — the Core Bridge
    /// `(len * 2) / 3 + 1` for len 19. Pinned, not derived from the live set.
    pub const QUORUM_THRESHOLD: u32 = 13;
}

const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(PendingObservationsLayout, digest) == 0);
    assert!(offset_of!(PendingObservationsLayout, payer) == 32);
    assert!(offset_of!(PendingObservationsLayout, guardian_set_index) == 64);
    assert!(offset_of!(PendingObservationsLayout, signatures) == 68);
    assert!(offset_of!(PendingObservationsLayout, chain) == 72);
    assert!(PendingObservationsLayout::LEN == 76);
};

/// Zero-copy balance account for a `(chain, token_chain, token_address)`
/// triple. On-disk size is **76 bytes**.
///
/// | offset | size | field         |
/// |--------|------|---------------|
/// | 0      | 2    | chain         |
/// | 2      | 2    | token_chain   |
/// | 4      | 32   | token_address |
/// | 36     | 32   | balance       |
/// | 68     | 8    | _reserved     |
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct BalanceAccountLayout {
    /// Chain on which this balance is held.
    pub chain: u16,
    /// Native chain of the token.
    pub token_chain: u16,
    /// Token address on its native chain.
    pub token_address: [u8; 32],
    /// Current balance, 32-byte big-endian (matches the VAA `amount` encoding).
    pub balance: Uint256,
    /// Reserved; crate-private so callers go through `Zeroable`.
    pub(crate) _reserved: [u8; 8],
}

impl BalanceAccountLayout {
    pub const LEN: usize = core::mem::size_of::<Self>();
}

// Compile-time pins for the balance layout.
const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(BalanceAccountLayout, chain) == 0);
    assert!(offset_of!(BalanceAccountLayout, token_chain) == 2);
    assert!(offset_of!(BalanceAccountLayout, token_address) == 4);
    assert!(offset_of!(BalanceAccountLayout, balance) == 36);
    assert!(offset_of!(BalanceAccountLayout, _reserved) == 68);
    assert!(BalanceAccountLayout::LEN == 76);
};

/// Zero-copy per-chain Token Bridge emitter registration. One PDA per chain at
/// `(b"chain_registration", chain_be)`, holding the canonical emitter address.
/// Written only by `register_chain`; re-registration with a higher-sequence VAA
/// overwrites the emitter (supports emitter rotation).
///
/// | offset | size | field           |
/// |--------|------|-----------------|
/// | 0      | 2    | chain           |
/// | 2      | 30   | _padding        |
/// | 32     | 32   | emitter_address |
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct ChainRegistrationLayout {
    /// Wormhole chain ID this PDA registers (mirrors the seed bytes).
    pub chain: u16,
    /// Reserved; crate-private so callers go through `Zeroable`.
    pub(crate) _padding: [u8; 30],
    /// Canonical Token Bridge emitter address on `chain`.
    pub emitter_address: [u8; 32],
}

impl ChainRegistrationLayout {
    pub const LEN: usize = core::mem::size_of::<Self>();
}

const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(ChainRegistrationLayout, chain) == 0);
    assert!(offset_of!(ChainRegistrationLayout, _padding) == 2);
    assert!(offset_of!(ChainRegistrationLayout, emitter_address) == 32);
    assert!(ChainRegistrationLayout::LEN == 64);
};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn digest_layout_is_pod_friendly() {
        // Round-trip through bytes.
        let original = DigestAccountLayout {
            emitter: [1u8; 32],
            digest: [2u8; 32],
            payer: [3u8; 32],
            sequence: 0xdead_beef_cafe_babe,
            quorum_at_slot: 42,
            guardian_set_index: 7,
            chain: 1,
            _padding: [0; 2],
        };
        let bytes = bytemuck::bytes_of(&original);
        let copy: &DigestAccountLayout = bytemuck::from_bytes(bytes);
        assert_eq!(&original, copy);
        assert_eq!(DigestAccountLayout::LEN, bytes.len());
    }

    // ---- Uint256 unit tests ----

    #[test]
    fn uint256_add_basic() {
        let a = Uint256::from_u128(500);
        let b = Uint256::from_u128(200);
        assert_eq!(a.checked_add(b), Some(Uint256::from_u128(700)));
    }

    #[test]
    fn uint256_add_overflow() {
        assert_eq!(Uint256::MAX.checked_add(Uint256::from_u128(200)), None);
    }

    #[test]
    fn uint256_sub_basic() {
        let a = Uint256::from_u128(500);
        let b = Uint256::from_u128(200);
        assert_eq!(a.checked_sub(b), Some(Uint256::from_u128(300)));
    }

    #[test]
    fn uint256_sub_underflow() {
        assert_eq!(
            Uint256::ZERO.checked_sub(Uint256::from_u128(200)),
            None,
            "subtracting from zero must return None, not wrap"
        );
    }

    #[test]
    fn uint256_round_trip_bytes() {
        let original = Uint256::from_u128(0xdead_beef_cafe_babe_u128);
        let restored = Uint256::from_be_bytes(original.0);
        assert_eq!(original, restored);
    }

    #[test]
    fn uint256_big_endian_wire_order() {
        // `0x1234` packs MSB-first (network byte order), matching the VAA
        // `amount` encoding.
        let v = Uint256::from_u128(0x1234);
        let mut expected = [0u8; 32];
        expected[30] = 0x12;
        expected[31] = 0x34;
        assert_eq!(v.0, expected);
    }

    #[test]
    fn uint256_ordering() {
        assert!(Uint256::from_u128(1) < Uint256::from_u128(2));
        assert!(Uint256::MAX > Uint256::from_u128(u128::MAX));
        assert!(Uint256::ZERO < Uint256::from_u128(1));
        // Higher MSB sorts higher even with smaller LSBs.
        let mut a_bytes = [0u8; 32];
        a_bytes[0] = 0x01;
        let a = Uint256::from_be_bytes(a_bytes);
        let b = Uint256::from_u128(u128::MAX);
        assert!(a > b);
    }

    #[test]
    fn uint256_add_then_sub_round_trips() {
        let start = Uint256::from_u128(500);
        let added = start.checked_add(Uint256::from_u128(200)).unwrap();
        assert_eq!(added, Uint256::from_u128(700));
        let restored = added.checked_sub(Uint256::from_u128(200)).unwrap();
        assert_eq!(restored, start);
    }

    // ---- BalanceAccountLayout tests ----

    #[test]
    fn balance_layout_size_matches_cosmwasm() {
        assert_eq!(BalanceAccountLayout::LEN, 76);
    }

    #[test]
    fn balance_layout_uint256_offsets_pinned() {
        // Runtime mirror of the const-assert block above.
        use core::mem::offset_of;
        assert_eq!(offset_of!(BalanceAccountLayout, chain), 0);
        assert_eq!(offset_of!(BalanceAccountLayout, token_chain), 2);
        assert_eq!(offset_of!(BalanceAccountLayout, token_address), 4);
        assert_eq!(offset_of!(BalanceAccountLayout, balance), 36);
        assert_eq!(offset_of!(BalanceAccountLayout, _reserved), 68);
    }

    #[test]
    fn balance_layout_is_pod_friendly() {
        let mut token_address = [0u8; 32];
        for (i, b) in token_address.iter_mut().enumerate() {
            *b = i as u8;
        }
        let original = BalanceAccountLayout {
            chain: 1,
            token_chain: 2,
            token_address,
            balance: Uint256::from_u128(0xcafe_babe),
            _reserved: [0u8; 8],
        };
        let bytes = bytemuck::bytes_of(&original);
        assert_eq!(bytes.len(), BalanceAccountLayout::LEN);
        let copy: &BalanceAccountLayout = bytemuck::from_bytes(bytes);
        assert_eq!(&original, copy);
    }

    // ---- PendingObservationsLayout tests ----

    #[test]
    fn pending_layout_size_pinned() {
        // Runtime mirror of the const-assert above (incl. 2 bytes tail padding).
        assert_eq!(PendingObservationsLayout::LEN, 76);
    }

    #[test]
    fn pending_layout_offsets_pinned() {
        use core::mem::offset_of;
        assert_eq!(offset_of!(PendingObservationsLayout, digest), 0);
        assert_eq!(offset_of!(PendingObservationsLayout, payer), 32);
        assert_eq!(
            offset_of!(PendingObservationsLayout, guardian_set_index),
            64
        );
        assert_eq!(offset_of!(PendingObservationsLayout, signatures), 68);
        assert_eq!(offset_of!(PendingObservationsLayout, chain), 72);
    }

    #[test]
    fn pending_layout_is_pod_friendly() {
        let mut digest = [0u8; 32];
        for (i, b) in digest.iter_mut().enumerate() {
            *b = i as u8;
        }
        let original = PendingObservationsLayout {
            digest,
            payer: [0xAA; 32],
            guardian_set_index: 0x0BAD_CAFE,
            signatures: 0x0000_1FFFu32, // 13 low bits set
            chain: 1,
            _padding: [0; 2],
        };
        let bytes = bytemuck::bytes_of(&original);
        let copy: &PendingObservationsLayout = bytemuck::from_bytes(bytes);
        assert_eq!(&original, copy);
        assert_eq!(PendingObservationsLayout::LEN, bytes.len());
    }

    #[test]
    fn balance_layout_balance_encodes_big_endian_on_disk() {
        // The on-disk balance bytes must be the big-endian encoding, so a VAA
        // `amount` slice copies in without byte-order conversion.
        let original = BalanceAccountLayout {
            chain: 0,
            token_chain: 0,
            token_address: [0u8; 32],
            balance: Uint256::from_u128(0x1234_5678),
            _reserved: [0u8; 8],
        };
        let bytes = bytemuck::bytes_of(&original);
        let balance_slice = &bytes[36..68];
        let mut expected = [0u8; 32];
        expected[28] = 0x12;
        expected[29] = 0x34;
        expected[30] = 0x56;
        expected[31] = 0x78;
        assert_eq!(balance_slice, &expected);
    }
}
