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

/// Custom error codes returned via `ProgramError::Custom(u32)`. Codes are
/// append-only: a new variant takes the next free value, and existing values
/// are never renumbered once the program ships.
#[repr(u32)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum GlobalAccountantError {
    InvalidInstruction = 0,
    InvalidInstructionData = 1,
    InvalidPda = 2,
    PayerMismatch = 3,
    /// The instruction handler (or a support function it relies on) is not
    /// implemented yet. Raised only by scaffolding stubs.
    NotImplemented = 4,
    /// `(chain, emitter, sequence)` already marked accounted-for in NoReplay.
    AlreadyAccounted = 5,
    /// Signature failed `secp256k1_recover`, or the recovered pubkey did not
    /// match the `guardian_index` in the GuardianSet PDA.
    InvalidSignature = 6,
    /// `guardian_index` out of bounds for the guardian set.
    InvalidGuardianIndex = 7,
    /// The guardian's bit is already set in the pending bitmap.
    AlreadySigned = 8,
    /// Observation references a guardian set older than the one the pending PDA
    /// is accumulating against (stale observation after rotation).
    StaleGuardianSet = 9,
    /// `keccak256(keccak256(body)) != digest`. Rejected before any mutation
    /// since the body carries the transfer payload the commit branch reads.
    BodyDigestMismatch = 10,
    /// Token Bridge payload action is not `0x01`/`0x02`/`0x03`. Rejecting
    /// (rather than committing) leaves the NoReplay slot unconsumed so a future
    /// upgrade can process the VAA.
    UnknownTokenBridgePayload = 11,
}

impl From<GlobalAccountantError> for u32 {
    fn from(e: GlobalAccountantError) -> Self {
        e as u32
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

/// Decoded Token Bridge VAA payload, carrying only the fields the accountant
/// needs at quorum commit.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum TokenBridgeAction {
    /// Action 0x01 (`Transfer`) and 0x03 (`TransferWithPayload`) — same
    /// accountant logic; only these fields affect balances.
    Transfer {
        amount: Uint256,
        token_chain: u16,
        token_address: [u8; 32],
        recipient_chain: u16,
    },
    /// Action 0x02 (`Attest`) — moves no value; commit finishes but skips
    /// balance updates.
    Attest,
    /// Any other action byte. Both commit paths reject it with
    /// [`GlobalAccountantError::UnknownTokenBridgePayload`], leaving the
    /// NoReplay slot unconsumed for a future upgrade.
    Other,
}

/// Fixed VAA body header length: timestamp (4) + nonce (4) + emitter_chain (2)
/// + emitter_address (32) + sequence (8) + consistency_level (1).
pub const VAA_BODY_HEADER_LEN: usize = 51;

/// Routing fields of a VAA body header. `(chain, emitter, sequence)` keys all
/// accountant state, so this struct and [`parse_vaa_body_header`] are the sole
/// authority for these offsets — do not re-derive them in instruction modules.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct VaaBodyHeader {
    /// `emitter_chain`, body bytes `[8..10]` (u16 BE).
    pub chain: u16,
    /// `emitter_address`, body bytes `[10..42]`.
    pub emitter: [u8; 32],
    /// `sequence`, body bytes `[42..50]` (u64 BE).
    pub sequence: u64,
}

/// Parse the routing tuple from a VAA body header. Rejects bodies shorter than
/// the 51-byte header with `InvalidInstructionData`.
pub fn parse_vaa_body_header(body: &[u8]) -> Result<VaaBodyHeader, GlobalAccountantError> {
    if body.len() < VAA_BODY_HEADER_LEN {
        return Err(GlobalAccountantError::InvalidInstructionData);
    }
    let chain = u16::from_be_bytes([body[8], body[9]]);
    let mut emitter = [0u8; 32];
    emitter.copy_from_slice(&body[10..42]);
    let mut sequence_bytes = [0u8; 8];
    sequence_bytes.copy_from_slice(&body[42..50]);
    Ok(VaaBodyHeader {
        chain,
        emitter,
        sequence: u64::from_be_bytes(sequence_bytes),
    })
}

/// Parse a VAA body's payload (bytes at `body[51..]`) into a
/// [`TokenBridgeAction`].
///
/// VAA body layout:
///
/// | offset | size | field              |
/// |--------|------|--------------------|
/// | 0      | 4    | timestamp (u32 BE) |
/// | 4      | 4    | nonce (u32 BE)     |
/// | 8      | 2    | emitter_chain      |
/// | 10     | 32   | emitter_address    |
/// | 42     | 8    | sequence (u64 BE)  |
/// | 50     | 1    | consistency_level  |
/// | 51..   | rest | payload            |
///
/// Token Bridge transfer payload, starting at offset 51:
///
/// | offset | size | field            |
/// |--------|------|------------------|
/// | 0      | 1    | action           |
/// | 1      | 32   | amount (Uint256) |
/// | 33     | 32   | token_address    |
/// | 65     | 2    | token_chain      |
/// | 67     | 32   | recipient        |
/// | 99     | 2    | recipient_chain  |
/// | 101    | 32   | fee (action 1)   |
/// | 133..  | rest | extra (action 3) |
///
/// Requires ≥ 52 bytes (to read the action), or ≥ 184 for transfer actions;
/// returns `InvalidInstructionData` on any short slice.
pub fn parse_token_bridge_payload(body: &[u8]) -> Result<TokenBridgeAction, GlobalAccountantError> {
    const ACTION_TRANSFER: u8 = 0x01;
    const ACTION_ATTEST: u8 = 0x02;
    const ACTION_TRANSFER_WITH_PAYLOAD: u8 = 0x03;
    const TRANSFER_PAYLOAD_MIN: usize = 1 + 32 + 32 + 2 + 32 + 2 + 32; // 133

    if body.len() <= VAA_BODY_HEADER_LEN {
        return Err(GlobalAccountantError::InvalidInstructionData);
    }
    let payload = &body[VAA_BODY_HEADER_LEN..];
    let action = payload[0];
    match action {
        ACTION_TRANSFER | ACTION_TRANSFER_WITH_PAYLOAD => {
            if payload.len() < TRANSFER_PAYLOAD_MIN {
                return Err(GlobalAccountantError::InvalidInstructionData);
            }
            let mut amount = [0u8; 32];
            amount.copy_from_slice(&payload[1..33]);
            let mut token_address = [0u8; 32];
            token_address.copy_from_slice(&payload[33..65]);
            let token_chain = u16::from_be_bytes([payload[65], payload[66]]);
            // payload[67..99] is recipient — ignored.
            let recipient_chain = u16::from_be_bytes([payload[99], payload[100]]);
            Ok(TokenBridgeAction::Transfer {
                amount: Uint256::from_be_bytes(amount),
                token_chain,
                token_address,
                recipient_chain,
            })
        }
        ACTION_ATTEST => Ok(TokenBridgeAction::Attest),
        _ => Ok(TokenBridgeAction::Other),
    }
}

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

    // ---- parse_token_bridge_payload tests ----

    /// Build a 184-byte VAA body (51-byte header + 133-byte transfer payload)
    /// in a stack array.
    fn transfer_body(
        action: u8,
        amount: u128,
        token_address: [u8; 32],
        token_chain: u16,
        recipient_chain: u16,
    ) -> [u8; 184] {
        let mut body = [0u8; 184];
        // Header is zeroed; transfer payload starts at offset 51.
        body[51] = action;
        // amount: 32-byte BE, low 16 bytes hold the u128.
        body[52 + 16..52 + 32].copy_from_slice(&amount.to_be_bytes());
        body[84..116].copy_from_slice(&token_address);
        body[116..118].copy_from_slice(&token_chain.to_be_bytes());
        // recipient (118..150): recognisable bytes to catch off-by-one.
        body[118] = 0xAB;
        body[149] = 0xCD;
        body[150..152].copy_from_slice(&recipient_chain.to_be_bytes());
        body
    }

    #[test]
    fn parse_vaa_body_header_decodes_routing_tuple() {
        let mut body = [0u8; VAA_BODY_HEADER_LEN];
        body[8..10].copy_from_slice(&2u16.to_be_bytes());
        body[10] = 0xAA;
        body[41] = 0xBB;
        body[42..50].copy_from_slice(&0x0102_0304_0506_0708u64.to_be_bytes());

        let header = parse_vaa_body_header(&body).unwrap();
        assert_eq!(header.chain, 2);
        assert_eq!(header.emitter[0], 0xAA);
        assert_eq!(header.emitter[31], 0xBB);
        assert_eq!(header.sequence, 0x0102_0304_0506_0708);
    }

    #[test]
    fn parse_vaa_body_header_accepts_exact_header_len() {
        assert!(parse_vaa_body_header(&[0u8; VAA_BODY_HEADER_LEN]).is_ok());
    }

    #[test]
    fn parse_vaa_body_header_short_body_rejects() {
        assert_eq!(
            parse_vaa_body_header(&[0u8; VAA_BODY_HEADER_LEN - 1]),
            Err(GlobalAccountantError::InvalidInstructionData)
        );
    }

    #[test]
    fn parse_token_bridge_payload_transfer_decodes_amount_token_recipient() {
        let mut token_address = [0u8; 32];
        token_address[0] = 0x11;
        token_address[31] = 0x99;
        let body = transfer_body(0x01, 1_000_000_u128, token_address, 2, 10);
        let action = parse_token_bridge_payload(&body).expect("transfer parses");
        match action {
            TokenBridgeAction::Transfer {
                amount,
                token_chain,
                token_address: ta,
                recipient_chain,
            } => {
                assert_eq!(amount, Uint256::from_u128(1_000_000));
                assert_eq!(token_chain, 2);
                assert_eq!(ta, token_address);
                assert_eq!(recipient_chain, 10);
            }
            other => panic!("expected Transfer, got {other:?}"),
        }
    }

    #[test]
    fn parse_token_bridge_payload_transfer_with_payload_same_as_transfer() {
        // Action 0x03 must decode to the same Transfer variant as 0x01.
        let token_address = [0x42u8; 32];
        let body_01 = transfer_body(0x01, 99, token_address, 5, 7);
        let body_03 = transfer_body(0x03, 99, token_address, 5, 7);
        let a = parse_token_bridge_payload(&body_01).unwrap();
        let b = parse_token_bridge_payload(&body_03).unwrap();
        assert_eq!(a, b, "action 0x01 and 0x03 must decode identically");
    }

    #[test]
    fn parse_token_bridge_payload_attest() {
        // Action 0x02 only needs the one-byte action past the 51-byte header.
        let mut body = [0u8; 52];
        body[51] = 0x02;
        let action = parse_token_bridge_payload(&body).expect("attest parses");
        assert_eq!(action, TokenBridgeAction::Attest);
    }

    #[test]
    fn parse_token_bridge_payload_unknown_action() {
        // Any byte other than 0x01/0x02/0x03 decodes to Other.
        let mut body = [0u8; 52];
        body[51] = 0x77;
        let action = parse_token_bridge_payload(&body).expect("unknown action parses");
        assert_eq!(action, TokenBridgeAction::Other);
    }

    #[test]
    fn parse_token_bridge_payload_short_body_rejects() {
        // 51-byte body (no action byte) must reject.
        let body = [0u8; 51];
        let err = parse_token_bridge_payload(&body).unwrap_err();
        assert_eq!(err, GlobalAccountantError::InvalidInstructionData);
    }

    #[test]
    fn parse_token_bridge_payload_short_transfer_payload_rejects() {
        // Header + action 0x01 + 10 bytes — short of the 133-byte minimum.
        let mut body = [0u8; 62];
        body[51] = 0x01;
        let err = parse_token_bridge_payload(&body).unwrap_err();
        assert_eq!(err, GlobalAccountantError::InvalidInstructionData);
    }
}
