//! Integration tests for `submit_observations`.
//!
//! Driven against a Mollusk instance with the real `solana_noreplay.so` loaded
//! at the canonical program ID (see `common::mollusk_fixtures`).

#![allow(clippy::too_many_arguments)]

use {
    global_accountant_definitions::{
        BalanceAccountLayout, ChainRegistrationLayout, GlobalAccountantError,
        Instruction as IxDiscriminator, PendingObservationsLayout, Uint256, ACCOUNT_SEED_PREFIX,
        CHAIN_REGISTRATION_SEED_PREFIX, CORE_BRIDGE_PROGRAM_ID, MAX_QUORUM_BRANCH_CU,
        NOREPLAY_AUTHORITY_SEED_PREFIX, NOREPLAY_BITMAP_BYTES, NOREPLAY_BITMAP_OFFSET,
        NOREPLAY_BITS_PER_BUCKET, NOREPLAY_PROGRAM_ID, PENDING_SEED_PREFIX,
    },
    libsecp256k1::{sign, Message, PublicKey, SecretKey},
    mollusk_svm::{program::keyed_account_for_system_program, result::ProgramResult, Mollusk},
    solana_account::Account,
    solana_instruction::{AccountMeta, Instruction},
    solana_pubkey::Pubkey,
};

mod common;
use common::mollusk_fixtures::{keyed_account_for_noreplay_program, mollusk_with_fixtures};

const PROGRAM_NAME: &str = "global_accountant";

fn program_id() -> Pubkey {
    // Fixed program id so test PDA derivation matches the program's view.
    Pubkey::new_from_array([7u8; 32])
}

fn mollusk() -> Mollusk {
    mollusk_with_fixtures(&program_id(), PROGRAM_NAME)
}

fn system_program_id() -> Pubkey {
    keyed_account_for_system_program().0
}

// ============================================================================
// PDA / instruction-data helpers
// ============================================================================

fn derive_pending_pda(
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
    digest: &[u8; 32],
) -> (Pubkey, u8) {
    let chain_be = chain.to_be_bytes();
    let sequence_be = sequence.to_be_bytes();
    Pubkey::find_program_address(
        &[
            PENDING_SEED_PREFIX,
            &chain_be,
            emitter,
            &sequence_be,
            digest,
        ],
        &program_id(),
    )
}

fn derive_account_pda(chain: u16, token_chain: u16, token_address: &[u8; 32]) -> (Pubkey, u8) {
    let chain_be = chain.to_be_bytes();
    let token_chain_be = token_chain.to_be_bytes();
    Pubkey::find_program_address(
        &[
            ACCOUNT_SEED_PREFIX,
            &chain_be,
            &token_chain_be,
            token_address,
        ],
        &program_id(),
    )
}

fn derive_chain_registration_pda(chain: u16) -> (Pubkey, u8) {
    let chain_be = chain.to_be_bytes();
    Pubkey::find_program_address(&[CHAIN_REGISTRATION_SEED_PREFIX, &chain_be], &program_id())
}

/// Host-side derivation of the canonical NoReplay bitmap PDA for
/// `(authority, chain, emitter, sequence)`. Mirrors the on-chain
/// `instructions::noreplay::derive_bucket_pda`.
fn derive_canonical_noreplay_bucket(
    authority: &Pubkey,
    chain: u16,
    emitter: &[u8; 32],
    sequence: u64,
) -> Pubkey {
    let mut namespace = [0u8; 34];
    namespace[..2].copy_from_slice(&chain.to_be_bytes());
    namespace[2..].copy_from_slice(emitter);
    let bucket_index = (sequence / NOREPLAY_BITS_PER_BUCKET).to_le_bytes();
    let (pda, _) = Pubkey::find_program_address(
        &[
            authority.as_ref(),
            &namespace[..32],
            &namespace[32..],
            &bucket_index,
        ],
        &Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
    );
    pda
}

/// Program-owned chain-registration PDA fixture for the given emitter,
/// bypassing the `register_chain` governance path.
fn chain_registration_account(chain: u16, emitter_address: &[u8; 32]) -> Account {
    let mut layout: ChainRegistrationLayout = bytemuck::Zeroable::zeroed();
    layout.tag = ChainRegistrationLayout::TAG;
    layout.chain = chain;
    layout.emitter_address = *emitter_address;
    Account {
        lamports: 1_000_000,
        data: bytemuck::bytes_of(&layout).to_vec(),
        owner: program_id(),
        executable: false,
        rent_epoch: 0,
    }
}

/// Host-side `keccak256(keccak256(body))` — the Wormhole VAA digest convention.
fn double_keccak256_host(body: &[u8]) -> [u8; 32] {
    let inner = solana_keccak_hasher::hashv(&[body]).to_bytes();
    solana_keccak_hasher::hashv(&[&inner]).to_bytes()
}

/// Attest-payload VAA body (action 0x02). The scenario digest is derived from
/// the body, not supplied independently.
fn build_attest_body(emitter_chain: u16, emitter_address: &[u8; 32], sequence: u64) -> Vec<u8> {
    // 51-byte header + action byte. The parser only reads the action, so 52
    // bytes suffice.
    let mut body = vec![0u8; 52];
    body[8..10].copy_from_slice(&emitter_chain.to_be_bytes());
    body[10..42].copy_from_slice(emitter_address);
    body[42..50].copy_from_slice(&sequence.to_be_bytes());
    body[51] = 0x02;
    body
}

/// Token Bridge transfer VAA body (action 0x01).
fn build_transfer_body(
    emitter_chain: u16,
    emitter_address: &[u8; 32],
    sequence: u64,
    amount: u128,
    token_chain: u16,
    token_address: &[u8; 32],
    recipient_chain: u16,
) -> Vec<u8> {
    let mut body = vec![0u8; 51 + 133];
    body[8..10].copy_from_slice(&emitter_chain.to_be_bytes());
    body[10..42].copy_from_slice(emitter_address);
    body[42..50].copy_from_slice(&sequence.to_be_bytes());
    body[51] = 0x01; // transfer payload starts at offset 51
    body[52 + 16..52 + 32].copy_from_slice(&amount.to_be_bytes()); // amount: 32-byte BE, low 16 hold the u128
    body[84..116].copy_from_slice(token_address);
    body[116..118].copy_from_slice(&token_chain.to_be_bytes());
    body[118] = 0xAB; // recipient: opaque to the accountant
    body[149] = 0xCD;
    body[150..152].copy_from_slice(&recipient_chain.to_be_bytes());
    body
}

fn submit_ix_data(
    digest: &[u8; 32],
    guardian_set_index: u32,
    guardian_index: u8,
    signature: &[u8; 65],
    body: &[u8],
) -> Vec<u8> {
    // Wire: discriminator + 102-byte fixed prefix + 2-byte body len (LE) + body.
    // No PDA bumps travel; routing tuple is read from the body header [8..50].
    let mut data = Vec::with_capacity(1 + 102 + 2 + body.len());
    data.push(IxDiscriminator::SubmitObservations as u8);
    data.extend_from_slice(digest);
    data.extend_from_slice(&guardian_set_index.to_le_bytes());
    data.push(guardian_index);
    data.extend_from_slice(signature);
    data.extend_from_slice(&(body.len() as u16).to_le_bytes());
    data.extend_from_slice(body);
    data
}

// ============================================================================
// Account fixtures
// ============================================================================

fn system_owned_account(lamports: u64) -> Account {
    Account {
        lamports,
        data: vec![],
        owner: system_program_id(),
        executable: false,
        rent_epoch: 0,
    }
}

fn uninitialised_pda_account() -> Account {
    system_owned_account(0)
}

/// Fresh NoReplay bucket: lazy-create entry state (system-owned, zero data).
fn noreplay_bucket_unmarked() -> Account {
    system_owned_account(0)
}

/// Pre-marked NoReplay bucket: 129-byte bitmap owned by `solana_noreplay` with
/// the bit at `sequence % 1024` set.
fn noreplay_bucket_marked(sequence: u64) -> Account {
    let mut data = vec![0u8; NOREPLAY_BITMAP_OFFSET + NOREPLAY_BITMAP_BYTES];
    let bit_index = (sequence % NOREPLAY_BITS_PER_BUCKET) as usize;
    data[NOREPLAY_BITMAP_OFFSET + bit_index / 8] |= 1u8 << (bit_index % 8);
    Account {
        lamports: 1_500_000_000,
        data,
        owner: Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
        executable: false,
        rent_epoch: 0,
    }
}

/// Core-Bridge-style `GuardianSet` account with the supplied 20-byte guardian
/// addresses, `creation_time`, and `expiration_time`.
fn guardian_set_account(
    index: u32,
    keys: &[[u8; 20]],
    creation_time: u32,
    expiration_time: u32,
) -> Account {
    let mut data = Vec::with_capacity(8 + keys.len() * 20 + 8);
    data.extend_from_slice(&index.to_le_bytes());
    data.extend_from_slice(&(keys.len() as u32).to_le_bytes());
    for key in keys {
        data.extend_from_slice(key);
    }
    data.extend_from_slice(&creation_time.to_le_bytes());
    data.extend_from_slice(&expiration_time.to_le_bytes());
    Account {
        lamports: 1_000_000,
        data,
        owner: Pubkey::new_from_array(CORE_BRIDGE_PROGRAM_ID),
        executable: false,
        rent_epoch: 0,
    }
}

// ============================================================================
// Guardian-set generation: deterministic per-test seeded keys.
// ============================================================================

#[derive(Clone)]
struct Guardian {
    secret: SecretKey,
    eth_address: [u8; 20],
}

/// Generate `count` deterministic secp256k1 keypairs and their 20-byte
/// Ethereum-style addresses. Small-magnitude seed bytes keep the scalar inside
/// the group order without needing a crypto-grade RNG.
fn make_guardians(count: usize, seed: u8) -> Vec<Guardian> {
    let mut out = Vec::with_capacity(count);
    for i in 0..count {
        let mut sk_bytes = [0u8; 32];
        sk_bytes[0] = seed;
        sk_bytes[1] = i as u8;
        sk_bytes[31] = (i as u8).wrapping_add(1);

        let secret =
            SecretKey::parse(&sk_bytes).expect("deterministic seed inside secp256k1 group order");
        let public = PublicKey::from_secret_key(&secret);
        // `serialize()` emits 0x04 prefix + 64 raw (X||Y); strip prefix.
        let pk_uncompressed = public.serialize();
        let raw = &pk_uncompressed[1..];
        let hash = keccak256_host(raw);
        let mut eth_address = [0u8; 20];
        eth_address.copy_from_slice(&hash[12..]);
        out.push(Guardian {
            secret,
            eth_address,
        });
    }
    out
}

/// Host-side keccak256 for deriving guardian-set fixture addresses.
fn keccak256_host(data: &[u8]) -> [u8; 32] {
    solana_keccak_hasher::hashv(&[data]).to_bytes()
}

fn sign_digest(guardian: &Guardian, digest: &[u8; 32]) -> [u8; 65] {
    let msg = Message::parse(digest);
    let (sig, rec) = sign(&msg, &guardian.secret);
    let sig_bytes = sig.serialize(); // 64-byte r||s + recovery byte
    let mut out = [0u8; 65];
    out[..64].copy_from_slice(&sig_bytes);
    out[64] = rec.serialize();
    out
}

// ============================================================================
// Scenario builder — assembles the account list and submits observations.
// ============================================================================

#[derive(Clone)]
struct Scenario {
    chain: u16,
    emitter: [u8; 32],
    sequence: u64,
    /// VAA body — the source of truth; `digest` is derived from it. Default is
    /// an Attest payload (no balance work); transfer tests use
    /// `Self::with_transfer_body`.
    body: Vec<u8>,
    digest: [u8; 32],
    guardian_set_index: u32,
    guardians: Vec<Guardian>,
    submitter: Pubkey,
    pending_pda: Pubkey,
    guardian_set_pubkey: Pubkey,
    noreplay_bucket_pubkey: Pubkey,
    noreplay_program_pubkey: Pubkey,
    /// Canonical `noreplay-authority` PDA.
    noreplay_authority_pubkey: Pubkey,
    /// Source-chain Account PDA (slot 7). Attest scenarios use the
    /// noreplay-authority pubkey as a sentinel since the slot is untouched.
    source_account_pubkey: Pubkey,
    /// Destination-chain Account PDA (slot 8). Same semantics as `source`.
    dest_account_pubkey: Pubkey,
    /// Chain-registration PDA (slot 10), default pre-populated `chain -> emitter`.
    /// Negative tests override it to drive `MissingChainRegistration` /
    /// `UnregisteredEmitter`.
    chain_registration_pubkey: Pubkey,
}

impl Scenario {
    fn new(guardian_count: usize, gsi: u32, seed: u8) -> Self {
        let chain: u16 = 2;
        let mut emitter = [0u8; 32];
        emitter[31] = 0x77;
        let sequence: u64 = 0x0000_0000_0000_0042;

        let body = build_attest_body(chain, &emitter, sequence);
        let digest = double_keccak256_host(&body);

        let guardians = make_guardians(guardian_count, seed);
        let submitter = Pubkey::new_from_array([0x11u8; 32]);
        let (pending_pda, _) = derive_pending_pda(chain, &emitter, sequence, &digest);
        let (noreplay_authority_pubkey, _) =
            Pubkey::find_program_address(&[NOREPLAY_AUTHORITY_SEED_PREFIX], &program_id());
        let (chain_registration_pubkey, _) = derive_chain_registration_pda(chain);
        let noreplay_bucket_pubkey =
            derive_canonical_noreplay_bucket(&noreplay_authority_pubkey, chain, &emitter, sequence);

        Self {
            chain,
            emitter,
            sequence,
            body,
            digest,
            guardian_set_index: gsi,
            guardians,
            submitter,
            pending_pda,
            guardian_set_pubkey: Pubkey::new_from_array([0xC1u8; 32]),
            noreplay_bucket_pubkey,
            noreplay_program_pubkey: Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
            noreplay_authority_pubkey,
            // Attest payload ⇒ slots 8/9 untouched; reuse noreplay-authority as
            // a sentinel.
            source_account_pubkey: noreplay_authority_pubkey,
            dest_account_pubkey: noreplay_authority_pubkey,
            chain_registration_pubkey,
        }
    }

    /// Swap the Attest body for a Transfer body and re-derive the digest,
    /// pending PDA, and source/dest Account PDAs.
    fn with_transfer_body(
        guardian_count: usize,
        gsi: u32,
        seed: u8,
        amount: u128,
        token_chain: u16,
        token_address: [u8; 32],
        recipient_chain: u16,
    ) -> Self {
        let mut base = Self::new(guardian_count, gsi, seed);
        base.body = build_transfer_body(
            base.chain,
            &base.emitter,
            base.sequence,
            amount,
            token_chain,
            &token_address,
            recipient_chain,
        );
        base.digest = double_keccak256_host(&base.body);
        let (pending_pda, _) =
            derive_pending_pda(base.chain, &base.emitter, base.sequence, &base.digest);
        base.pending_pda = pending_pda;
        // Source keys on the emitter chain; dest keys on recipient_chain.
        let (src, _) = derive_account_pda(base.chain, token_chain, &token_address);
        let (dst, _) = derive_account_pda(recipient_chain, token_chain, &token_address);
        base.source_account_pubkey = src;
        base.dest_account_pubkey = dst;
        base
    }

    fn guardian_keys(&self) -> Vec<[u8; 20]> {
        self.guardians.iter().map(|g| g.eth_address).collect()
    }

    /// Submit one observation from `guardian_index`. Pass the previous result's
    /// accounts to persist PDA state across observations.
    fn submit_once(
        &self,
        mollusk: &Mollusk,
        starting_accounts: Vec<(Pubkey, Account)>,
        guardian_index: u8,
    ) -> mollusk_svm::result::InstructionResult {
        let guardian = &self.guardians[guardian_index as usize];
        let signature = sign_digest(guardian, &self.digest);
        let ix = Instruction::new_with_bytes(
            program_id(),
            &submit_ix_data(
                &self.digest,
                self.guardian_set_index,
                guardian_index,
                &signature,
                &self.body,
            ),
            self.account_metas(),
        );
        mollusk.process_instruction(&ix, &starting_accounts)
    }

    /// 11-entry account-meta list. Slot 9 is rent_recipient (= submitter),
    /// slot 10 the chain-registration PDA. Multi-submitter and
    /// registration-negative tests build their own meta vec inline.
    fn account_metas(&self) -> Vec<AccountMeta> {
        vec![
            AccountMeta::new(self.submitter, true),
            AccountMeta::new(self.pending_pda, false),
            AccountMeta::new_readonly(self.guardian_set_pubkey, false),
            AccountMeta::new(self.noreplay_bucket_pubkey, false),
            AccountMeta::new_readonly(system_program_id(), false),
            AccountMeta::new_readonly(self.noreplay_program_pubkey, false),
            AccountMeta::new_readonly(self.noreplay_authority_pubkey, false),
            AccountMeta::new(self.source_account_pubkey, false),
            AccountMeta::new(self.dest_account_pubkey, false),
            AccountMeta::new(self.submitter, false),
            AccountMeta::new_readonly(self.chain_registration_pubkey, false),
        ]
    }

    /// Initial account list with all PDAs uninitialised. Slots 7/8 are
    /// system-owned + empty so the lazy-init path fires on Transfer quorum.
    fn initial_accounts(&self) -> Vec<(Pubkey, Account)> {
        let mut accounts = vec![
            (self.submitter, system_owned_account(50_000_000_000)),
            (self.pending_pda, uninitialised_pda_account()),
            (
                self.guardian_set_pubkey,
                guardian_set_account(self.guardian_set_index, &self.guardian_keys(), 0, 0),
            ),
            (self.noreplay_bucket_pubkey, noreplay_bucket_unmarked()),
            keyed_account_for_system_program(),
            keyed_account_for_noreplay_program(),
            (self.noreplay_authority_pubkey, system_owned_account(0)),
        ];
        // Slots 7/8: append only when the sentinel hasn't collapsed them onto
        // the noreplay-authority pubkey (Attest scenario).
        if self.source_account_pubkey != self.noreplay_authority_pubkey {
            accounts.push((self.source_account_pubkey, uninitialised_pda_account()));
        }
        if self.dest_account_pubkey != self.noreplay_authority_pubkey
            && self.dest_account_pubkey != self.source_account_pubkey
        {
            accounts.push((self.dest_account_pubkey, uninitialised_pda_account()));
        }
        // Slot 10: chain-registration PDA pre-populated with the scenario emitter.
        accounts.push((
            self.chain_registration_pubkey,
            chain_registration_account(self.chain, &self.emitter),
        ));
        accounts
    }

    /// Run `n` observations from guardian indices `0..n`, returning the final
    /// accounts.
    fn submit_n(&self, mollusk: &Mollusk, n: u8) -> Vec<(Pubkey, Account)> {
        let mut accounts = self.initial_accounts();
        for i in 0..n {
            let result = self.submit_once(mollusk, accounts.clone(), i);
            assert!(
                matches!(result.program_result, ProgramResult::Success),
                "submit #{i} expected success, got {:?}",
                result.program_result
            );
            accounts = result.resulting_accounts.clone();
        }
        accounts
    }
}

fn find_account<'a>(accounts: &'a [(Pubkey, Account)], key: &Pubkey) -> &'a Account {
    &accounts
        .iter()
        .find(|(k, _)| k == key)
        .unwrap_or_else(|| panic!("account {key} not in result list"))
        .1
}

// ============================================================================
// Tests
// ============================================================================

/// First observation allocates and populates the pending PDA.
#[test]
fn submit_first_observation_creates_pending_pda() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x42);

    let accounts = scenario.submit_n(&mollusk, 1);

    let pending = find_account(&accounts, &scenario.pending_pda);
    assert_eq!(pending.owner, program_id(), "pending PDA owned by program");
    assert_eq!(
        pending.data.len(),
        PendingObservationsLayout::LEN,
        "pending PDA allocated to full layout length"
    );
    let layout: &PendingObservationsLayout = bytemuck::from_bytes(&pending.data);
    assert_eq!(layout.digest, scenario.digest, "digest persisted");
    assert_eq!(
        layout.guardian_set_index, scenario.guardian_set_index,
        "guardian_set_index persisted"
    );
    assert_eq!(layout.signatures, 0b1, "bit 0 set after first observation");
    assert_eq!(layout.chain, scenario.chain, "chain persisted");
    assert_eq!(
        layout.payer,
        scenario.submitter.to_bytes(),
        "submitter is the recorded payer"
    );

    // No quorum: NoReplay bucket stays in lazy-create entry state.
    let bucket = find_account(&accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        system_program_id(),
        "bucket still system-owned"
    );
    assert!(bucket.data.is_empty(), "bucket still uninitialised");
}

/// 12 observations (sub-quorum) accumulate in the bitmap without committing.
#[test]
fn submit_12_observations_accumulates_without_commit() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x43);

    let accounts = scenario.submit_n(&mollusk, 12);

    let pending = find_account(&accounts, &scenario.pending_pda);
    let layout: &PendingObservationsLayout = bytemuck::from_bytes(&pending.data);
    assert_eq!(
        layout.signatures.count_ones(),
        12,
        "bitmap has exactly 12 bits set after 12 observations"
    );
    assert_eq!(
        layout.signatures, 0b1111_1111_1111u32,
        "bits 0..12 set in low-to-high order"
    );

    // Sub-quorum: NoReplay unmarked.
    let bucket = find_account(&accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        system_program_id(),
        "bucket still system-owned"
    );
    assert!(
        bucket.data.is_empty(),
        "bucket still uninitialised at 12/19"
    );
}

/// 13th observation reaches quorum: pending closes, commit-log emitted, NoReplay flips.
#[test]
fn submit_13th_observation_reaches_quorum_and_commits() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x44);

    let accounts = scenario.submit_n(&mollusk, 13);

    // Pending PDA closed: lamports drained, owner reverted to system.
    let pending = find_account(&accounts, &scenario.pending_pda);
    assert_eq!(
        pending.lamports, 0,
        "pending PDA lamports drained on commit"
    );
    assert_eq!(
        pending.owner,
        system_program_id(),
        "pending PDA reassigned to system program on close"
    );
    assert!(
        pending.data.is_empty(),
        "pending PDA data dropped on close, got {} bytes",
        pending.data.len()
    );

    // The canonical commit log is emitted on the quorum-completing branch via
    // `sol_log_data` (see `instructions/commit_log.rs`). Mollusk's
    // `InstructionResult` does not expose program logs, so log content is
    // verified in the surfpool e2e suite (`tx.meta.logMessages`).

    // NoReplay flipped: bitmap allocated, owned by noreplay, bit at
    // `sequence % 1024` set.
    let bucket = find_account(&accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
        "bucket owned by solana_noreplay after MarkUsed"
    );
    assert_eq!(
        bucket.data.len(),
        NOREPLAY_BITMAP_OFFSET + NOREPLAY_BITMAP_BYTES,
        "bucket data sized to bitmap layout"
    );
    let bit = (scenario.sequence % NOREPLAY_BITS_PER_BUCKET) as usize;
    let byte = NOREPLAY_BITMAP_OFFSET + bit / 8;
    let mask = 1u8 << (bit % 8);
    assert_eq!(
        bucket.data[byte] & mask,
        mask,
        "bitmap bit set on quorum reach"
    );
}

/// Quorum completed by a different submitter: rent refunds to the recorded
/// payer (slot 10), not the quorum-completing signer. A wrong rent_recipient
/// must fail with `PayerMismatch`.
#[test]
fn submit_observations_quorum_with_different_submitter_refunds_recorded_payer() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x70);

    // Alice (= scenario.submitter) accumulates the first 12 signatures.
    let accounts_after_12 = scenario.submit_n(&mollusk, 12);
    let alice = scenario.submitter;
    let alice_lamports_pre = find_account(&accounts_after_12, &alice).lamports;
    let pending_lamports = find_account(&accounts_after_12, &scenario.pending_pda).lamports;
    assert!(
        pending_lamports > 0,
        "pending PDA must be rent-funded at 12/19"
    );

    // Bob arrives with the 13th signature from a freshly-funded wallet.
    let bob = Pubkey::new_from_array([0xB0u8; 32]);
    let bob_starting_lamports = 50_000_000_000u64;
    let mut accounts = accounts_after_12.clone();
    accounts.push((bob, system_owned_account(bob_starting_lamports)));

    let signature = sign_digest(&scenario.guardians[12], &scenario.digest);
    let ix_data = submit_ix_data(
        &scenario.digest,
        scenario.guardian_set_index,
        12,
        &signature,
        &scenario.body,
    );

    // (1) Wrong rent_recipient (bob): must fail with PayerMismatch.
    let wrong_metas = vec![
        AccountMeta::new(bob, true),
        AccountMeta::new(scenario.pending_pda, false),
        AccountMeta::new_readonly(scenario.guardian_set_pubkey, false),
        AccountMeta::new(scenario.noreplay_bucket_pubkey, false),
        AccountMeta::new_readonly(system_program_id(), false),
        AccountMeta::new_readonly(scenario.noreplay_program_pubkey, false),
        AccountMeta::new_readonly(scenario.noreplay_authority_pubkey, false),
        AccountMeta::new(scenario.source_account_pubkey, false),
        AccountMeta::new(scenario.dest_account_pubkey, false),
        AccountMeta::new(bob, false), // rent_recipient = bob (wrong)
        AccountMeta::new_readonly(scenario.chain_registration_pubkey, false),
    ];
    let ix_wrong = Instruction::new_with_bytes(program_id(), &ix_data, wrong_metas);
    let r_wrong = mollusk.process_instruction(&ix_wrong, &accounts);
    match r_wrong.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::PayerMismatch as u32,
                "wrong rent_recipient must fail with PayerMismatch, got {code:?}"
            );
        }
        other => panic!("expected Failure(PayerMismatch), got {other:?}"),
    }

    // (2) Correct rent_recipient (alice): succeeds and refunds alice.
    let correct_metas = vec![
        AccountMeta::new(bob, true),
        AccountMeta::new(scenario.pending_pda, false),
        AccountMeta::new_readonly(scenario.guardian_set_pubkey, false),
        AccountMeta::new(scenario.noreplay_bucket_pubkey, false),
        AccountMeta::new_readonly(system_program_id(), false),
        AccountMeta::new_readonly(scenario.noreplay_program_pubkey, false),
        AccountMeta::new_readonly(scenario.noreplay_authority_pubkey, false),
        AccountMeta::new(scenario.source_account_pubkey, false),
        AccountMeta::new(scenario.dest_account_pubkey, false),
        AccountMeta::new(alice, false), // rent_recipient = alice (correct)
        AccountMeta::new_readonly(scenario.chain_registration_pubkey, false),
    ];
    let ix_correct = Instruction::new_with_bytes(program_id(), &ix_data, correct_metas);
    let r_correct = mollusk.process_instruction(&ix_correct, &accounts);
    assert!(
        matches!(r_correct.program_result, ProgramResult::Success),
        "13th submission from a different submitter with correct rent_recipient \
         must succeed, got {:?}",
        r_correct.program_result
    );

    // Refund routes to alice (recorded payer). The commit log emit's
    // payer-of-record is now implicit in the submitting tx's fee payer rather
    // than a field on the (removed) DigestAccount PDA; the log content itself
    // is verified by the surfpool e2e suite, which has access to
    // `tx.meta.logMessages` (mollusk's `InstructionResult` does not expose them).
    let alice_post = find_account(&r_correct.resulting_accounts, &alice);
    assert_eq!(
        alice_post.lamports,
        alice_lamports_pre + pending_lamports,
        "alice (recorded payer) received the pending-PDA rent refund"
    );
}

/// Security: the routing tuple (chain, emitter, sequence) is read from the
/// signed body header [8..50], never from caller-supplied prefix bytes. An
/// attacker-supplied pending PDA at a different namespace is rejected.
#[test]
fn submit_observations_routes_by_body_header_not_caller_supplied_prefix() {
    let mollusk = mollusk();

    // Authoritative routing tuple lives in body[8..50].
    let body_chain = 2u16;
    let mut body_emitter = [0u8; 32];
    body_emitter[31] = 0x77;
    let body_sequence = 0x42u64;
    let body = build_attest_body(body_chain, &body_emitter, body_sequence);
    let digest = double_keccak256_host(&body);

    let guardians = make_guardians(19, 0x80);
    let signature = sign_digest(&guardians[0], &digest);

    // Attack: supply a pending PDA canonical for an attacker namespace, not
    // the body's. The program derives the canonical address from body[8..50],
    // so it cannot sign for the attacker's address.
    let attacker_chain = 99u16;
    let attacker_emitter = [0xFFu8; 32];
    let attacker_sequence = 0x9999u64;
    let (attacker_pending_pda, _) = derive_pending_pda(
        attacker_chain,
        &attacker_emitter,
        attacker_sequence,
        &digest,
    );
    let (body_pending_pda, _) =
        derive_pending_pda(body_chain, &body_emitter, body_sequence, &digest);
    assert_ne!(
        body_pending_pda, attacker_pending_pda,
        "test fixture must drive distinct pending PDA addresses"
    );

    let submitter = Pubkey::new_from_array([0x11u8; 32]);
    let guardian_set_pubkey = Pubkey::new_from_array([0xC1u8; 32]);
    let (noreplay_authority_pubkey, _) =
        Pubkey::find_program_address(&[NOREPLAY_AUTHORITY_SEED_PREFIX], &program_id());
    let noreplay_bucket_pubkey = derive_canonical_noreplay_bucket(
        &noreplay_authority_pubkey,
        body_chain,
        &body_emitter,
        body_sequence,
    );
    let noreplay_program_pubkey = Pubkey::new_from_array(NOREPLAY_PROGRAM_ID);

    let ix_data = submit_ix_data(&digest, 4, 0, &signature, &body);

    // Pre-populate the body-chain registration so the registration check
    // passes; this test targets the pending-PDA rejection.
    let (registration_pda, _) = derive_chain_registration_pda(body_chain);

    let guardian_keys: Vec<[u8; 20]> = guardians.iter().map(|g| g.eth_address).collect();
    let accounts = vec![
        (submitter, system_owned_account(50_000_000_000)),
        (attacker_pending_pda, uninitialised_pda_account()),
        (
            guardian_set_pubkey,
            guardian_set_account(4, &guardian_keys, 0, 0),
        ),
        (noreplay_bucket_pubkey, noreplay_bucket_unmarked()),
        keyed_account_for_system_program(),
        keyed_account_for_noreplay_program(),
        (noreplay_authority_pubkey, system_owned_account(0)),
        (
            registration_pda,
            chain_registration_account(body_chain, &body_emitter),
        ),
    ];

    let metas = vec![
        AccountMeta::new(submitter, true),
        AccountMeta::new(attacker_pending_pda, false),
        AccountMeta::new_readonly(guardian_set_pubkey, false),
        AccountMeta::new(noreplay_bucket_pubkey, false),
        AccountMeta::new_readonly(system_program_id(), false),
        AccountMeta::new_readonly(noreplay_program_pubkey, false),
        AccountMeta::new_readonly(noreplay_authority_pubkey, false),
        AccountMeta::new(noreplay_authority_pubkey, false),
        AccountMeta::new(noreplay_authority_pubkey, false),
        AccountMeta::new(submitter, false),
        AccountMeta::new_readonly(registration_pda, false),
    ];
    let ix = Instruction::new_with_bytes(program_id(), &ix_data, metas);
    let r = mollusk.process_instruction(&ix, &accounts);

    // The init CPI can only sign for the body-derived canonical address, so a
    // pending PDA at any other address aborts with `PrivilegeEscalation`.
    assert!(
        !matches!(r.program_result, ProgramResult::Success),
        "spoofed pending PDA must be rejected, got {:?}",
        r.program_result
    );
    let result_debug = format!("{:?}", r.program_result);
    assert!(
        result_debug.contains("PrivilegeEscalation"),
        "spoofed pending PDA must reject with PrivilegeEscalation — \
         create_pending_pda signs only for the body-derived canonical address, \
         so the init CPI cannot sign for the attacker's address; got {result_debug}"
    );

    // Attacker's pending PDA stays uninitialised (tx unwinds atomically).
    let attacker_after = find_account(&r.resulting_accounts, &attacker_pending_pda);
    assert_eq!(
        attacker_after.owner,
        system_program_id(),
        "attacker-supplied pending PDA must NOT be initialised after rejection"
    );
}

/// Observation whose emitter_chain has no registration PDA is refused with
/// `MissingChainRegistration`.
#[test]
fn submit_observations_rejects_unregistered_chain() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0xA0);

    let (registration_pda, _) = derive_chain_registration_pda(scenario.chain);
    let signature = sign_digest(&scenario.guardians[0], &scenario.digest);

    // Replace the default registration with an uninitialised PDA.
    let mut accounts = scenario.initial_accounts();
    for entry in accounts.iter_mut() {
        if entry.0 == registration_pda {
            entry.1 = uninitialised_pda_account();
            break;
        }
    }

    let metas = vec![
        AccountMeta::new(scenario.submitter, true),
        AccountMeta::new(scenario.pending_pda, false),
        AccountMeta::new_readonly(scenario.guardian_set_pubkey, false),
        AccountMeta::new(scenario.noreplay_bucket_pubkey, false),
        AccountMeta::new_readonly(system_program_id(), false),
        AccountMeta::new_readonly(scenario.noreplay_program_pubkey, false),
        AccountMeta::new_readonly(scenario.noreplay_authority_pubkey, false),
        AccountMeta::new(scenario.source_account_pubkey, false),
        AccountMeta::new(scenario.dest_account_pubkey, false),
        AccountMeta::new(scenario.submitter, false),
        AccountMeta::new_readonly(registration_pda, false),
    ];
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &scenario.body,
        ),
        metas,
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::MissingChainRegistration as u32,
                "expected MissingChainRegistration, got {code:?}"
            );
        }
        other => panic!("expected Failure(MissingChainRegistration), got {other:?}"),
    }
}

/// Registration exists but holds a different emitter than the body header:
/// rejects with `UnregisteredEmitter`.
#[test]
fn submit_observations_rejects_wrong_emitter_for_registered_chain() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0xA1);

    let registration_pubkey = scenario.chain_registration_pubkey;
    let wrong_emitter = [0xCCu8; 32];
    let mut accounts = scenario.initial_accounts();
    for entry in accounts.iter_mut() {
        if entry.0 == registration_pubkey {
            entry.1 = chain_registration_account(scenario.chain, &wrong_emitter);
            break;
        }
    }

    let signature = sign_digest(&scenario.guardians[0], &scenario.digest);
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &scenario.body,
        ),
        scenario.account_metas(),
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::UnregisteredEmitter as u32,
                "expected UnregisteredEmitter, got {code:?}"
            );
        }
        other => panic!("expected Failure(UnregisteredEmitter), got {other:?}"),
    }
}

/// Wrong-seed registration PDA is rejected by the canonical-address check
/// (`InvalidPda`) before any data read.
#[test]
fn submit_observations_rejects_spoofed_registration_pda() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0xA2);

    // Spoofed PDA derived from a different chain ID.
    let (spoofed_pda, _) = derive_chain_registration_pda(99);
    assert_ne!(spoofed_pda, scenario.chain_registration_pubkey);

    let mut accounts = scenario.initial_accounts();
    accounts.push((spoofed_pda, chain_registration_account(99, &[0xAA; 32])));

    let mut metas = scenario.account_metas();
    let last_idx = metas.len() - 1;
    metas[last_idx] = AccountMeta::new_readonly(spoofed_pda, false);

    let signature = sign_digest(&scenario.guardians[0], &scenario.digest);
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &scenario.body,
        ),
        metas,
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::InvalidPda as u32,
                "expected InvalidPda from canonical-address check, got {code:?}"
            );
        }
        other => panic!("expected Failure(InvalidPda), got {other:?}"),
    }
}

/// A corrupted signature is rejected with `InvalidSignature`.
#[test]
fn submit_with_invalid_signature_fails() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x45);

    let mut signature = sign_digest(&scenario.guardians[0], &scenario.digest);
    signature[0] ^= 0xff;

    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &scenario.body,
        ),
        scenario.account_metas(),
    );
    let result = mollusk.process_instruction(&ix, &scenario.initial_accounts());
    match result.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::InvalidSignature as u32,
                "expected InvalidSignature, got {code:?}"
            );
        }
        other => panic!("expected Failure(InvalidSignature), got {other:?}"),
    }
}

/// Recovery id outside {0,1,2,3} is rejected before `secp256k1_recover`.
#[test]
fn submit_with_recovery_id_4_rejects() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x52);
    let mut signature = sign_digest(&scenario.guardians[0], &scenario.digest);
    signature[64] = 4;

    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &scenario.body,
        ),
        scenario.account_metas(),
    );
    let result = mollusk.process_instruction(&ix, &scenario.initial_accounts());
    match result.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::InvalidSignature as u32,
                "expected InvalidSignature, got {code:?}"
            );
        }
        other => panic!("expected Failure(InvalidSignature), got {other:?}"),
    }
}

/// Two observations from the same guardian index: second fails `AlreadySigned`.
#[test]
fn submit_with_duplicate_guardian_index_fails() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x46);

    let mut accounts = scenario.initial_accounts();
    let r1 = scenario.submit_once(&mollusk, accounts.clone(), 0);
    assert!(matches!(r1.program_result, ProgramResult::Success));
    accounts = r1.resulting_accounts.clone();

    let r2 = scenario.submit_once(&mollusk, accounts, 0);
    match r2.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::AlreadySigned as u32,
                "expected AlreadySigned, got {code:?}"
            );
        }
        other => panic!("expected Failure(AlreadySigned), got {other:?}"),
    }
}

/// Drives each `read_guardian_key` rejection branch:
///   (a) truncated header              -> InvalidPda
///   (b) on-chain index != wire index  -> InvalidGuardianIndex
///   (c) guardian_index >= keys_len    -> InvalidGuardianIndex
///   (d) keys array truncated          -> InvalidPda
#[test]
fn submit_with_malformed_guardian_set_rejects() {
    let mollusk = mollusk();
    let core_bridge = Pubkey::new_from_array(CORE_BRIDGE_PROGRAM_ID);

    let truncated_header = Account {
        lamports: 1_000_000,
        data: vec![0u8; 4], // < 8 bytes
        owner: core_bridge,
        executable: false,
        rent_epoch: 0,
    };

    let mismatched_index = {
        // Same keys, header encodes a different index.
        let scenario = Scenario::new(19, 4, 0x53);
        guardian_set_account(99, &scenario.guardian_keys(), 0, 0)
    };

    let short_keys_array = {
        // keys_len = 3 but wire guardian_index = 5.
        let scenario = Scenario::new(19, 4, 0x53);
        let truncated: Vec<[u8; 20]> = scenario.guardians[..3]
            .iter()
            .map(|g| g.eth_address)
            .collect();
        guardian_set_account(4, &truncated, 0, 0)
    };

    let truncated_keys_buffer = {
        // keys_len = 19 but only 5 keys present; an 18th-key read overruns.
        let mut data = Vec::with_capacity(8 + 5 * 20);
        data.extend_from_slice(&4u32.to_le_bytes());
        data.extend_from_slice(&19u32.to_le_bytes());
        data.extend_from_slice(&[0u8; 5 * 20]);
        Account {
            lamports: 1_000_000,
            data,
            owner: core_bridge,
            executable: false,
            rent_epoch: 0,
        }
    };

    let cases: [(&str, Account, u8, u32); 4] = [
        (
            "truncated header",
            truncated_header,
            0,
            GlobalAccountantError::InvalidPda as u32,
        ),
        (
            "on-chain index mismatch",
            mismatched_index,
            0,
            GlobalAccountantError::InvalidGuardianIndex as u32,
        ),
        (
            "guardian_index >= keys_len",
            short_keys_array,
            5,
            GlobalAccountantError::InvalidGuardianIndex as u32,
        ),
        (
            "keys buffer truncated",
            truncated_keys_buffer,
            18,
            GlobalAccountantError::InvalidPda as u32,
        ),
    ];

    for (label, gs_account, guardian_index, expected_code) in cases {
        let scenario = Scenario::new(19, 4, 0x53);
        let mut accounts = scenario.initial_accounts();
        if let Some(entry) = accounts
            .iter_mut()
            .find(|(k, _)| *k == scenario.guardian_set_pubkey)
        {
            entry.1 = gs_account;
        }

        // Valid signature — handler must short-circuit before secp256k1_recover.
        let signature = sign_digest(
            &scenario.guardians[guardian_index as usize],
            &scenario.digest,
        );
        let ix = Instruction::new_with_bytes(
            program_id(),
            &submit_ix_data(
                &scenario.digest,
                scenario.guardian_set_index,
                guardian_index,
                &signature,
                &scenario.body,
            ),
            scenario.account_metas(),
        );
        let result = mollusk.process_instruction(&ix, &accounts);
        match result.program_result {
            ProgramResult::Failure(err) => {
                let code = u64::from(err) as u32;
                assert_eq!(
                    code, expected_code,
                    "[{label}] expected code {expected_code}, got {code}"
                );
            }
            other => panic!("[{label}] expected Failure, got {other:?}"),
        }
    }
}

/// Pending PDA at GSI=5; an observation under stale GSI=4 fails `StaleGuardianSet`.
#[test]
fn submit_with_stale_old_set_observation_fails() {
    let mollusk = mollusk();
    let new_scenario = Scenario::new(19, 5, 0x47);
    let accounts_after_first = new_scenario.submit_n(&mollusk, 1);

    let old_guardians = make_guardians(19, 0x48); // distinct keys for GSI=4
    let stale_signature = sign_digest(&old_guardians[1], &new_scenario.digest);

    // GSI=4 set at the same pubkey so the program reads index 4.
    let old_gs_account = guardian_set_account(
        4,
        &old_guardians
            .iter()
            .map(|g| g.eth_address)
            .collect::<Vec<_>>(),
        0,
        0,
    );
    let mut accounts = accounts_after_first.clone();
    if let Some(entry) = accounts
        .iter_mut()
        .find(|(k, _)| *k == new_scenario.guardian_set_pubkey)
    {
        entry.1 = old_gs_account;
    }

    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &new_scenario.digest,
            4, // stale index
            1,
            &stale_signature,
            &new_scenario.body,
        ),
        new_scenario.account_metas(),
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::StaleGuardianSet as u32,
                "expected StaleGuardianSet, got {code:?}"
            );
        }
        other => panic!("expected Failure(StaleGuardianSet), got {other:?}"),
    }
}

/// Observation under a newer GSI wipes the old pending PDA and recreates it
/// with a single bit set under the new index.
#[test]
fn submit_with_new_set_observation_wipes_old_pending() {
    let mollusk = mollusk();
    let old_scenario = Scenario::new(19, 4, 0x4A);
    let accounts_after_first = old_scenario.submit_n(&mollusk, 1);

    // Same emitter/sequence/chain so the pending PDA address collides.
    let new_guardians = make_guardians(19, 0x4B);
    let new_gs_account = guardian_set_account(
        5,
        &new_guardians
            .iter()
            .map(|g| g.eth_address)
            .collect::<Vec<_>>(),
        0,
        0,
    );
    let mut accounts = accounts_after_first.clone();
    if let Some(entry) = accounts
        .iter_mut()
        .find(|(k, _)| *k == old_scenario.guardian_set_pubkey)
    {
        entry.1 = new_gs_account;
    }

    let signature = sign_digest(&new_guardians[0], &old_scenario.digest);
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(&old_scenario.digest, 5, 0, &signature, &old_scenario.body),
        old_scenario.account_metas(),
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    assert!(
        matches!(r.program_result, ProgramResult::Success),
        "rotation submit must succeed, got {:?}",
        r.program_result
    );

    let pending = find_account(&r.resulting_accounts, &old_scenario.pending_pda);
    assert_eq!(
        pending.owner,
        program_id(),
        "pending PDA still owned by program after rotation"
    );
    let layout: &PendingObservationsLayout = bytemuck::from_bytes(&pending.data);
    assert_eq!(
        layout.guardian_set_index, 5,
        "guardian_set_index advanced to the new set"
    );
    assert_eq!(
        layout.signatures, 0b1,
        "bitmap reset to a single bit under the new set"
    );
}

/// A second digest under the same guardian set routes into a sibling pending
/// PDA (digest is in the seeds) rather than being rejected.
#[test]
fn submit_with_different_digest_under_same_set_creates_sibling_bucket() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x4C);

    // First observation under digest D1.
    let accounts_after_first = scenario.submit_n(&mollusk, 1);

    // D2 must land in a sibling PDA. Mutate consistency_level (offset 50) so
    // the routing tuple at body[8..50] is unchanged.
    let mut alternate_body = scenario.body.clone();
    alternate_body[50] = 0xAA;
    let alternate_digest = double_keccak256_host(&alternate_body);
    assert_ne!(
        alternate_digest, scenario.digest,
        "one-byte body change must yield a different digest"
    );
    let signature = sign_digest(&scenario.guardians[1], &alternate_digest);

    let (d2_pending_pda, _) = derive_pending_pda(
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
        &alternate_digest,
    );
    assert_ne!(
        d2_pending_pda, scenario.pending_pda,
        "per-digest PDA seeds must yield distinct addresses for distinct digests"
    );

    let mut accounts = accounts_after_first.clone();
    accounts.push((d2_pending_pda, uninitialised_pda_account()));

    let mut metas = scenario.account_metas();
    metas[1] = AccountMeta::new(d2_pending_pda, false); // slot 1 = D2 sibling
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &alternate_digest,
            scenario.guardian_set_index,
            1,
            &signature,
            &alternate_body,
        ),
        metas,
    );
    let r = mollusk.process_instruction(&ix, &accounts);
    assert!(
        matches!(r.program_result, ProgramResult::Success),
        "different-digest submit under same GSI must succeed into a sibling \
         bucket, got {:?}",
        r.program_result
    );

    // D1 bucket untouched at 1 sig.
    let d1 = find_account(&r.resulting_accounts, &scenario.pending_pda);
    let d1_layout: &PendingObservationsLayout = bytemuck::from_bytes(&d1.data);
    assert_eq!(d1_layout.digest, scenario.digest);
    assert_eq!(
        d1_layout.signatures, 0b1,
        "D1 bucket still at one signature"
    );

    // D2 bucket has one sig at guardian-index 1.
    let d2 = find_account(&r.resulting_accounts, &d2_pending_pda);
    assert_eq!(d2.owner, program_id(), "D2 sibling PDA owned by program");
    let d2_layout: &PendingObservationsLayout = bytemuck::from_bytes(&d2.data);
    assert_eq!(d2_layout.digest, alternate_digest, "D2 digest persisted");
    assert_eq!(
        d2_layout.signatures, 0b10,
        "D2 bucket has bit 1 set (only guardian-index 1 has signed)"
    );
    assert_eq!(
        d2_layout.guardian_set_index, scenario.guardian_set_index,
        "D2 bucket recorded the active guardian set"
    );
}

/// Fork-recovery: two digests for the same `(chain, emitter, sequence)` under
/// one guardian set accumulate in separate sibling PDAs and race to quorum.
/// The digest-in-seeds design substitutes for CosmWasm's `tx_hash` bucket
/// discriminator; NoReplay (keyed by `(chain, emitter, sequence)` only) is
/// shared across siblings, so the losing digest's PDA is later reclaimable via
/// `close_pending`.
#[test]
fn fork_recovery_different_digest_same_seq_under_same_set_both_accumulate() {
    // 1. 7 guardians observe D1 under set 6.
    // 2-4. A reorg flips the body timestamp; 13 guardians observe D2 in a
    //      fresh sibling PDA.
    // 5. D2 reaches quorum: NoReplay flips, DigestAccount opens with D2, D2
    //    pending closes.
    // 6. The D1 pending PDA (7 sigs) is stranded.

    let mollusk = mollusk();
    let scenario = Scenario::new(19, 6, 0x5A);

    // Accumulate 7 D1 signatures.
    let accounts_after_d1 = scenario.submit_n(&mollusk, 7);
    let d1_after = find_account(&accounts_after_d1, &scenario.pending_pda);
    let d1_layout: &PendingObservationsLayout = bytemuck::from_bytes(&d1_after.data);
    assert_eq!(
        d1_layout.signatures.count_ones(),
        7,
        "D1 bucket accumulated 7 sigs before reorg"
    );

    // Switch to D2. Mutate only consistency_level (byte 50) so the routing
    // tuple at body[8..50] is unchanged.
    let mut alternate_body = scenario.body.clone();
    alternate_body[50] = 0xA5;
    let alternate_digest = double_keccak256_host(&alternate_body);
    assert_ne!(alternate_digest, scenario.digest);

    let (d2_pending_pda, _) = derive_pending_pda(
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
        &alternate_digest,
    );
    assert_ne!(d2_pending_pda, scenario.pending_pda);

    let mut accounts = accounts_after_d1.clone();
    accounts.push((d2_pending_pda, uninitialised_pda_account()));

    // Drive 13 D2 observations to quorum.
    for i in 0..13u8 {
        let signature = sign_digest(&scenario.guardians[i as usize], &alternate_digest);
        let mut metas = scenario.account_metas();
        metas[1] = AccountMeta::new(d2_pending_pda, false);
        let ix = Instruction::new_with_bytes(
            program_id(),
            &submit_ix_data(
                &alternate_digest,
                scenario.guardian_set_index,
                i,
                &signature,
                &alternate_body,
            ),
            metas,
        );
        let r = mollusk.process_instruction(&ix, &accounts);
        assert!(
            matches!(r.program_result, ProgramResult::Success),
            "D2 submit #{i} expected success, got {:?}",
            r.program_result
        );
        accounts = r.resulting_accounts.clone();
    }

    // Post-quorum: NoReplay flipped, DigestAccount = D2, D2 pending closed,
    // D1 pending stranded at 7 sigs.
    let bucket = find_account(&accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        Pubkey::new_from_array(NOREPLAY_PROGRAM_ID),
        "NoReplay flipped on D2 quorum reach (shared bucket across siblings)"
    );
    assert_eq!(
        bucket.data.len(),
        NOREPLAY_BITMAP_OFFSET + NOREPLAY_BITMAP_BYTES
    );

    // The winning digest (D2) is emitted via `sol_log_data` on the
    // quorum-completing branch; log content is verified by the surfpool e2e
    // suite (mollusk does not expose `program_logs`).

    let d2_post = find_account(&accounts, &d2_pending_pda);
    assert_eq!(d2_post.lamports, 0, "D2 pending PDA drained on commit");
    assert!(d2_post.data.is_empty(), "D2 pending PDA closed");
    assert_eq!(d2_post.owner, system_program_id());

    let d1_post = find_account(&accounts, &scenario.pending_pda);
    assert_eq!(
        d1_post.owner,
        program_id(),
        "D1 pending PDA stranded but still owned by program"
    );
    let d1_post_layout: &PendingObservationsLayout = bytemuck::from_bytes(&d1_post.data);
    assert_eq!(
        d1_post_layout.signatures.count_ones(),
        7,
        "D1 pending PDA still holds 7 sigs after D2 sibling reached quorum"
    );
    assert_eq!(
        d1_post_layout.digest, scenario.digest,
        "D1 pending PDA still records the original digest"
    );
}

/// A marked NoReplay bucket aborts submit before any signature or PDA work.
#[test]
fn submit_rejected_when_noreplay_already_marked() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x4D);

    let mut accounts = scenario.initial_accounts();
    if let Some(entry) = accounts
        .iter_mut()
        .find(|(k, _)| *k == scenario.noreplay_bucket_pubkey)
    {
        entry.1 = noreplay_bucket_marked(scenario.sequence);
    }

    let r = scenario.submit_once(&mollusk, accounts, 0);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::AlreadyAccounted as u32,
                "expected AlreadyAccounted, got {code:?}"
            );
        }
        other => panic!("expected Failure(AlreadyAccounted), got {other:?}"),
    }
}

// ============================================================================
// Balance-accounting tests. The quorum-completing observation parses the
// Token Bridge payload and routes balance updates through
// `lock_or_burn` / `unlock_or_mint` against the source/dest Account PDAs.
// ============================================================================

/// Drive 13 observations against a transfer scenario, returning the final
/// `InstructionResult`.
fn drive_transfer_to_quorum(
    mollusk: &Mollusk,
    scenario: &Scenario,
) -> mollusk_svm::result::InstructionResult {
    let mut accounts = scenario.initial_accounts();
    for i in 0..PendingObservationsLayout::QUORUM_THRESHOLD as u8 {
        let result = scenario.submit_once(mollusk, accounts.clone(), i);
        assert!(
            matches!(result.program_result, ProgramResult::Success),
            "submit #{i} expected success, got {:?}",
            result.program_result
        );
        accounts = result.resulting_accounts.clone();
        // Return the final InstructionResult so callers can inspect CU usage.
        if i + 1 == PendingObservationsLayout::QUORUM_THRESHOLD as u8 {
            return result;
        }
    }
    unreachable!("loop above always returns on the final iteration")
}

/// Transfer of Ethereum-native USDC (chain=2) to Solana (chain=1): source
/// (native) credits, dest (wrapped) credits; both Account PDAs lazy-init.
#[test]
fn quorum_with_transfer_credits_native_chain_and_mints_wrapped_chain() {
    let mollusk = mollusk();
    let token_address = [0x77u8; 32];
    let scenario = Scenario::with_transfer_body(
        19,
        4,
        0x60,
        500_000u128,
        2, // token_chain = source-native (Ethereum)
        token_address,
        1, // recipient_chain = Solana (wrapped destination)
    );
    let result = drive_transfer_to_quorum(&mollusk, &scenario);
    assert!(
        matches!(result.program_result, ProgramResult::Success),
        "quorum tx must succeed, got {:?}",
        result.program_result
    );

    // Source (chain == token_chain == 2): native lock ⇒ credit.
    let src = find_account(&result.resulting_accounts, &scenario.source_account_pubkey);
    assert_eq!(
        src.owner,
        program_id(),
        "source Account PDA owned by program"
    );
    assert_eq!(src.data.len(), BalanceAccountLayout::LEN);
    let src_layout: &BalanceAccountLayout = bytemuck::from_bytes(&src.data);
    assert_eq!(src_layout.chain, 2);
    assert_eq!(src_layout.token_chain, 2);
    assert_eq!(src_layout.token_address, token_address);
    assert_eq!(src_layout.balance, Uint256::from_u128(500_000));

    // Dest (chain 1 != token_chain 2): wrapped mint ⇒ credit.
    let dst = find_account(&result.resulting_accounts, &scenario.dest_account_pubkey);
    assert_eq!(dst.owner, program_id(), "dest Account PDA owned by program");
    let dst_layout: &BalanceAccountLayout = bytemuck::from_bytes(&dst.data);
    assert_eq!(dst_layout.chain, 1);
    assert_eq!(dst_layout.token_chain, 2);
    assert_eq!(dst_layout.balance, Uint256::from_u128(500_000));
}

/// Reverse direction (Solana wUSDC back to Ethereum): the wrapped-source debit
/// underflows from zero and the whole tx reverts.
#[test]
fn quorum_with_transfer_underflows_when_wrapped_chain_has_insufficient_balance() {
    let mollusk = mollusk();
    let token_address = [0x88u8; 32];
    let mut scenario = Scenario::with_transfer_body(
        19,
        4,
        0x61,
        1_000u128,
        2, // token_chain = Ethereum (token-native)
        token_address,
        2, // recipient_chain = Ethereum
    );
    // VAA emitter is Solana, not Ethereum.
    scenario.chain = 1;
    scenario.body = build_transfer_body(
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
        1_000,
        2,
        &token_address,
        2,
    );
    scenario.digest = double_keccak256_host(&scenario.body);
    let (pending_pda, _) = derive_pending_pda(
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
        &scenario.digest,
    );
    scenario.pending_pda = pending_pda;
    let (src, _) = derive_account_pda(1, 2, &token_address);
    let (dst, _) = derive_account_pda(2, 2, &token_address);
    // Re-derive registration PDA and noreplay bucket for the new chain.
    let (registration_pda, _) = derive_chain_registration_pda(scenario.chain);
    scenario.chain_registration_pubkey = registration_pda;
    scenario.noreplay_bucket_pubkey = derive_canonical_noreplay_bucket(
        &scenario.noreplay_authority_pubkey,
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
    );
    scenario.source_account_pubkey = src;
    scenario.dest_account_pubkey = dst;

    let mut accounts = scenario.initial_accounts();
    for i in 0..12u8 {
        let r = scenario.submit_once(&mollusk, accounts.clone(), i);
        assert!(matches!(r.program_result, ProgramResult::Success));
        accounts = r.resulting_accounts;
    }
    // 13th observation: quorum + balance work ⇒ underflow.
    let r = scenario.submit_once(&mollusk, accounts, 12);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::BalanceUnderflow as u32,
                "expected BalanceUnderflow, got {code:?}"
            );
        }
        other => panic!("expected Failure(BalanceUnderflow), got {other:?}"),
    }
    // Tx rolled back: bucket stays untouched (the commit log is similarly
    // unwound by tx-level atomicity, but mollusk does not expose program logs
    // for inspection here — surfpool e2e covers the positive log assertion).
    let bucket = find_account(&r.resulting_accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        system_program_id(),
        "bucket still system-owned"
    );
    assert!(
        bucket.data.is_empty(),
        "NoReplay must not flip on failed quorum"
    );
}

/// A fresh destination Account PDA is lazy-initialised under the program on
/// quorum.
#[test]
fn quorum_with_lazy_init_destination_account_succeeds() {
    let mollusk = mollusk();
    let token_address = [0x42u8; 32];
    let scenario = Scenario::with_transfer_body(19, 4, 0x62, 9_999u128, 2, token_address, 1);

    let initial = scenario.initial_accounts();
    let dst_pre = find_account(&initial, &scenario.dest_account_pubkey);
    assert_eq!(
        dst_pre.owner,
        system_program_id(),
        "dest Account PDA must start system-owned"
    );
    assert_eq!(
        dst_pre.data.len(),
        0,
        "dest Account PDA must start with zero data"
    );

    let result = drive_transfer_to_quorum(&mollusk, &scenario);
    assert!(matches!(result.program_result, ProgramResult::Success));

    let dst_post = find_account(&result.resulting_accounts, &scenario.dest_account_pubkey);
    assert_eq!(
        dst_post.owner,
        program_id(),
        "dest lazy-init flips owner to program"
    );
    assert_eq!(
        dst_post.data.len(),
        BalanceAccountLayout::LEN,
        "dest data sized to full layout"
    );
    let layout: &BalanceAccountLayout = bytemuck::from_bytes(&dst_post.data);
    assert_eq!(layout.balance, Uint256::from_u128(9_999));
}

/// CU regression guard: the quorum-commit branch (lazy-init of both Account
/// PDAs — the program's most expensive tx) must stay below `MAX_QUORUM_BRANCH_CU`.
#[test]
fn quorum_branch_cu_stays_below_ceiling() {
    let mollusk = mollusk();
    let token_address = [0xCFu8; 32];
    let scenario = Scenario::with_transfer_body(
        19,
        4,
        0xCF,
        12_345u128,
        2,
        token_address,
        1, // recipient_chain = Solana so both Account PDAs lazy-init
    );

    // First 12 (accumulator path) without measuring CU.
    let mut accounts = scenario.initial_accounts();
    for i in 0..(PendingObservationsLayout::QUORUM_THRESHOLD as u8 - 1) {
        let r = scenario.submit_once(&mollusk, accounts.clone(), i);
        assert!(matches!(r.program_result, ProgramResult::Success));
        accounts = r.resulting_accounts;
    }

    // 13th observation — full commit branch.
    let result = scenario.submit_once(
        &mollusk,
        accounts,
        PendingObservationsLayout::QUORUM_THRESHOLD as u8 - 1,
    );
    assert!(
        matches!(result.program_result, ProgramResult::Success),
        "quorum tx must succeed for the CU measurement to be meaningful, got {:?}",
        result.program_result
    );

    assert!(
        result.compute_units_consumed <= MAX_QUORUM_BRANCH_CU,
        "quorum-branch CU ({}) exceeded ceiling ({}); investigate before raising the constant",
        result.compute_units_consumed,
        MAX_QUORUM_BRANCH_CU
    );
}

/// Attest quorum completes the commit branch but touches neither Account PDA.
#[test]
fn quorum_with_attest_payload_skips_balance_work_but_finishes_commit() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x63);
    let result = drive_transfer_to_quorum(&mollusk, &scenario);
    assert!(
        matches!(result.program_result, ProgramResult::Success),
        "attest quorum tx must succeed, got {:?}",
        result.program_result
    );

    // Sentinel Account PDA slots stay system-owned.
    assert_eq!(
        scenario.source_account_pubkey,
        scenario.noreplay_authority_pubkey
    );
    let sentinel = find_account(
        &result.resulting_accounts,
        &scenario.noreplay_authority_pubkey,
    );
    assert_eq!(
        sentinel.owner,
        system_program_id(),
        "sentinel slot stays system-owned across attest commit"
    );
    assert!(
        sentinel.data.is_empty(),
        "sentinel slot data untouched across attest commit"
    );
}

/// An unknown payload action rejects at quorum with `UnknownTokenBridgePayload`,
/// rolling back the NoReplay mark so the replay slot stays unconsumed.
#[test]
fn quorum_with_unknown_payload_rejects_and_preserves_replay_slot() {
    let mollusk = mollusk();
    let mut scenario = Scenario::new(19, 4, 0x65);
    scenario.body[51] = 0x05; // unknown Token Bridge action byte
    scenario.digest = double_keccak256_host(&scenario.body);
    let (pending_pda, _) = derive_pending_pda(
        scenario.chain,
        &scenario.emitter,
        scenario.sequence,
        &scenario.digest,
    );
    scenario.pending_pda = pending_pda;

    // Payload is parsed only at quorum; the first 12 accumulate normally.
    let accounts = scenario.submit_n(&mollusk, 12);

    let result = scenario.submit_once(&mollusk, accounts, 12);
    match result.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::UnknownTokenBridgePayload as u32,
                "expected UnknownTokenBridgePayload, got code {code}"
            );
        }
        other => panic!("expected Failure(UnknownTokenBridgePayload), got {other:?}"),
    }

    // NoReplay mark reverted: bucket stays uninitialised.
    let bucket = find_account(&result.resulting_accounts, &scenario.noreplay_bucket_pubkey);
    assert_eq!(
        bucket.owner,
        system_program_id(),
        "noreplay bucket must stay system-owned after rejection"
    );
    assert!(
        bucket.data.is_empty(),
        "noreplay bucket data untouched after rejection"
    );

    // Pending bucket survives with its 12 signatures.
    let pending = find_account(&result.resulting_accounts, &scenario.pending_pda);
    assert_eq!(pending.owner, program_id(), "pending PDA still live");
    let layout: &PendingObservationsLayout = bytemuck::from_bytes(&pending.data);
    assert_eq!(
        layout.signatures.count_ones(),
        12,
        "12 signatures still recorded in the surviving bucket"
    );
}

/// A body that doesn't double-keccak to the supplied digest is rejected with
/// `BodyDigestMismatch` before any PDA work.
#[test]
fn quorum_with_body_digest_mismatch_rejects() {
    let mollusk = mollusk();
    let scenario = Scenario::new(19, 4, 0x64);

    let mut tampered_body = scenario.body.clone();
    tampered_body[0] ^= 0xAA; // mutate the timestamp byte
    let signature = sign_digest(&scenario.guardians[0], &scenario.digest);
    let ix = Instruction::new_with_bytes(
        program_id(),
        &submit_ix_data(
            &scenario.digest,
            scenario.guardian_set_index,
            0,
            &signature,
            &tampered_body, // body doesn't double-keccak to `digest`
        ),
        scenario.account_metas(),
    );
    let r = mollusk.process_instruction(&ix, &scenario.initial_accounts());
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::BodyDigestMismatch as u32,
                "expected BodyDigestMismatch, got {code:?}"
            );
        }
        other => panic!("expected Failure(BodyDigestMismatch), got {other:?}"),
    }
    // Pending PDA stays uninitialised.
    let pending = find_account(&r.resulting_accounts, &scenario.pending_pda);
    assert_eq!(pending.owner, system_program_id());
    assert!(pending.data.is_empty());
}

/// A wrong-seed source Account PDA in slot 8 is rejected at quorum with
/// `InvalidAccountPda`.
#[test]
fn quorum_with_invalid_source_account_pda_rejects() {
    let mollusk = mollusk();
    let token_address = [0x99u8; 32];
    let mut scenario = Scenario::with_transfer_body(19, 4, 0x65, 100u128, 2, token_address, 1);
    // Spoofed source PDA (wrong token chain).
    let (spoofed, _) = derive_account_pda(2, 99, &token_address);
    scenario.source_account_pubkey = spoofed;

    let mut accounts = scenario.initial_accounts();
    for i in 0..12u8 {
        let r = scenario.submit_once(&mollusk, accounts.clone(), i);
        assert!(matches!(r.program_result, ProgramResult::Success));
        accounts = r.resulting_accounts;
    }
    let r = scenario.submit_once(&mollusk, accounts, 12);
    match r.program_result {
        ProgramResult::Failure(err) => {
            let code = u64::from(err) as u32;
            assert_eq!(
                code,
                GlobalAccountantError::InvalidAccountPda as u32,
                "expected InvalidAccountPda, got {code:?}"
            );
        }
        other => panic!("expected Failure(InvalidAccountPda), got {other:?}"),
    }
}
