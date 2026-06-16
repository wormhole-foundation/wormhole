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
#[repr(u8)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum Instruction {
    SubmitObservations = 0,
    ClosePending = 1,
    /// Permissionless signed-VAA backfill via the Verify VAA Shim CPI; applies
    /// balance effects directly, bypassing the quorum tracker.
    SubmitVaas = 2,
    /// Token Bridge `RegisterChain` governance handler: verifies the VAA and
    /// initialises/updates the canonical `ChainRegistration` PDA.
    RegisterChain = 3,
    /// Accountant `ModifyBalance` governance handler: verifies the VAA and
    /// applies an Add/Subtract delta to the canonical `BalanceAccount` PDA.
    ModifyBalance = 4,
}

impl Instruction {
    pub const fn from_u8(value: u8) -> Option<Self> {
        match value {
            0 => Some(Self::SubmitObservations),
            1 => Some(Self::ClosePending),
            2 => Some(Self::SubmitVaas),
            3 => Some(Self::RegisterChain),
            4 => Some(Self::ModifyBalance),
            _ => None,
        }
    }
}

/// Custom error codes returned via `ProgramError::Custom(u32)`. Stable across
/// program versions; do not renumber.
#[repr(u32)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum GlobalAccountantError {
    InvalidInstruction = 0,
    InvalidInstructionData = 1,
    InvalidPda = 2,
    DigestMismatch = 3,
    PayerMismatch = 4,
    /// Reserved slot; never raised. Kept to preserve ABI numbering.
    NotImplemented = 5,
    /// The instruction was feature-gated off in this build (keeps
    /// `test_only_open_digest` out of production).
    NotEnabled = 6,
    /// `(chain, emitter, sequence)` already marked accounted-for in NoReplay.
    AlreadyAccounted = 7,
    /// Retired — never raised. `pinocchio::cpi::invoke_signed` only surfaces
    /// pre-CPI validation errors via `Result`; an inner-program failure aborts
    /// THIS program directly via the SBF runtime, bypassing any `map_err`.
    /// Race-loss `AccountAlreadyInitialized` from `MarkUsed` propagates as
    /// itself, not as this code. Kept so error-code numbering stays stable.
    NoReplayCpiFailed = 8,
    /// Signature failed `secp256k1_recover`, or the recovered pubkey did not
    /// match the `guardian_index` in the GuardianSet PDA.
    InvalidSignature = 9,
    /// `guardian_index` out of bounds for the guardian set.
    InvalidGuardianIndex = 10,
    /// The guardian's bit is already set in the pending bitmap.
    AlreadySigned = 11,
    /// Observation references a guardian set older than the one the pending PDA
    /// is accumulating against (stale observation after rotation).
    StaleGuardianSet = 12,
    /// Retired — no longer emitted. Kept so error-code numbering stays stable.
    DigestForgery = 13,
    /// `close_pending` triggers unmet: recorded guardian set still active AND
    /// NoReplay does not mark the entry accounted-for.
    CannotCleanup = 14,
    /// Balance overflow on the transfer path (`lock_or_burn` /
    /// `unlock_or_mint`).
    BalanceOverflow = 15,
    /// Balance underflow on the transfer path (insufficient source balance).
    BalanceUnderflow = 16,
    /// `keccak256(keccak256(body)) != digest`. Rejected before any mutation
    /// since the body carries the transfer payload the commit branch reads.
    BodyDigestMismatch = 17,
    /// Supplied Account PDA does not match the canonical seeds for the
    /// source/destination side of the transfer.
    InvalidAccountPda = 18,
    /// No `ChainRegistration` PDA for the body's `emitter_chain` — wait for the
    /// Token Bridge `RegisterChain` VAA before observations are accepted.
    MissingChainRegistration = 19,
    /// Registration PDA exists but its `emitter_address` does not match the
    /// body header's emitter.
    UnregisteredEmitter = 20,
    /// `register_chain` body did not come from the governance emitter
    /// `(chain=1, GOVERNANCE_EMITTER)`.
    InvalidGovernanceEmitter = 21,
    /// `register_chain` payload module is not `TOKEN_BRIDGE_GOVERNANCE_MODULE`.
    InvalidGovernanceModule = 22,
    /// `register_chain` payload action byte is not `0x01` (RegisterChain).
    InvalidGovernanceAction = 23,
    /// `register_chain` target chain is neither `0x0000` (Any) nor Wormchain.
    GovernanceChainMismatch = 24,
    /// `modify_balance` `kind` byte is neither `1` (Add) nor `2` (Subtract).
    InvalidModificationKind = 25,
    /// `modify_balance` `Add` overflow. Distinct from `BalanceOverflow` so logs
    /// disambiguate the entrypoint.
    ModifyBalanceOverflow = 26,
    /// `modify_balance` `Subtract` underflow (also raised when subtracting from
    /// an uninitialised PDA, rejected before allocation).
    ModifyBalanceUnderflow = 27,
    /// A `ModificationLog` PDA already exists at `(b"modification", sequence)`.
    /// Replay protection keyed on the payload's modification sequence.
    DuplicateModification = 28,
    /// Token Bridge payload action is not `0x01`/`0x02`/`0x03`. Rejecting
    /// (rather than committing) leaves the NoReplay slot unconsumed so a future
    /// upgrade can process the VAA.
    UnknownTokenBridgePayload = 29,
}

impl From<GlobalAccountantError> for u32 {
    fn from(e: GlobalAccountantError) -> Self {
        e as u32
    }
}

/// 8-byte tag prefixing every accountant commit log entry. Off-chain indexers
/// filter program logs for this prefix to find the canonical
/// `(chain, emitter, sequence, digest, guardian_set_index)` record emitted on
/// the quorum-completing branch of `submit_observations` and on every
/// successful `submit_vaas`.
///
/// The log payload layout (86 bytes total) is:
///
/// | offset | size | field                              |
/// |--------|------|------------------------------------|
/// | 0      | 8    | `ACCOUNTANT_DIGEST_LOG_TAG`        |
/// | 8      | 2    | chain (big endian)                 |
/// | 10     | 32   | emitter                            |
/// | 42     | 8    | sequence (big endian)              |
/// | 50     | 32   | digest                             |
/// | 82     | 4    | guardian_set_index (little endian) |
///
/// `guardian_set_index` is the set that reached quorum on the observations
/// path; `submit_vaas` records the sentinel `0` (the Shim accepts any
/// currently-active set, so no single index meaningfully describes the
/// authorisation).
pub const ACCOUNTANT_DIGEST_LOG_TAG: [u8; 8] = *b"ACCDGST\0";

/// Total byte length of an emitted commit log entry (tag + payload).
pub const ACCOUNTANT_DIGEST_LOG_LEN: usize = 8 + 2 + 32 + 8 + 32 + 4;

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

/// PDA seed prefix for [`ModificationLogLayout`]. Full tuple:
/// `(b"modification", sequence_be)`. Existence of this PDA enforces replay
/// protection on the governance path.
pub const MODIFICATION_SEED_PREFIX: &[u8] = b"modification";

/// Wormhole governance emitter — `chain = 1 (Solana)`, `address = [0; 31] ||
/// 0x04`. `register_chain` only accepts governance VAAs signed by this emitter.
pub const GOVERNANCE_EMITTER: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04,
];

/// Wormhole chain ID for Solana, also stamped on the governance emitter pair.
pub const SOLANA_CHAIN_ID: u16 = 1;

/// Wormhole chain ID for Wormchain. `register_chain` governance VAAs must
/// target either chain `0x0000` (Any) or this.
pub const WORMCHAIN_CHAIN_ID: u16 = 3104;

/// Token Bridge governance module — first 32 bytes of a Token Bridge
/// governance payload. "TokenBridge" right-aligned in 32 bytes.
pub const TOKEN_BRIDGE_GOVERNANCE_MODULE: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, b'T', b'o', b'k', b'e', b'n', b'B', b'r', b'i', b'd', b'g', b'e',
];

/// Token Bridge governance `RegisterChain` action byte.
pub const REGISTER_CHAIN_ACTION: u8 = 0x01;

/// Accountant governance module — first 32 bytes of a `ModifyBalance` payload.
/// "GlobalAccountant" right-aligned in 32 bytes. The action byte `0x01`
/// overlaps RegisterChain, so the module is what disambiguates the flows.
pub const ACCOUNTANT_GOVERNANCE_MODULE: [u8; 32] = [
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    b'G', b'l', b'o', b'b', b'a', b'l', b'A', b'c', b'c', b'o', b'u', b'n', b't', b'a', b'n', b't',
];

/// Accountant governance `ModifyBalance` action byte.
pub const MODIFY_BALANCE_ACTION: u8 = 0x01;

/// Compute-unit ceiling for the hottest `submit_observations` / `submit_vaas`
/// path (quorum commit branch with a Transfer + lazy-init of both Account
/// PDAs). Pinned by a regression test in `tests/submit_observations.rs`.
pub const MAX_QUORUM_BRANCH_CU: u64 = 80_000;

/// `ModifyBalance` payload `kind` byte values. Any byte other than `Add` or
/// `Subtract` is rejected as `InvalidModificationKind`.
#[repr(u8)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum ModificationKind {
    Add = 1,
    Subtract = 2,
}

impl ModificationKind {
    pub const fn from_u8(value: u8) -> Option<Self> {
        match value {
            1 => Some(Self::Add),
            2 => Some(Self::Subtract),
            _ => None,
        }
    }
}

/// Account-type tag stored at offset 0 of every program-owned PDA layout.
///
/// Seeds namespace writes on-chain but are not recoverable from
/// `getProgramAccounts`, so off-chain consumers discriminate account types with
/// a single `memcmp(offset 0, [tag])` filter. On-chain, the load helpers compare
/// the tag as defense-in-depth against a handler that forgets to re-derive a PDA.
///
/// Values are append-only (same discipline as [`GlobalAccountantError`]); never
/// renumber once shipped. `0` is reserved: a freshly allocated account is all
/// zeroes, so zeroed data must never parse as a valid tag. The NTT accountant
/// port extends this space (4+).
#[repr(u8)]
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum AccountTag {
    Pending = 1,
    Balance = 2,
    ChainRegistration = 3,
    ModificationLog = 4,
}

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

/// Discriminator for `solana-noreplay`'s `MarkUsedBulk`. Wire format:
/// `[disc: u8][namespace_len: u16 LE][namespace: ≤64 B][bucket_index: u64 LE]
///  [or_mask: 128 B]`. Semantics: `bitmap |= or_mask` (OR-only; never clears
/// bits). Used by the backfill program to flip many bits per CPI, dropping
/// per-entry CU from ~3,000 to ~80 in the dense-bucket case.
///
/// Allocated as the next free byte in the noreplay program's dispatch table
/// after `CreateBitmap=0`, `MarkUsed=1`, `UnmarkUsed=2`.
pub const NOREPLAY_MARK_USED_BULK_DISCRIMINATOR: u8 = 3;

/// Bits per bitmap bucket. Bucket index is `sequence / BITS_PER_BUCKET`, bit
/// offset is `sequence % BITS_PER_BUCKET`.
pub const NOREPLAY_BITS_PER_BUCKET: u64 = 1024;

/// Bitmap payload size inside a noreplay PDA (account is 1-byte bump + bitmap).
pub const NOREPLAY_BITMAP_BYTES: usize = 128;

/// Byte offset of the bitmap payload (byte 0 is the stored canonical bump).
pub const NOREPLAY_BITMAP_OFFSET: usize = 1;

/// Wormhole Core Bridge program ID on Solana mainnet
/// (`worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth`). Raw bytes to keep this
/// crate Solana-SDK-free. Used by `close_pending` to verify the `GuardianSet`
/// account is Core-Bridge-owned before reading it — otherwise a forged
/// "expired" set could permanently DoS a pending PDA.
pub const CORE_BRIDGE_PROGRAM_ID: Pubkey = [
    0x0e, 0x0a, 0x58, 0x9a, 0x41, 0xa5, 0x5f, 0xbd, 0x66, 0xc5, 0x2a, 0x47, 0x5f, 0x2d, 0x92, 0xa6,
    0xd3, 0xdc, 0x9b, 0x47, 0x47, 0x11, 0x4c, 0xb9, 0xaf, 0x82, 0x5a, 0x98, 0xb5, 0x45, 0xd3, 0xce,
];

/// Verify VAA Shim program ID (`EFaNWErqAtVWufdNb7yofSHHfWFos843DFpu4JBw24at`),
/// same address on mainnet, devnet, and Tilt localnet. Raw bytes to keep this
/// crate Solana-SDK-free.
pub const VERIFY_VAA_SHIM_PROGRAM_ID: Pubkey = [
    196, 227, 203, 55, 17, 156, 166, 124, 168, 35, 28, 170, 3, 131, 164, 140, 195, 254, 137, 233,
    101, 80, 83, 225, 249, 25, 254, 66, 226, 131, 254, 161,
];

/// Anchor discriminator for the Verify VAA Shim's `verify_hash` — first 8 bytes
/// of `sha256("global:verify_hash")`.
pub const VERIFY_HASH_SELECTOR: [u8; 8] = [22, 152, 160, 69, 241, 148, 14, 124];

/// Wire size of `verify_hash` instruction data: 8-byte selector + 1-byte
/// guardian-set bump + 32-byte digest.
pub const VERIFY_HASH_DATA_LEN: usize = 8 + 1 + 32;

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

/// Zero-copy layout for a per-`(chain, emitter, sequence)` pending-quorum PDA.
/// On-disk size is **76 bytes** (4-byte alignment, tag at offset 0).
///
/// | offset | size | field              |
/// |--------|------|--------------------|
/// | 0      | 1    | tag ([`AccountTag::Pending`]) |
/// | 1      | 1    | _pad0              |
/// | 2      | 2    | chain              |
/// | 4      | 4    | guardian_set_index |
/// | 8      | 4    | signatures (u32 bitmap; bit N == guardian-index N signed) |
/// | 12     | 32   | digest             |
/// | 44     | 32   | payer              |
///
/// The 32-bit bitmap covers 32 guardian indices. A protocol move to >32
/// guardians requires widening the field and bumping the layout version.
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct PendingObservationsLayout {
    /// Account-type tag; always [`AccountTag::Pending`]. See [`Self::TAG`].
    pub tag: u8,
    /// Alignment padding; crate-private so callers go through `Zeroable`.
    pub(crate) _pad0: u8,
    pub chain: u16,
    pub guardian_set_index: u32,
    pub signatures: u32,
    pub digest: [u8; 32],
    pub payer: Pubkey,
}

impl PendingObservationsLayout {
    /// Byte length of the layout (also the rent-paying allocation size).
    pub const LEN: usize = core::mem::size_of::<Self>();

    /// Account-type tag stamped at offset 0. See [`AccountTag`].
    pub const TAG: u8 = AccountTag::Pending as u8;

    /// Quorum threshold: 13 of 19 guardians — the Core Bridge
    /// `(len * 2) / 3 + 1` for len 19. Pinned, not derived from the live set.
    pub const QUORUM_THRESHOLD: u32 = 13;
}

const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(PendingObservationsLayout, tag) == 0);
    assert!(offset_of!(PendingObservationsLayout, chain) == 2);
    assert!(offset_of!(PendingObservationsLayout, guardian_set_index) == 4);
    assert!(offset_of!(PendingObservationsLayout, signatures) == 8);
    assert!(offset_of!(PendingObservationsLayout, digest) == 12);
    assert!(offset_of!(PendingObservationsLayout, payer) == 44);
    assert!(PendingObservationsLayout::LEN == 76);
};

/// Zero-copy balance account for a `(chain, token_chain, token_address)`
/// triple. On-disk size is **70 bytes** (tag at offset 0).
///
/// | offset | size | field         |
/// |--------|------|---------------|
/// | 0      | 1    | tag ([`AccountTag::Balance`]) |
/// | 1      | 1    | _pad0         |
/// | 2      | 2    | chain         |
/// | 4      | 2    | token_chain   |
/// | 6      | 32   | token_address |
/// | 38     | 32   | balance       |
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct BalanceAccountLayout {
    /// Account-type tag; always [`AccountTag::Balance`]. See [`Self::TAG`].
    pub tag: u8,
    /// Alignment padding; crate-private so callers go through `Zeroable`.
    pub(crate) _pad0: u8,
    /// Chain on which this balance is held.
    pub chain: u16,
    /// Native chain of the token.
    pub token_chain: u16,
    /// Token address on its native chain.
    pub token_address: [u8; 32],
    /// Current balance, 32-byte big-endian (matches the VAA `amount` encoding).
    pub balance: Uint256,
}

impl BalanceAccountLayout {
    pub const LEN: usize = core::mem::size_of::<Self>();

    /// Account-type tag stamped at offset 0. See [`AccountTag`].
    pub const TAG: u8 = AccountTag::Balance as u8;

    /// Apply a `lock_or_burn`: credits when `chain == token_chain` (native
    /// lock), debits otherwise (wrapped burn). Overflow/underflow surface as
    /// `BalanceOverflow`/`BalanceUnderflow`.
    pub fn lock_or_burn(&mut self, amount: Uint256) -> Result<(), GlobalAccountantError> {
        if self.chain == self.token_chain {
            self.balance = self
                .balance
                .checked_add(amount)
                .ok_or(GlobalAccountantError::BalanceOverflow)?;
        } else {
            self.balance = self
                .balance
                .checked_sub(amount)
                .ok_or(GlobalAccountantError::BalanceUnderflow)?;
        }
        Ok(())
    }

    /// Apply an `unlock_or_mint`: debits when `chain == token_chain` (native
    /// unlock), credits otherwise (wrapped mint). Symmetric to
    /// [`Self::lock_or_burn`].
    pub fn unlock_or_mint(&mut self, amount: Uint256) -> Result<(), GlobalAccountantError> {
        if self.chain == self.token_chain {
            self.balance = self
                .balance
                .checked_sub(amount)
                .ok_or(GlobalAccountantError::BalanceUnderflow)?;
        } else {
            self.balance = self
                .balance
                .checked_add(amount)
                .ok_or(GlobalAccountantError::BalanceOverflow)?;
        }
        Ok(())
    }

    /// Raw `balance += amount` for the governance `modify_balance` path (no
    /// native/wrapped dispatch). Overflow surfaces as `ModifyBalanceOverflow`.
    pub fn raw_add(&mut self, amount: Uint256) -> Result<(), GlobalAccountantError> {
        self.balance = self
            .balance
            .checked_add(amount)
            .ok_or(GlobalAccountantError::ModifyBalanceOverflow)?;
        Ok(())
    }

    /// Raw `balance -= amount` for the governance `modify_balance` path.
    /// Underflow surfaces as `ModifyBalanceUnderflow`.
    pub fn raw_sub(&mut self, amount: Uint256) -> Result<(), GlobalAccountantError> {
        self.balance = self
            .balance
            .checked_sub(amount)
            .ok_or(GlobalAccountantError::ModifyBalanceUnderflow)?;
        Ok(())
    }
}

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

// Compile-time pins for the balance layout.
const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(BalanceAccountLayout, tag) == 0);
    assert!(offset_of!(BalanceAccountLayout, chain) == 2);
    assert!(offset_of!(BalanceAccountLayout, token_chain) == 4);
    assert!(offset_of!(BalanceAccountLayout, token_address) == 6);
    assert!(offset_of!(BalanceAccountLayout, balance) == 38);
    assert!(BalanceAccountLayout::LEN == 70);
};

/// Zero-copy per-chain Token Bridge emitter registration. One PDA per chain at
/// `(b"chain_registration", chain_be)`, holding the canonical emitter address.
/// Written only by `register_chain`; re-registration with a higher-sequence VAA
/// overwrites the emitter (supports emitter rotation).
///
/// | offset | size | field           |
/// |--------|------|-----------------|
/// | 0      | 1    | tag ([`AccountTag::ChainRegistration`]) |
/// | 1      | 1    | _pad0           |
/// | 2      | 2    | chain           |
/// | 4      | 28   | _padding        |
/// | 32     | 32   | emitter_address |
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct ChainRegistrationLayout {
    /// Account-type tag; always [`AccountTag::ChainRegistration`]. See [`Self::TAG`].
    pub tag: u8,
    /// Alignment padding; crate-private so callers go through `Zeroable`.
    pub(crate) _pad0: u8,
    /// Wormhole chain ID this PDA registers (mirrors the seed bytes).
    pub chain: u16,
    /// Reserved; crate-private so callers go through `Zeroable`.
    pub(crate) _padding: [u8; 28],
    /// Canonical Token Bridge emitter address on `chain`.
    pub emitter_address: [u8; 32],
}

impl ChainRegistrationLayout {
    pub const LEN: usize = core::mem::size_of::<Self>();

    /// Account-type tag stamped at offset 0. See [`AccountTag`].
    pub const TAG: u8 = AccountTag::ChainRegistration as u8;
}

const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(ChainRegistrationLayout, tag) == 0);
    assert!(offset_of!(ChainRegistrationLayout, chain) == 2);
    assert!(offset_of!(ChainRegistrationLayout, _padding) == 4);
    assert!(offset_of!(ChainRegistrationLayout, emitter_address) == 32);
    assert!(ChainRegistrationLayout::LEN == 64);
};

/// Zero-copy per-modification audit-log PDA. Each `modify_balance` lazy-inits
/// one PDA at `(b"modification", payload_sequence_be)`. A second VAA with the
/// same sequence collides on this address and is rejected with
/// `DuplicateModification` — this is the governance-path replay protection.
///
/// | offset | size | field         |
/// |--------|------|---------------|
/// | 0      | 1    | tag ([`AccountTag::ModificationLog`]) |
/// | 1      | 1    | kind          |
/// | 2      | 2    | chain_id      |
/// | 4      | 2    | token_chain   |
/// | 6      | 2    | _pad0         |
/// | 8      | 8    | sequence      |
/// | 16     | 32   | token_address |
/// | 48     | 32   | amount        |
/// | 80     | 32   | reason        |
///
/// Total: 112 bytes (multiple of 8 for `Pod` alignment). Small fields are
/// clustered ahead of the 8-aligned `sequence` so the tag sits at offset 0.
#[repr(C)]
#[derive(Clone, Copy, Debug, Eq, PartialEq, Pod, Zeroable)]
pub struct ModificationLogLayout {
    /// Account-type tag; always [`AccountTag::ModificationLog`]. See [`Self::TAG`].
    pub tag: u8,
    /// `1` for `Add`, `2` for `Subtract` (see [`ModificationKind`]).
    pub kind: u8,
    /// Chain whose balance was modified.
    pub chain_id: u16,
    /// Native chain of the modified token.
    pub token_chain: u16,
    /// Alignment padding ahead of `sequence`; crate-private so callers go
    /// through `Zeroable`.
    pub(crate) _pad0: [u8; 2],
    /// Modification's own sequence (distinct from the VAA emitter sequence).
    pub sequence: u64,
    /// Token address on its native chain.
    pub token_address: [u8; 32],
    /// Modification amount, big-endian 256-bit unsigned integer.
    pub amount: Uint256,
    /// Free-form reason, 32-byte right-padded ASCII. Audit-trail only.
    pub reason: [u8; 32],
}

impl ModificationLogLayout {
    pub const LEN: usize = core::mem::size_of::<Self>();

    /// Account-type tag stamped at offset 0. See [`AccountTag`].
    pub const TAG: u8 = AccountTag::ModificationLog as u8;
}

const _: () = {
    use core::mem::offset_of;
    assert!(offset_of!(ModificationLogLayout, tag) == 0);
    assert!(offset_of!(ModificationLogLayout, kind) == 1);
    assert!(offset_of!(ModificationLogLayout, chain_id) == 2);
    assert!(offset_of!(ModificationLogLayout, token_chain) == 4);
    assert!(offset_of!(ModificationLogLayout, sequence) == 8);
    assert!(offset_of!(ModificationLogLayout, token_address) == 16);
    assert!(offset_of!(ModificationLogLayout, amount) == 48);
    assert!(offset_of!(ModificationLogLayout, reason) == 80);
    assert!(ModificationLogLayout::LEN == 112);
};

#[cfg(test)]
mod tests {
    use super::*;

    // ---- AccountTag retrofit tests ----

    #[test]
    fn account_tag_values_pinned() {
        // Append-only: never renumber once shipped.
        assert_eq!(AccountTag::Pending as u8, 1);
        assert_eq!(AccountTag::Balance as u8, 2);
        assert_eq!(AccountTag::ChainRegistration as u8, 3);
        assert_eq!(AccountTag::ModificationLog as u8, 4);
        // Each layout's TAG const mirrors its AccountTag value.
        assert_eq!(PendingObservationsLayout::TAG, AccountTag::Pending as u8);
        assert_eq!(BalanceAccountLayout::TAG, AccountTag::Balance as u8);
        assert_eq!(
            ChainRegistrationLayout::TAG,
            AccountTag::ChainRegistration as u8
        );
        assert_eq!(
            ModificationLogLayout::TAG,
            AccountTag::ModificationLog as u8
        );
    }

    #[test]
    fn tag_at_offset_zero_for_all_layouts() {
        use core::mem::offset_of;
        assert_eq!(offset_of!(PendingObservationsLayout, tag), 0);
        assert_eq!(offset_of!(BalanceAccountLayout, tag), 0);
        assert_eq!(offset_of!(ChainRegistrationLayout, tag), 0);
        assert_eq!(offset_of!(ModificationLogLayout, tag), 0);
    }

    #[test]
    fn zeroed_layout_is_not_a_valid_tag() {
        // A freshly allocated (all-zero) account must not parse as any type:
        // tag 0 is reserved, distinct from every AccountTag value.
        assert_eq!(<PendingObservationsLayout as Zeroable>::zeroed().tag, 0);
        assert_eq!(<BalanceAccountLayout as Zeroable>::zeroed().tag, 0);
        assert_eq!(<ChainRegistrationLayout as Zeroable>::zeroed().tag, 0);
        assert_eq!(<ModificationLogLayout as Zeroable>::zeroed().tag, 0);
        for tag in [
            AccountTag::Pending,
            AccountTag::Balance,
            AccountTag::ChainRegistration,
            AccountTag::ModificationLog,
        ] {
            assert_ne!(tag as u8, 0);
        }
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
    fn balance_layout_size_pinned() {
        // 70 bytes — tag (1) + _pad0 (1) + chain (2) + token_chain (2) +
        // token_address (32) + balance (32). Grew from 68 to 70 when the
        // offset-0 account tag was retrofitted.
        assert_eq!(BalanceAccountLayout::LEN, 70);
    }

    #[test]
    fn balance_layout_uint256_offsets_pinned() {
        // Runtime mirror of the const-assert block above.
        use core::mem::offset_of;
        assert_eq!(offset_of!(BalanceAccountLayout, tag), 0);
        assert_eq!(offset_of!(BalanceAccountLayout, chain), 2);
        assert_eq!(offset_of!(BalanceAccountLayout, token_chain), 4);
        assert_eq!(offset_of!(BalanceAccountLayout, token_address), 6);
        assert_eq!(offset_of!(BalanceAccountLayout, balance), 38);
    }

    #[test]
    fn balance_layout_is_pod_friendly() {
        let mut token_address = [0u8; 32];
        for (i, b) in token_address.iter_mut().enumerate() {
            *b = i as u8;
        }
        let original = BalanceAccountLayout {
            tag: BalanceAccountLayout::TAG,
            _pad0: 0,
            chain: 1,
            token_chain: 2,
            token_address,
            balance: Uint256::from_u128(0xcafe_babe),
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
        assert_eq!(offset_of!(PendingObservationsLayout, tag), 0);
        assert_eq!(offset_of!(PendingObservationsLayout, chain), 2);
        assert_eq!(offset_of!(PendingObservationsLayout, guardian_set_index), 4);
        assert_eq!(offset_of!(PendingObservationsLayout, signatures), 8);
        assert_eq!(offset_of!(PendingObservationsLayout, digest), 12);
        assert_eq!(offset_of!(PendingObservationsLayout, payer), 44);
    }

    #[test]
    fn pending_layout_is_pod_friendly() {
        let mut digest = [0u8; 32];
        for (i, b) in digest.iter_mut().enumerate() {
            *b = i as u8;
        }
        let original = PendingObservationsLayout {
            tag: PendingObservationsLayout::TAG,
            _pad0: 0,
            chain: 1,
            guardian_set_index: 0x0BAD_CAFE,
            signatures: 0x0000_1FFFu32, // 13 low bits set
            digest,
            payer: [0xAA; 32],
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
            tag: BalanceAccountLayout::TAG,
            _pad0: 0,
            chain: 0,
            token_chain: 0,
            token_address: [0u8; 32],
            balance: Uint256::from_u128(0x1234_5678),
        };
        let bytes = bytemuck::bytes_of(&original);
        let balance_slice = &bytes[38..70];
        let mut expected = [0u8; 32];
        expected[28] = 0x12;
        expected[29] = 0x34;
        expected[30] = 0x56;
        expected[31] = 0x78;
        assert_eq!(balance_slice, &expected);
    }

    // ---- lock_or_burn / unlock_or_mint tests ----

    fn balance_with(chain: u16, token_chain: u16, balance: Uint256) -> BalanceAccountLayout {
        // Recognisable token-address pattern; irrelevant to the arithmetic.
        let mut token_address = [0u8; 32];
        token_address[0] = 0x62;
        token_address[31] = 0x61;
        BalanceAccountLayout {
            tag: BalanceAccountLayout::TAG,
            _pad0: 0,
            chain,
            token_chain,
            token_address,
            balance,
        }
    }

    #[test]
    fn lock_or_burn_native_chain_credits() {
        // chain == token_chain ⇒ native-side credit (500 + 200 = 700).
        let mut acc = balance_with(0xbae2, 0xbae2, Uint256::from_u128(500));
        acc.lock_or_burn(Uint256::from_u128(200)).unwrap();
        assert_eq!(acc.balance, Uint256::from_u128(700));
    }

    #[test]
    fn lock_or_burn_wrapped_chain_debits() {
        // chain != token_chain ⇒ wrapped-side debit (500 - 200 = 300).
        let mut acc = balance_with(0xcae8, 0xbae2, Uint256::from_u128(500));
        acc.lock_or_burn(Uint256::from_u128(200)).unwrap();
        assert_eq!(acc.balance, Uint256::from_u128(300));
    }

    #[test]
    fn lock_or_burn_wrapped_chain_underflow_rejects() {
        // Underflow ⇒ BalanceUnderflow.
        let mut acc = balance_with(0xcae8, 0xbae2, Uint256::ZERO);
        let err = acc.lock_or_burn(Uint256::from_u128(200)).unwrap_err();
        assert_eq!(err, GlobalAccountantError::BalanceUnderflow);
        assert_eq!(acc.balance, Uint256::ZERO, "balance unchanged on error");
    }

    #[test]
    fn lock_or_burn_native_chain_overflow_rejects() {
        // Overflow ⇒ BalanceOverflow.
        let mut acc = balance_with(0xbae2, 0xbae2, Uint256::MAX);
        let err = acc.lock_or_burn(Uint256::from_u128(200)).unwrap_err();
        assert_eq!(err, GlobalAccountantError::BalanceOverflow);
        assert_eq!(acc.balance, Uint256::MAX, "balance unchanged on error");
    }

    #[test]
    fn unlock_or_mint_native_chain_debits() {
        // chain == token_chain ⇒ native-side debit (500 - 200 = 300).
        let mut acc = balance_with(0xbae2, 0xbae2, Uint256::from_u128(500));
        acc.unlock_or_mint(Uint256::from_u128(200)).unwrap();
        assert_eq!(acc.balance, Uint256::from_u128(300));
    }

    #[test]
    fn unlock_or_mint_native_chain_underflow_rejects() {
        // Underflow ⇒ BalanceUnderflow.
        let mut acc = balance_with(0xbae2, 0xbae2, Uint256::ZERO);
        let err = acc.unlock_or_mint(Uint256::from_u128(200)).unwrap_err();
        assert_eq!(err, GlobalAccountantError::BalanceUnderflow);
        assert_eq!(acc.balance, Uint256::ZERO);
    }

    #[test]
    fn unlock_or_mint_wrapped_chain_credits() {
        // chain != token_chain ⇒ wrapped-side credit (500 + 200 = 700).
        let mut acc = balance_with(0xcae8, 0xbae2, Uint256::from_u128(500));
        acc.unlock_or_mint(Uint256::from_u128(200)).unwrap();
        assert_eq!(acc.balance, Uint256::from_u128(700));
    }

    #[test]
    fn unlock_or_mint_wrapped_chain_overflow_rejects() {
        // Overflow ⇒ BalanceOverflow.
        let mut acc = balance_with(0xcae8, 0xbae2, Uint256::MAX);
        let err = acc.unlock_or_mint(Uint256::from_u128(200)).unwrap_err();
        assert_eq!(err, GlobalAccountantError::BalanceOverflow);
        assert_eq!(acc.balance, Uint256::MAX);
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
