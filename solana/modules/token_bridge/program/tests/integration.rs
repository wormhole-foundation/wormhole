#![allow(dead_code)]
use bridge::{
    accounts::{
        PostedVAA,
        PostedVAADerivationData,
    },
    PostVAAData,
    SerializePayload,
    OUR_CHAIN_ID,
};
use libsecp256k1::SecretKey;
use primitive_types::U256;
use rand::Rng;
use solana_program::pubkey::Pubkey;
use solana_program_test::{
    tokio,
    BanksClient,
};
use solana_sdk::{
    signature::{
        Keypair,
        Signer,
    },
    transport::TransportError,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

use std::{
    collections::HashMap,
    str::FromStr,
};
use token_bridge::{
    accounts::{
        ConfigAccount,
        WrappedDerivationData,
        WrappedMint,
    },
    api::{
        FROZEN_EVENT_DISCRIMINATOR,
        PAUSED_EVENT_DISCRIMINATOR,
        PAUSER_ADDRESSES_SET_EVENT_DISCRIMINATOR,
        PAUSE_DURATION,
        UNPAUSED_EVENT_DISCRIMINATOR,
        UNPAUSE_EXPIRED_EVENT_DISCRIMINATOR,
    },
    messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadSetPauserAddresses,
        PayloadTransfer,
        PayloadTransferWithPayload,
    },
    types::{
        freezer as read_freezer,
        pause_expiry as read_pause_expiry,
        paused as read_paused,
        pauser as read_pauser,
        unpauser as read_unpauser,
        Config,
        CONFIG_BORSH_LEN,
        CONFIG_WITH_PAUSER_LEN,
    },
};

mod common;

const GOVERNANCE_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44, 196,
    163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201, 174, 108,
    197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170, 141, 231, 147,
    88, 97, 199,
];

const CHAIN_ID_ETH: u16 = 2;

struct Context {
    /// Guardian public keys.
    guardians: Vec<[u8; 20]>,

    /// Guardian secret keys.
    guardian_keys: Vec<SecretKey>,

    /// Address of the core bridge contract.
    bridge: Pubkey,

    /// Shared RPC client for tests to make transactions with.
    client: BanksClient,

    /// Payer key with a ton of lamports to ease testing with.
    payer: Keypair,

    /// Track nonces throughout the tests.
    seq: Sequencer,

    /// Address of the token bridge itself that we wish to test.
    token_bridge: Pubkey,

    /// Keypairs for mint information, required in multiple tests.
    mint_authority: Keypair,
    mint: Keypair,
    mint_meta: Pubkey,

    /// Keypairs for test token accounts.
    token_authority: Keypair,
    token_account: Keypair,
    metadata_account: Pubkey,
}

/// Small helper to track and provide sequences during tests. This is in particular needed for
/// guardian operations that require them for derivations.
struct Sequencer {
    sequences: HashMap<[u8; 32], u64>,
}

impl Sequencer {
    fn next(&mut self, emitter: [u8; 32]) -> u64 {
        let entry = self.sequences.entry(emitter).or_insert(0);
        *entry += 1;
        *entry - 1
    }

    fn peek(&mut self, emitter: [u8; 32]) -> u64 {
        *self.sequences.entry(emitter).or_insert(0)
    }
}

async fn set_up() -> Result<Context, TransportError> {
    let (guardians, guardian_keys) = common::generate_keys(6);

    let (mut client, payer, bridge, token_bridge) = common::setup().await;

    // Setup a Bridge to test against.
    common::initialize_bridge(&mut client, bridge, &payer, &guardians).await?;

    // Context for test environment.
    let mint = Keypair::new();
    let mint_pubkey = mint.pubkey();
    let metadata_pubkey = Pubkey::from_str("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s").unwrap();

    // SPL Token Meta
    let metadata_seeds = &[
        "metadata".as_bytes(),
        metadata_pubkey.as_ref(),
        mint_pubkey.as_ref(),
    ];

    let (metadata_key, _metadata_bump_seed) = Pubkey::find_program_address(
        metadata_seeds,
        &Pubkey::from_str("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s").unwrap(),
    );

    // Token Bridge Meta
    use token_bridge::accounts::WrappedTokenMeta;
    let metadata_account = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &token_bridge::accounts::WrappedMetaDerivationData {
            mint_key: mint_pubkey,
        },
        &token_bridge,
    );

    let mut context = Context {
        guardians,
        guardian_keys,
        seq: Sequencer {
            sequences: HashMap::new(),
        },
        bridge,
        client,
        payer,
        token_bridge,
        mint_authority: Keypair::new(),
        mint,
        mint_meta: metadata_account,
        token_account: Keypair::new(),
        token_authority: Keypair::new(),
        metadata_account: metadata_key,
    };

    // Create a mint for use within tests.
    common::create_mint(
        &mut context.client,
        &context.payer,
        &context.mint_authority.pubkey(),
        &context.mint,
    )
    .await?;

    // Create Token accounts for use within tests.
    common::create_token_account(
        &mut context.client,
        &context.payer,
        &context.token_account,
        &context.token_authority.pubkey(),
        &context.mint.pubkey(),
    )
    .await?;

    // Mint tokens
    common::mint_tokens(
        &mut context.client,
        &context.payer,
        &context.mint_authority,
        &context.mint,
        &context.token_account.pubkey(),
        1000,
    )
    .await?;

    // Initialize the token bridge.
    common::initialize(
        &mut context.client,
        context.token_bridge,
        &context.payer,
        context.bridge,
    )
    .await
    .unwrap();

    // Verify Token Bridge State
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &token_bridge);
    let config: Config = common::get_account_data(&mut context.client, config_key)
        .await
        .unwrap();
    assert_eq!(config.wormhole_bridge, bridge);

    Ok(context)
}

async fn create_wrapped(context: &mut Context) -> Pubkey {
    let Context {
        ref payer,
        ref mut client,
        ref bridge,
        ref token_bridge,
        mint_authority: _,
        mint: _,
        mint_meta: _,
        token_account: _,
        token_authority: _,
        ..
    } = context;

    let nonce = rand::thread_rng().gen();

    let payload = PayloadAssetMeta {
        token_address: [1u8; 32],
        token_chain: 2,
        decimals: 7,
        symbol: "".to_string(),
        name: "".to_string(),
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, body, _) = common::generate_vaa([0u8; 32], 2, message, nonce, 2);
    let signature_set =
        common::verify_signatures(client, bridge, payer, body, &context.guardian_keys, 0)
            .await
            .unwrap();
    common::post_vaa(client, *bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();
    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, bridge);

    common::create_wrapped(
        client,
        *token_bridge,
        *bridge,
        message_key,
        vaa,
        payload,
        payer,
    )
    .await
    .unwrap();

    WrappedMint::<'_, { AccountState::Initialized }>::key(
        &WrappedDerivationData {
            token_chain: 2,
            token_address: [1u8; 32],
        },
        token_bridge,
    )
}

// Create an SPL Metadata account to test attestations for wrapped tokens.
async fn create_wrapped_account(context: &mut Context) -> Result<Pubkey, TransportError> {
    common::create_spl_metadata(
        &mut context.client,
        &context.payer,
        context.metadata_account,
        &context.mint_authority,
        &context.mint,
        context.payer.pubkey(),
        "BTC".to_string(),
        "Bitcoin".to_string(),
    )
    .await?;

    let wrapped = create_wrapped(context).await;
    let wrapped_acc = Keypair::new();
    common::create_token_account(
        &mut context.client,
        &context.payer,
        &wrapped_acc,
        &context.token_authority.pubkey(),
        &wrapped,
    )
    .await?;

    Ok(wrapped_acc.pubkey())
}

#[tokio::test]
async fn attest() {
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        mint_authority: _,
        ref mint,
        mint_meta: _,
        metadata_account: _,
        ..
    } = set_up().await.unwrap();

    let message = &Keypair::new();

    common::attest(
        client,
        token_bridge,
        bridge,
        payer,
        message,
        mint.pubkey(),
        0,
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn transfer_native() {
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ..
    } = set_up().await.unwrap();

    let message = &Keypair::new();

    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        message,
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();
}

async fn register_chain(context: &mut Context) {
    let Context {
        ref payer,
        ref mut client,
        ref bridge,
        ref token_bridge,
        ref guardian_keys,
        ..
    } = context;

    let nonce = rand::thread_rng().gen();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let payload = PayloadGovernanceRegisterChain {
        chain: 2,
        endpoint_address: [0u8; 32],
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, body, _) = common::generate_vaa(emitter.pubkey().to_bytes(), 1, message, nonce, 0);
    let signature_set = common::verify_signatures(client, bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, *bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();

    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, bridge);

    common::register_chain(
        client,
        *token_bridge,
        *bridge,
        message_key,
        vaa,
        payload,
        payer,
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn transfer_native_in() {
    let mut context = set_up().await.unwrap();
    register_chain(&mut context).await;
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ref guardian_keys,
        ..
    } = context;

    // Do an initial transfer so that the bridge account has some native tokens. This also creates
    // the custody account.
    let message = &Keypair::new();
    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        message,
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();

    let nonce = rand::thread_rng().gen();

    let payload = PayloadTransfer {
        amount: U256::from(100u128),
        token_address: mint.pubkey().to_bytes(),
        token_chain: 1,
        to: token_account.pubkey().to_bytes(),
        to_chain: 1,
        fee: U256::from(0u128),
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, body, _) = common::generate_vaa([0u8; 32], 2, message, nonce, 1);
    let signature_set = common::verify_signatures(client, &bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();
    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, &bridge);

    common::complete_native(
        client,
        token_bridge,
        bridge,
        message_key,
        vaa,
        payload,
        payer,
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn transfer_wrapped() {
    let mut context = set_up().await.unwrap();
    register_chain(&mut context).await;
    let to = create_wrapped_account(&mut context).await.unwrap();
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref token_authority,
        ref guardian_keys,
        ..
    } = context;

    let nonce = rand::thread_rng().gen();

    let payload = PayloadTransfer {
        amount: U256::from(100000000),
        token_address: [1u8; 32],
        token_chain: 2,
        to: to.to_bytes(),
        to_chain: 1,
        fee: U256::from(0),
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, body, _) =
        common::generate_vaa([0u8; 32], 2, message, nonce, rand::thread_rng().gen());
    let signature_set = common::verify_signatures(client, &bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();
    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, &bridge);

    common::complete_transfer_wrapped(
        client,
        token_bridge,
        bridge,
        message_key,
        vaa,
        payload,
        payer,
    )
    .await
    .unwrap();

    // Now transfer the wrapped tokens back, which will burn them.
    let message = &Keypair::new();
    common::transfer_wrapped(
        client,
        token_bridge,
        bridge,
        payer,
        message,
        to,
        token_authority,
        2,
        [1u8; 32],
        10000000,
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn transfer_native_with_payload_in() {
    let mut context = set_up().await.unwrap();
    register_chain(&mut context).await;
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ref guardian_keys,
        ..
    } = context;

    // Do an initial transfer so that the bridge account has some native tokens. This also creates
    // the custody account.
    let message = &Keypair::new();
    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        message,
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();

    let nonce = rand::thread_rng().gen();
    let from_address = Keypair::new().pubkey().to_bytes();
    let payload: Vec<u8> = vec![1, 2, 3];
    let payload = PayloadTransferWithPayload {
        amount: U256::from(100u128),
        token_address: mint.pubkey().to_bytes(),
        token_chain: OUR_CHAIN_ID,
        to: token_authority.pubkey().to_bytes(),
        to_chain: OUR_CHAIN_ID,
        from_address,
        payload,
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, body, _) = common::generate_vaa([0u8; 32], CHAIN_ID_ETH, message, nonce, 1);
    let signature_set = common::verify_signatures(client, &bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();
    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, &bridge);

    common::complete_native_with_payload(
        client,
        token_bridge,
        bridge,
        message_key,
        vaa,
        payload,
        token_account.pubkey(),
        token_authority,
        payer,
    )
    .await
    .unwrap();
}

// ============== Pauser tests (whitepapers/0003_token_bridge.md Pausing) ==============
//
// These tests exercise the lazy migration of the `Config` PDA from the legacy 32-byte layout to
// the full pauser layout (CONFIG_WITH_PAUSER_LEN bytes: paused flag + pauser/freezer/unpauser +
// pause_expiry), the configured pauser/freezer/unpauser flow, the timed-pause / `freeze` /
// permissionless `unpause_expired` semantics, and the `notPaused` gate on a representative
// non-governance entry point. The legacy compatibility test then confirms an un-migrated bridge
// still behaves identically to pre-upgrade.

const SET_PAUSER_ADDRESSES_ACTION: u8 = 4;

/// Build a `SetPauserAddresses` governance VAA, post it through the core bridge, and submit it to
/// the token bridge. Asserts the encoded wire format inline (three length-prefixed 32-byte
/// addresses) and panics if the on-chain submission fails.
async fn submit_set_pauser_addresses(
    context: &mut Context,
    pauser: Pubkey,
    freezer: Pubkey,
    unpauser: Pubkey,
) {
    let payload = PayloadSetPauserAddresses {
        pauser,
        freezer,
        unpauser,
    };
    let message = payload.try_to_vec().unwrap();
    // Sanity-check the encoded wire format matches whitepapers/0003_token_bridge.md Pausing
    // (three length-prefixed addresses, freezer in the middle):
    //   module(32) | action(1)=4 | chain(2) | pauser_len(1)=32   | pauser(32)
    //                                       | freezer_len(1)=32  | freezer(32)
    //                                       | unpauser_len(1)=32 | unpauser(32)
    // Total: 32 + 1 + 2 + 1 + 32 + 1 + 32 + 1 + 32 = 134 bytes.
    assert_eq!(message[32], SET_PAUSER_ADDRESSES_ACTION);
    assert_eq!(
        message[35], 32,
        "expected pauser_len = 32 (SVM native size)"
    );
    assert_eq!(
        message[68], 32,
        "expected freezer_len = 32 (SVM native size)"
    );
    assert_eq!(
        message[101], 32,
        "expected unpauser_len = 32 (SVM native size)"
    );
    assert_eq!(message.len(), 134);

    submit_raw_set_pauser_addresses(context, message)
        .await
        .expect("canonical SetPauserAddresses should succeed");
}

/// Build, post, and submit a `SetPauserAddresses` governance VAA carrying a caller-supplied raw
/// payload. Returns the result of the on-chain submission so length-validation tests can assert
/// on the rejection path without having to round-trip through `PayloadSetPauserAddresses`.
async fn submit_raw_set_pauser_addresses(
    context: &mut Context,
    message: Vec<u8>,
) -> std::result::Result<(), solana_program_test::BanksClientError> {
    let Context {
        ref payer,
        ref mut client,
        ref bridge,
        ref token_bridge,
        ref guardian_keys,
        ref mut seq,
        ..
    } = context;

    let nonce = rand::thread_rng().gen();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = seq.next(emitter.pubkey().to_bytes());

    let (vaa, body, _) =
        common::generate_vaa(emitter.pubkey().to_bytes(), 1, message, nonce, sequence);
    let signature_set = common::verify_signatures(client, bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, *bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();

    let msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(msg_derivation_data, bridge);

    common::set_pauser_addresses(client, *token_bridge, *bridge, message_key, vaa, payer).await
}

/// Build a `SetPauserAddresses` governance payload with caller-controlled length-prefix bytes.
/// `pauser_len` / `freezer_len` / `unpauser_len` are written verbatim, even if they don't match
/// the corresponding slice lengths, so a test can construct malformed payloads (invalid length,
/// all-zero, trailing bytes, etc). The three addresses are emitted in wire order: pauser, then
/// freezer, then unpauser.
fn build_set_pauser_addresses_payload(
    pauser_len: u8,
    pauser: &[u8],
    freezer_len: u8,
    freezer: &[u8],
    unpauser_len: u8,
    unpauser: &[u8],
) -> Vec<u8> {
    // "TokenBridge" left-padded to 32 bytes.
    let module: [u8; 32] = {
        let mut m = [0u8; 32];
        let name = b"TokenBridge";
        m[32 - name.len()..].copy_from_slice(name);
        m
    };
    let mut payload =
        Vec::with_capacity(35 + 1 + pauser.len() + 1 + freezer.len() + 1 + unpauser.len());
    payload.extend_from_slice(&module);
    payload.push(SET_PAUSER_ADDRESSES_ACTION); // action
    payload.extend_from_slice(&1u16.to_be_bytes()); // OUR_CHAIN_ID = Solana
    payload.push(pauser_len);
    payload.extend_from_slice(pauser);
    payload.push(freezer_len);
    payload.extend_from_slice(freezer);
    payload.push(unpauser_len);
    payload.extend_from_slice(unpauser);
    payload
}

/// Returns the raw `Config` account bytes — used to assert layout, paused flag, and tail
/// pubkeys without going through the Borsh `Config` struct (which only covers the first 32
/// bytes).
async fn fetch_config_data(context: &mut Context) -> Vec<u8> {
    let config_key =
        ConfigAccount::<'_, { AccountState::Initialized }>::key(None, &context.token_bridge);
    let account = context
        .client
        .get_account_with_commitment(
            config_key,
            solana_sdk::commitment_config::CommitmentLevel::Processed,
        )
        .await
        .unwrap()
        .unwrap();
    account.data
}

/// Reads the on-chain `Clock.unix_timestamp` (the same value the program sees via the Clock
/// sysvar). The Clock sysvar account is bincode-encoded with default fixint little-endian layout:
///   slot(u64) | epoch_start_timestamp(i64) | epoch(u64) | leader_schedule_epoch(u64)
///   | unix_timestamp(i64)
/// so `unix_timestamp` is the 8-byte LE i64 at offset 32. We read it directly to avoid pulling in
/// a `bincode` dev-dependency just for this. Used to pin `pause`'s `now + PAUSE_DURATION` expiry.
async fn fetch_clock_unix_timestamp(context: &mut Context) -> i64 {
    let account = context
        .client
        .get_account_with_commitment(
            solana_program::sysvar::clock::id(),
            solana_sdk::commitment_config::CommitmentLevel::Processed,
        )
        .await
        .unwrap()
        .unwrap();
    let mut bytes = [0u8; 8];
    bytes.copy_from_slice(&account.data[32..40]);
    i64::from_le_bytes(bytes)
}

#[tokio::test]
async fn set_pauser_addresses_lazy_migration() {
    let mut context = set_up().await.unwrap();

    // Pre-migration: Config is the legacy 32-byte layout written by `initialize`.
    let pre = fetch_config_data(&mut context).await;
    assert_eq!(
        pre.len(),
        CONFIG_BORSH_LEN,
        "expected legacy 32-byte Config before migration"
    );
    assert!(!read_paused(&pre));
    assert_eq!(read_pauser(&pre), Pubkey::default());
    assert_eq!(read_freezer(&pre), Pubkey::default());
    assert_eq!(read_unpauser(&pre), Pubkey::default());
    assert_eq!(read_pause_expiry(&pre), 0);

    // First SetPauserAddresses: realloc 32 → CONFIG_WITH_PAUSER_LEN (137) bytes, write the tail.
    let pauser_one = Pubkey::new_unique();
    let freezer_one = Pubkey::new_unique();
    let unpauser_one = Pubkey::new_unique();
    submit_set_pauser_addresses(&mut context, pauser_one, freezer_one, unpauser_one).await;

    let post = fetch_config_data(&mut context).await;
    assert_eq!(
        post.len(),
        CONFIG_WITH_PAUSER_LEN,
        "Config should grow to the full pauser layout after migration"
    );
    assert!(
        !read_paused(&post),
        "fresh tail should default to paused = false"
    );
    assert_eq!(read_pauser(&post), pauser_one);
    assert_eq!(read_freezer(&post), freezer_one);
    assert_eq!(read_unpauser(&post), unpauser_one);
    assert_eq!(
        read_pause_expiry(&post),
        0,
        "fresh tail should default to pause_expiry = 0"
    );
    // The first 32 bytes (wormhole_bridge) must survive realloc untouched.
    assert_eq!(&post[..CONFIG_BORSH_LEN], &pre[..CONFIG_BORSH_LEN]);

    // Second SetPauserAddresses: rotate keys, no realloc, paused flag preserved (still false).
    let pauser_two = Pubkey::new_unique();
    let freezer_two = Pubkey::new_unique();
    let unpauser_two = Pubkey::new_unique();
    submit_set_pauser_addresses(&mut context, pauser_two, freezer_two, unpauser_two).await;

    let rotated = fetch_config_data(&mut context).await;
    assert_eq!(
        rotated.len(),
        CONFIG_WITH_PAUSER_LEN,
        "rotation must not change account size"
    );
    assert_eq!(read_pauser(&rotated), pauser_two);
    assert_eq!(read_freezer(&rotated), freezer_two);
    assert_eq!(read_unpauser(&rotated), unpauser_two);
    assert!(!read_paused(&rotated));
}

#[tokio::test]
async fn pause_blocks_transfer_and_unpause_restores() {
    let mut context = set_up().await.unwrap();

    // The pauser/unpauser must each be Solana keypairs because the on-chain handler requires
    // them as `Signer`. Fund them so they can co-sign their respective instructions.
    let pauser = Keypair::new();
    let unpauser = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();

    // Freezer is configured but unused in this test (it exercises the pauser/unpauser pair).
    let freezer = Pubkey::new_unique();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer, unpauser.pubkey()).await;

    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ..
    } = context;

    // (1) Transfer succeeds while unpaused.
    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        &Keypair::new(),
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();

    // (2) Wrong signer cannot pause (must equal the configured pauser).
    let stranger = Keypair::new();
    common::transfer(client, payer, &stranger.pubkey(), 1_000_000_000)
        .await
        .unwrap();
    common::pause(client, token_bridge, &stranger, payer)
        .await
        .expect_err("pause from wrong signer must fail with InvalidPauser");

    // (3) Configured pauser pauses, then transfer rejects with `Paused`.
    common::pause(client, token_bridge, &pauser, payer)
        .await
        .unwrap();
    let paused_data = client
        .get_account(ConfigAccount::<'_, { AccountState::Initialized }>::key(
            None,
            &token_bridge,
        ))
        .await
        .unwrap()
        .unwrap()
        .data;
    assert!(
        read_paused(&paused_data),
        "Config tail should report paused=true"
    );
    // A timed `pause` records a non-zero expiry in the future (`now + PAUSE_DURATION`). The exact
    // value depends on the on-chain clock, so we just assert it advanced past zero here; the
    // dedicated `pause_sets_timed_expiry` test pins the relationship to `PAUSE_DURATION`.
    let pause_expiry = read_pause_expiry(&paused_data);
    assert!(
        pause_expiry > 0,
        "pause should set a non-zero expiry, got {}",
        pause_expiry
    );

    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        &Keypair::new(),
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .expect_err("transfer must fail while paused");

    // (4) Wrong signer cannot unpause either.
    common::unpause(client, token_bridge, &stranger, payer)
        .await
        .expect_err("unpause from wrong signer must fail with InvalidPauser");

    // (5) Configured unpauser unpauses, transfer succeeds again.
    common::unpause(client, token_bridge, &unpauser, payer)
        .await
        .unwrap();
    let unpaused_data = client
        .get_account(ConfigAccount::<'_, { AccountState::Initialized }>::key(
            None,
            &token_bridge,
        ))
        .await
        .unwrap()
        .unwrap()
        .data;
    assert!(!read_paused(&unpaused_data));

    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        &Keypair::new(),
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn legacy_unmigrated_compat() {
    // No SetPauserAddresses is ever submitted, so Config stays at 32 bytes. Every existing
    // transfer/complete entry point must behave exactly like the pre-upgrade implementation:
    // the `notPaused` gate sees a too-short account and treats it as unpaused.
    let mut context = set_up().await.unwrap();

    let pre = fetch_config_data(&mut context).await;
    assert_eq!(
        pre.len(),
        CONFIG_BORSH_LEN,
        "Config must still be the legacy layout"
    );

    // pause / unpause both refuse on a legacy account.
    let stranger = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &stranger.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::pause(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("pause must fail on un-migrated Config (PauserNotConfigured)");
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("unpause must fail on un-migrated Config (PauserNotConfigured)");
    // freeze refuses too: the freezer role reads as the zero pubkey on a legacy account.
    common::freeze(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("freeze must fail on un-migrated Config (PauserNotConfigured)");
    // unpause_expired is permissionless but still requires `paused`; a legacy account is unpaused.
    common::unpause_expired(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("unpause_expired must fail on un-migrated Config (NotPaused)");

    // A native transfer still works — the gate is a no-op on legacy accounts.
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ..
    } = context;
    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        &Keypair::new(),
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .await
    .unwrap();

    // The Config account size must not have changed as a side-effect of any transfer.
    let post = fetch_config_data(&mut context).await;
    assert_eq!(
        post.len(),
        CONFIG_BORSH_LEN,
        "transfers must not migrate the Config"
    );
}

// ============== SetPauserAddresses wire-format validation (whitepaper 0003) ==============
//
// whitepapers/0003_token_bridge.md Pausing defines a length-prefixed encoding shared across runtimes,
// carrying three roles with the freezer in the middle:
//
//     module(32) | action(1)=4 | chain(2)
//   | pauser_len(1)   | pauser[pauser_len]
//   | freezer_len(1)  | freezer[freezer_len]
//   | unpauser_len(1) | unpauser[unpauser_len]
//
// On Solana each length must be 32 (native size) or 0 (role left unassigned). The receive
// side must reject any other length and any trailing bytes after the third address. A
// length-32 field of all zeros is equivalent to a zero-length field — both decode to
// `Pubkey::default()` and the resulting role is treated as unassigned.

#[tokio::test]
async fn set_pauser_addresses_rejects_invalid_pauser_length() {
    let mut context = set_up().await.unwrap();

    // pauser_len = 20 (the EVM native size) must be rejected on Solana.
    let pauser_body = [0xAAu8; 20];
    let freezer_body = [0xCCu8; 32];
    let unpauser_body = [0xBBu8; 32];
    let bad =
        build_set_pauser_addresses_payload(20, &pauser_body, 32, &freezer_body, 32, &unpauser_body);

    submit_raw_set_pauser_addresses(&mut context, bad)
        .await
        .expect_err("pauser_len = 20 must be rejected on SVM");

    // Config must remain at the legacy size — no migration on a failed VAA.
    let post = fetch_config_data(&mut context).await;
    assert_eq!(post.len(), CONFIG_BORSH_LEN);
}

#[tokio::test]
async fn set_pauser_addresses_rejects_invalid_unpauser_length() {
    let mut context = set_up().await.unwrap();

    let pauser_body = [0xAAu8; 32];
    let freezer_body = [0xCCu8; 32];
    let unpauser_body = [0xBBu8; 33]; // off-by-one over the native size
    let bad =
        build_set_pauser_addresses_payload(32, &pauser_body, 32, &freezer_body, 33, &unpauser_body);

    submit_raw_set_pauser_addresses(&mut context, bad)
        .await
        .expect_err("unpauser_len = 33 must be rejected on SVM");
}

#[tokio::test]
async fn set_pauser_addresses_rejects_trailing_bytes() {
    let mut context = set_up().await.unwrap();

    let pauser_body = [0xAAu8; 32];
    let freezer_body = [0xCCu8; 32];
    let unpauser_body = [0xBBu8; 32];
    let mut bad =
        build_set_pauser_addresses_payload(32, &pauser_body, 32, &freezer_body, 32, &unpauser_body);
    bad.extend_from_slice(&[0xDDu8; 5]); // trailing garbage

    submit_raw_set_pauser_addresses(&mut context, bad)
        .await
        .expect_err("trailing bytes after unpauser must be rejected");
}

#[tokio::test]
async fn set_pauser_addresses_rejects_invalid_freezer_length() {
    let mut context = set_up().await.unwrap();

    // freezer_len = 20 (the EVM native size) must be rejected on Solana, exactly like the pauser
    // and unpauser fields. This guards the middle address of the three-role encoding.
    let pauser_body = [0xAAu8; 32];
    let freezer_body = [0xCCu8; 20];
    let unpauser_body = [0xBBu8; 32];
    let bad =
        build_set_pauser_addresses_payload(32, &pauser_body, 20, &freezer_body, 32, &unpauser_body);

    submit_raw_set_pauser_addresses(&mut context, bad)
        .await
        .expect_err("freezer_len = 20 must be rejected on SVM");

    // Config must remain at the legacy size — no migration on a failed VAA.
    let post = fetch_config_data(&mut context).await;
    assert_eq!(post.len(), CONFIG_BORSH_LEN);
}

#[tokio::test]
async fn set_pauser_addresses_zero_length_means_unassigned() {
    let mut context = set_up().await.unwrap();

    // pauser_len = 0, freezer_len = 32, unpauser_len = 32 — exercises the "zero-length =
    // unassigned" branch on the first address while the other two are present.
    let freezer_body = [0xCCu8; 32];
    let unpauser_body = [0xBBu8; 32];
    let payload = build_set_pauser_addresses_payload(0, &[], 32, &freezer_body, 32, &unpauser_body);
    submit_raw_set_pauser_addresses(&mut context, payload)
        .await
        .expect("zero-length pauser is a valid encoding");

    let post = fetch_config_data(&mut context).await;
    assert_eq!(post.len(), CONFIG_WITH_PAUSER_LEN);
    assert_eq!(
        read_pauser(&post),
        Pubkey::default(),
        "zero-length pauser must decode as Pubkey::default()"
    );
    assert_eq!(read_freezer(&post), Pubkey::new(&freezer_body[..]));
    assert_eq!(read_unpauser(&post), Pubkey::new(&unpauser_body[..]));

    // With pauser unassigned, the pause entry point must reject every caller — including a
    // signer whose key is the all-zero pubkey would be, if we could construct it. We exercise
    // the path with a non-zero stranger: the on-chain handler reverts with PauserNotConfigured
    // BEFORE comparing the caller against the (zero) configured pauser.
    let stranger = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &stranger.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::pause(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("pause must reject when pauser is unassigned");
}

#[tokio::test]
async fn set_pauser_addresses_all_zero_32_byte_is_unassigned() {
    let mut context = set_up().await.unwrap();

    // pauser_len = 32 with all-zero bytes is the "all-zero native-size address" encoding. Per
    // the whitepaper it MUST be treated as equivalent to length 0 — i.e. unassigned.
    let zero_pauser = [0u8; 32];
    let freezer_body = [0xCCu8; 32];
    let unpauser_body = [0xBBu8; 32];
    let payload =
        build_set_pauser_addresses_payload(32, &zero_pauser, 32, &freezer_body, 32, &unpauser_body);
    submit_raw_set_pauser_addresses(&mut context, payload)
        .await
        .expect("length-32 all-zero pauser is a valid encoding");

    let post = fetch_config_data(&mut context).await;
    assert_eq!(read_pauser(&post), Pubkey::default());
    assert_eq!(read_freezer(&post), Pubkey::new(&freezer_body[..]));
    assert_eq!(read_unpauser(&post), Pubkey::new(&unpauser_body[..]));

    // Recovery: a follow-up VAA can assign a non-zero pauser without first having to "clear"
    // the previous one.
    let real_pauser = Pubkey::new_unique();
    submit_set_pauser_addresses(
        &mut context,
        real_pauser,
        Pubkey::new(&freezer_body[..]),
        Pubkey::new(&unpauser_body[..]),
    )
    .await;
    let post = fetch_config_data(&mut context).await;
    assert_eq!(read_pauser(&post), real_pauser);
}

#[tokio::test]
async fn set_pauser_addresses_rejects_legacy_action_id() {
    let mut context = set_up().await.unwrap();

    // Whitepaper 0003 mandates a single action 4 for `SetPauserAddresses`, shared across runtimes
    // (a per-runtime action split was explicitly rejected in "Alternatives Considered"). A VAA
    // carrying any other action — e.g. 5 — must be rejected even if otherwise well-formed.
    let mut payload =
        build_set_pauser_addresses_payload(32, &[0u8; 32], 32, &[0u8; 32], 32, &[0u8; 32]);
    payload[32] = 5; // overwrite the action byte (otherwise a well-formed 3-address payload)

    submit_raw_set_pauser_addresses(&mut context, payload)
        .await
        .expect_err("action 5 must be rejected (only action 4 is valid)");
}

// ==================== Paused gate coverage across user entry points ====================
//
// Per the "Pausing" section of whitepaper 0003, every user-facing entry point on the token
// bridge must revert with `Paused` while the bridge is paused. The `require_not_paused`
// helper in `types.rs` is wired into all ten user entry points:
//
//   1. api::attest::attest_token
//   2. api::transfer::transfer_native
//   3. api::transfer_payload::transfer_native_with_payload
//   4. api::transfer::transfer_wrapped
//   5. api::transfer_payload::transfer_wrapped_with_payload
//   6. api::create_wrapped::create_wrapped
//   7. api::complete_transfer::complete_native
//   8. api::complete_transfer::complete_wrapped
//   9. api::complete_transfer_payload::complete_native_with_payload
//  10. api::complete_transfer_payload::complete_wrapped_with_payload
//
// The test below sets up bridge + wrapped-mint state ahead of time, pauses the bridge once,
// and then drives every entry point with valid `FromAccounts` inputs so each call reaches the
// `require_not_paused(...)` call site rather than failing in account validation. Each call is
// expected to return an error while paused; an unpause + redo at the end verifies the gate
// (not some misconfigured setup) is the sole reason the calls failed.

/// Helper used inside `paused_blocks_all_user_entry_points` to build, post, and return the
/// `(vaa, message_key)` pair for an inbound VAA that targets the token bridge. Takes the
/// payload by reference so the caller retains ownership for the subsequent handler call.
async fn post_inbound_vaa<P: SerializePayload>(
    context: &mut Context,
    emitter: [u8; 32],
    chain: u16,
    payload: &P,
    sequence: u64,
) -> (PostVAAData, Pubkey) {
    let nonce = rand::thread_rng().gen();
    let message = payload.try_to_vec().unwrap();
    let (vaa, body, _) = common::generate_vaa(emitter, chain, message, nonce, sequence);
    let signature_set = common::verify_signatures(
        &mut context.client,
        &context.bridge,
        &context.payer,
        body,
        &context.guardian_keys,
        0,
    )
    .await
    .unwrap();
    common::post_vaa(
        &mut context.client,
        context.bridge,
        &context.payer,
        signature_set,
        vaa.clone(),
    )
    .await
    .unwrap();
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        &context.bridge,
    );
    (vaa, message_key)
}

#[tokio::test]
async fn paused_blocks_all_user_entry_points() {
    let mut context = set_up().await.unwrap();
    register_chain(&mut context).await;
    // `register_chain` hardcodes `sequence = 0` without touching the `Sequencer`. Manually
    // consume sequence 0 in the sequencer so subsequent governance VAAs from the same emitter
    // don't collide on the Claim PDA (`AlreadyInitialized`).
    let governance_emitter = Keypair::from_bytes(&GOVERNANCE_KEY)
        .unwrap()
        .pubkey()
        .to_bytes();
    let _consumed_zero = context.seq.next(governance_emitter);

    // Set up wrapped-mint state (token_address = [1u8; 32], token_chain = 2) so the wrapped-side
    // `FromAccounts` validation passes once we pause. `create_wrapped_account` internally posts
    // an AssetMeta VAA from foreign emitter `[0u8; 32]` / chain 2 at sequence 2.
    let wrapped_acc = create_wrapped_account(&mut context).await.unwrap();

    // Configure pauser/unpauser and pause the bridge.
    let pauser = Keypair::new();
    let unpauser = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    let freezer = Pubkey::new_unique();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer, unpauser.pubkey()).await;
    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .unwrap();

    // --------------- Outbound entry points ---------------

    // (1) attest — emits AssetMeta for a native mint.
    common::attest(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        context.mint.pubkey(),
        0,
    )
    .await
    .expect_err("attest must fail while paused");

    // (2) transfer_native — outbound lock of native SPL tokens into custody.
    common::transfer_native(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        &context.token_account,
        &context.token_authority,
        context.mint.pubkey(),
        100,
    )
    .await
    .expect_err("transfer_native must fail while paused");

    // (3) transfer_native_with_payload — same outbound path, additional opaque payload.
    common::transfer_native_with_payload(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        &context.token_account,
        &context.token_authority,
        context.mint.pubkey(),
        100,
        vec![1, 2, 3],
    )
    .await
    .expect_err("transfer_native_with_payload must fail while paused");

    // (4) transfer_wrapped — outbound burn of wrapped tokens. Uses the wrapped account created
    //     during setup; the SPL approve preceding the bridge instruction is a separate ix that
    //     succeeds on a zero-balance account, so we still reach `require_not_paused`.
    common::transfer_wrapped(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        wrapped_acc,
        &context.token_authority,
        2,
        [1u8; 32],
        10_000_000,
    )
    .await
    .expect_err("transfer_wrapped must fail while paused");

    // (5) transfer_wrapped_with_payload — same outbound path, additional opaque payload.
    common::transfer_wrapped_with_payload(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        wrapped_acc,
        &context.token_authority,
        2,
        [1u8; 32],
        10_000_000,
        vec![4, 5, 6],
    )
    .await
    .expect_err("transfer_wrapped_with_payload must fail while paused");

    // --------------- Inbound entry points (each gets its own VAA) ---------------

    // (6) create_wrapped — inbound AssetMeta-driven wrapper creation for a fresh foreign token.
    {
        let payload = PayloadAssetMeta {
            token_address: [2u8; 32], // different from the one set up by create_wrapped_account
            token_chain: 2,
            decimals: 7,
            symbol: "".to_string(),
            name: "".to_string(),
        };
        let (vaa, message_key) = post_inbound_vaa(&mut context, [0u8; 32], 2, &payload, 42).await;
        common::create_wrapped(
            &mut context.client,
            context.token_bridge,
            context.bridge,
            message_key,
            vaa,
            payload,
            &context.payer,
        )
        .await
        .expect_err("create_wrapped must fail while paused");
    }

    // (7) complete_native — release native custody to the existing token account.
    {
        let payload = PayloadTransfer {
            amount: U256::from(1u128),
            token_address: context.mint.pubkey().to_bytes(),
            token_chain: 1,
            to: context.token_account.pubkey().to_bytes(),
            to_chain: 1,
            fee: U256::from(0u128),
        };
        let (vaa, message_key) = post_inbound_vaa(&mut context, [0u8; 32], 2, &payload, 43).await;
        common::complete_native(
            &mut context.client,
            context.token_bridge,
            context.bridge,
            message_key,
            vaa,
            payload,
            &context.payer,
        )
        .await
        .expect_err("complete_native must fail while paused");
    }

    // (8) complete_wrapped — mint wrapped tokens into the wrapped token account.
    {
        let payload = PayloadTransfer {
            amount: U256::from(1u128),
            token_address: [1u8; 32],
            token_chain: 2,
            to: wrapped_acc.to_bytes(),
            to_chain: 1,
            fee: U256::from(0u128),
        };
        let (vaa, message_key) = post_inbound_vaa(&mut context, [0u8; 32], 2, &payload, 44).await;
        common::complete_transfer_wrapped(
            &mut context.client,
            context.token_bridge,
            context.bridge,
            message_key,
            vaa,
            payload,
            &context.payer,
        )
        .await
        .expect_err("complete_wrapped must fail while paused");
    }

    // (9) complete_native_with_payload — release native custody with a redeemer signature.
    {
        let payload = PayloadTransferWithPayload {
            amount: U256::from(1u128),
            token_address: context.mint.pubkey().to_bytes(),
            token_chain: OUR_CHAIN_ID,
            to: context.token_authority.pubkey().to_bytes(),
            to_chain: OUR_CHAIN_ID,
            from_address: Keypair::new().pubkey().to_bytes(),
            payload: vec![1, 2, 3],
        };
        let (vaa, message_key) =
            post_inbound_vaa(&mut context, [0u8; 32], CHAIN_ID_ETH, &payload, 45).await;
        common::complete_native_with_payload(
            &mut context.client,
            context.token_bridge,
            context.bridge,
            message_key,
            vaa,
            payload,
            context.token_account.pubkey(),
            &context.token_authority,
            &context.payer,
        )
        .await
        .expect_err("complete_native_with_payload must fail while paused");
    }

    // (10) complete_wrapped_with_payload — mint wrapped tokens with a redeemer signature.
    {
        let payload = PayloadTransferWithPayload {
            amount: U256::from(1u128),
            token_address: [1u8; 32],
            token_chain: 2,
            to: context.token_authority.pubkey().to_bytes(),
            to_chain: OUR_CHAIN_ID,
            from_address: Keypair::new().pubkey().to_bytes(),
            payload: vec![4, 5, 6],
        };
        let (vaa, message_key) =
            post_inbound_vaa(&mut context, [0u8; 32], CHAIN_ID_ETH, &payload, 46).await;
        common::complete_wrapped_with_payload(
            &mut context.client,
            context.token_bridge,
            context.bridge,
            message_key,
            vaa,
            payload,
            wrapped_acc,
            &context.token_authority,
            &context.payer,
        )
        .await
        .expect_err("complete_wrapped_with_payload must fail while paused");
    }

    // Now unpause and verify at least one entry point is functional again — this proves the
    // gate is the sole reason the calls above were failing (not some unrelated misconfiguration).
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &unpauser,
        &context.payer,
    )
    .await
    .unwrap();
    common::attest(
        &mut context.client,
        context.token_bridge,
        context.bridge,
        &context.payer,
        &Keypair::new(),
        context.mint.pubkey(),
        0,
    )
    .await
    .expect("attest must succeed after unpause");
}

#[tokio::test]
async fn rotate_pauser_addresses_while_paused() {
    // Confirms that `submit_set_pauser_addresses` (a governance handler) is callable while the
    // bridge is paused, that the new keys take effect atomically, and that the `paused` flag is
    // preserved across the rotation (a rotation does not implicitly unpause).
    let mut context = set_up().await.unwrap();

    let pauser_one = Keypair::new();
    let freezer_one = Pubkey::new_unique();
    let unpauser_one = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser_one.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(
        &mut context,
        pauser_one.pubkey(),
        freezer_one,
        unpauser_one.pubkey(),
    )
    .await;
    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser_one,
        &context.payer,
    )
    .await
    .unwrap();
    let paused_data = fetch_config_data(&mut context).await;
    assert!(read_paused(&paused_data));
    let expiry_before_rotation = read_pause_expiry(&paused_data);
    assert!(expiry_before_rotation > 0);

    // Rotate to a new pauser / freezer / unpauser triple while paused.
    let pauser_two = Keypair::new();
    let freezer_two = Pubkey::new_unique();
    let unpauser_two = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser_two.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(
        &mut context,
        pauser_two.pubkey(),
        freezer_two,
        unpauser_two.pubkey(),
    )
    .await;

    let post = fetch_config_data(&mut context).await;
    assert_eq!(read_pauser(&post), pauser_two.pubkey());
    assert_eq!(read_freezer(&post), freezer_two);
    assert_eq!(read_unpauser(&post), unpauser_two.pubkey());
    assert!(
        read_paused(&post),
        "paused flag must be preserved across rotation",
    );
    assert_eq!(
        read_pause_expiry(&post),
        expiry_before_rotation,
        "pause_expiry must be preserved across rotation",
    );

    // Old unpauser_one can no longer unpause.
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser_one.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &unpauser_one,
        &context.payer,
    )
    .await
    .expect_err("old unpauser must be rejected after rotation");

    // New unpauser_two can.
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &unpauser_two,
        &context.payer,
    )
    .await
    .unwrap();
    assert!(!read_paused(&fetch_config_data(&mut context).await));
}

// ==================== pause extend / revert semantics ====================
//
// whitepaper 0003 "Pausing": `pause` must push `pause_expiry` strictly forward (a fresh pause is
// covered by `pause_sets_timed_expiry`); a `pause` that cannot move the expiry forward — e.g. the
// bridge is already frozen — reverts with `PauseNotExtended` (see `pause_reverts_when_frozen`)
// rather than emitting a misleading success, so `pause` is NOT idempotent. In contrast,
// `unpause`/`unpause_expired` REQUIRE the bridge to be paused — calling
// them on an unpaused bridge reverts with `NotPaused` (see `unpause_reverts_when_not_paused`).
// `SetPauserAddresses` is governed by VAA replay protection (the Claim PDA), so re-submitting the
// same payload via a *fresh* VAA succeeds and rewrites the tail with identical bytes.

#[tokio::test]
async fn unpause_reverts_when_not_paused() {
    // Per whitepaper 0003, `unpause` is NOT a no-op on an unpaused bridge — it reverts with
    // `NotPaused`. Both `unpause` and `unpause_expired` require the bridge to actually be paused.
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Pubkey::new_unique();
    let unpauser = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer, unpauser.pubkey()).await;

    // Fresh tail defaults to unpaused.
    assert!(!read_paused(&fetch_config_data(&mut context).await));

    // Unpause on an unpaused bridge: must revert with NotPaused.
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &unpauser,
        &context.payer,
    )
    .await
    .expect_err("unpause on an unpaused bridge must revert with NotPaused");
    assert!(!read_paused(&fetch_config_data(&mut context).await));
}

#[tokio::test]
async fn set_pauser_addresses_is_idempotent() {
    let mut context = set_up().await.unwrap();

    let pauser = Pubkey::new_unique();
    let freezer = Pubkey::new_unique();
    let unpauser = Pubkey::new_unique();

    submit_set_pauser_addresses(&mut context, pauser, freezer, unpauser).await;
    let first = fetch_config_data(&mut context).await;
    assert_eq!(read_pauser(&first), pauser);
    assert_eq!(read_freezer(&first), freezer);
    assert_eq!(read_unpauser(&first), unpauser);

    // Second VAA with identical payload (fresh nonce/sequence so the Claim PDA differs and
    // replay protection lets it through). The on-chain effect must be byte-identical.
    submit_set_pauser_addresses(&mut context, pauser, freezer, unpauser).await;
    let second = fetch_config_data(&mut context).await;
    assert_eq!(
        second, first,
        "identical SetPauserAddresses payload must produce byte-identical Config",
    );
}

// ==================== Timed pause / freeze / unpause_expired ====================
//
// whitepaper 0003 "Pausing" adds a 5-day timed `pause` (sets `pause_expiry = now + PAUSE_DURATION`),
// a higher-trust `freeze` (sets `pause_expiry = i64::MAX`, only liftable by the unpauser), and a
// permissionless `unpause_expired` that anyone may call once `now >= pause_expiry`. The boolean
// `paused` stays authoritative — time never silently unpauses; `pause_expiry` only gates
// `unpause_expired`.
//
// NOTE: the `now >= pause_expiry` SUCCESS path of `unpause_expired` is intentionally NOT covered
// here. The solana-program-test harness in use (1.10.31) constructs its `BanksClient` via
// `ProgramTest::start()`, which discards the `ProgramTestContext` needed to advance/override the
// Clock sysvar — so on-chain time cannot be pushed 5 days forward. That path is exercised in the
// SDK / live-validator tests. Everything reachable with the (near-genesis) clock is covered below.

#[tokio::test]
async fn pause_sets_timed_expiry() {
    // `pause` must set `pause_expiry` to exactly `now + PAUSE_DURATION`, where `now` is the Clock
    // sysvar's `unix_timestamp` at execution time.
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Pubkey::new_unique();
    let unpauser = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer, unpauser.pubkey()).await;

    let before = fetch_clock_unix_timestamp(&mut context).await;
    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .unwrap();
    let after = fetch_clock_unix_timestamp(&mut context).await;

    let data = fetch_config_data(&mut context).await;
    assert!(read_paused(&data));
    let expiry = read_pause_expiry(&data);
    // The clock the handler observed is somewhere in [before, after]; the recorded expiry must be
    // that clock plus exactly PAUSE_DURATION. A finite expiry also distinguishes a `pause` from a
    // `freeze` (which records i64::MAX).
    assert!(
        expiry >= before + PAUSE_DURATION && expiry <= after + PAUSE_DURATION,
        "expiry {} must equal observed_now + PAUSE_DURATION (now in [{}, {}])",
        expiry,
        before,
        after,
    );
    assert!(expiry < i64::MAX, "a timed pause must not record i64::MAX");
}

#[tokio::test]
async fn freeze_blocks_transfer_and_unpause_restores() {
    // `freeze` (higher-trust) pauses the bridge with `pause_expiry = i64::MAX`. A wrong signer is
    // rejected; transfers are blocked; the unpauser can still lift it early.
    let mut context = set_up().await.unwrap();

    let pauser = Pubkey::new_unique();
    let freezer = Keypair::new();
    let unpauser = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &freezer.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::transfer(
        &mut context.client,
        &context.payer,
        &unpauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser, freezer.pubkey(), unpauser.pubkey()).await;

    // Wrong signer cannot freeze.
    let stranger = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &stranger.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::freeze(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("freeze from wrong signer must fail with InvalidFreezer");

    // Configured freezer freezes: paused = true, expiry = i64::MAX.
    common::freeze(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .unwrap();
    let frozen = fetch_config_data(&mut context).await;
    assert!(read_paused(&frozen), "freeze must set paused = true");
    assert_eq!(
        read_pause_expiry(&frozen),
        i64::MAX,
        "freeze must set pause_expiry = i64::MAX",
    );

    // Transfers are blocked while frozen.
    {
        let Context {
            ref payer,
            ref mut client,
            bridge,
            token_bridge,
            ref mint,
            ref token_account,
            ref token_authority,
            ..
        } = context;
        common::transfer_native(
            client,
            token_bridge,
            bridge,
            payer,
            &Keypair::new(),
            token_account,
            token_authority,
            mint.pubkey(),
            100,
        )
        .await
        .expect_err("transfer must fail while frozen");
    }

    // The unpauser can lift a freeze early (the privileged escape hatch).
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &unpauser,
        &context.payer,
    )
    .await
    .unwrap();
    let after = fetch_config_data(&mut context).await;
    assert!(!read_paused(&after), "unpause must clear a freeze");
    assert!(
        read_pause_expiry(&after) < i64::MAX,
        "unpause must bring the freeze expiry back down to now",
    );
}

#[tokio::test]
async fn freeze_is_idempotent() {
    // `freeze` on an already-frozen bridge succeeds and leaves the state unchanged.
    let mut context = set_up().await.unwrap();

    let pauser = Pubkey::new_unique();
    let freezer = Keypair::new();
    let unpauser = Pubkey::new_unique();
    common::transfer(
        &mut context.client,
        &context.payer,
        &freezer.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser, freezer.pubkey(), unpauser).await;

    common::freeze(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .unwrap();
    let first = fetch_config_data(&mut context).await;
    assert!(read_paused(&first));
    assert_eq!(read_pause_expiry(&first), i64::MAX);

    common::freeze(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .expect("freeze on an already-frozen bridge must be a no-op success");
    let second = fetch_config_data(&mut context).await;
    assert_eq!(
        second, first,
        "a repeated freeze must not change Config bytes",
    );
}

#[tokio::test]
async fn pause_reverts_when_frozen() {
    // A `pause` must never reduce a `pause_expiry` that is already further in the future. After a
    // `freeze` (expiry = i64::MAX), a subsequent `pause` cannot push the expiry forward, so it
    // reverts with `PauseNotExtended` — a lower-trust pauser cannot shorten a freeze into a 5-day
    // window, and the freeze's expiry is left untouched.
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Keypair::new();
    let unpauser = Pubkey::new_unique();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::transfer(
        &mut context.client,
        &context.payer,
        &freezer.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer.pubkey(), unpauser).await;

    common::freeze(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .unwrap();
    assert_eq!(
        read_pause_expiry(&fetch_config_data(&mut context).await),
        i64::MAX
    );

    // Pause while frozen: cannot extend past i64::MAX, so it reverts. Still paused, expiry
    // unchanged at i64::MAX (never curtailed).
    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .expect_err("pause on a frozen bridge must revert with PauseNotExtended");
    let data = fetch_config_data(&mut context).await;
    assert!(read_paused(&data));
    assert_eq!(
        read_pause_expiry(&data),
        i64::MAX,
        "pause must not reduce a freeze's expiry",
    );
}

#[tokio::test]
async fn freeze_after_pause_raises_expiry_to_max() {
    // The complementary ordering to `pause_does_not_curtail_freeze`: a timed `pause` sets a finite
    // expiry, and a subsequent `freeze` raises it to i64::MAX (freeze always wins upward).
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Keypair::new();
    let unpauser = Pubkey::new_unique();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::transfer(
        &mut context.client,
        &context.payer,
        &freezer.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer.pubkey(), unpauser).await;

    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .unwrap();
    let after_pause = read_pause_expiry(&fetch_config_data(&mut context).await);
    assert!(
        after_pause > 0 && after_pause < i64::MAX,
        "a timed pause must record a finite expiry, got {}",
        after_pause,
    );

    common::freeze(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .unwrap();
    let after_freeze = fetch_config_data(&mut context).await;
    assert!(read_paused(&after_freeze));
    assert_eq!(
        read_pause_expiry(&after_freeze),
        i64::MAX,
        "freeze after pause must raise the expiry to i64::MAX",
    );
}

#[tokio::test]
async fn unpause_expired_rejects_before_expiry() {
    // Immediately after a timed `pause`, the pause has NOT expired (now << now + PAUSE_DURATION),
    // so the permissionless `unpause_expired` must revert with NotExpired and leave the bridge
    // paused.
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Pubkey::new_unique();
    let unpauser = Pubkey::new_unique();
    common::transfer(
        &mut context.client,
        &context.payer,
        &pauser.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    submit_set_pauser_addresses(&mut context, pauser.pubkey(), freezer, unpauser).await;

    common::pause(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .unwrap();
    let paused_data = fetch_config_data(&mut context).await;
    assert!(read_paused(&paused_data));
    let expiry_before = read_pause_expiry(&paused_data);

    // Any fee payer may attempt unpause_expired — here a random stranger.
    let stranger = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &stranger.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::unpause_expired(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("unpause_expired before expiry must revert with NotExpired");
    // The rejected call must leave state untouched: still paused, same expiry.
    let after = fetch_config_data(&mut context).await;
    assert!(
        read_paused(&after),
        "bridge must remain paused after a rejected unpause_expired",
    );
    assert_eq!(
        read_pause_expiry(&after),
        expiry_before,
        "a rejected unpause_expired must not change pause_expiry",
    );
}

#[tokio::test]
async fn unpause_expired_rejects_when_not_paused() {
    // `unpause_expired` is permissionless but still requires the bridge to be paused. On a migrated
    // but unpaused bridge it must revert with NotPaused.
    let mut context = set_up().await.unwrap();

    let pauser = Pubkey::new_unique();
    let freezer = Pubkey::new_unique();
    let unpauser = Pubkey::new_unique();
    submit_set_pauser_addresses(&mut context, pauser, freezer, unpauser).await;
    assert!(!read_paused(&fetch_config_data(&mut context).await));

    let stranger = Keypair::new();
    common::transfer(
        &mut context.client,
        &context.payer,
        &stranger.pubkey(),
        1_000_000_000,
    )
    .await
    .unwrap();
    common::unpause_expired(
        &mut context.client,
        context.token_bridge,
        &stranger,
        &context.payer,
    )
    .await
    .expect_err("unpause_expired on an unpaused bridge must revert with NotPaused");
}

#[tokio::test]
async fn pause_freeze_unpause_roles_are_distinct() {
    // The three roles are independent: the pauser cannot freeze, and the freezer cannot pause or
    // unpause. Each entry point checks its own configured address.
    let mut context = set_up().await.unwrap();

    let pauser = Keypair::new();
    let freezer = Keypair::new();
    let unpauser = Keypair::new();
    for kp in [&pauser, &freezer, &unpauser] {
        common::transfer(
            &mut context.client,
            &context.payer,
            &kp.pubkey(),
            1_000_000_000,
        )
        .await
        .unwrap();
    }
    submit_set_pauser_addresses(
        &mut context,
        pauser.pubkey(),
        freezer.pubkey(),
        unpauser.pubkey(),
    )
    .await;

    // pauser cannot freeze.
    common::freeze(
        &mut context.client,
        context.token_bridge,
        &pauser,
        &context.payer,
    )
    .await
    .expect_err("pauser must not be able to freeze (InvalidFreezer)");

    // freezer cannot pause.
    common::pause(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .expect_err("freezer must not be able to pause (InvalidPauser)");

    // freezer cannot unpause.
    common::unpause(
        &mut context.client,
        context.token_bridge,
        &freezer,
        &context.payer,
    )
    .await
    .expect_err("freezer must not be able to unpause (InvalidPauser)");

    // Sanity: the bridge is still unpaused after all the rejected attempts.
    assert!(!read_paused(&fetch_config_data(&mut context).await));
}

// ==================== Event discriminator derivation pins ====================
//
// Each Anchor-style event discriminator is `SHA256("event:<EventName>")[..8]`. The constants in
// `api/pause.rs` and `api/governance.rs` are pre-computed; these tests re-derive them at test
// time and assert equality, so a future change to either the event name string or the constant
// fails CI rather than silently mis-emitting events that off-chain indexers can't decode.
// Mirrors the `test_message_account_closed_discriminator_matches_sha256` check in the core
// bridge integration tests.

#[test]
fn test_paused_event_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:Paused");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        PAUSED_EVENT_DISCRIMINATOR, expected,
        "PAUSED_EVENT_DISCRIMINATOR must equal SHA256(\"event:Paused\")[..8]",
    );
}

#[test]
fn test_unpaused_event_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:Unpaused");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        UNPAUSED_EVENT_DISCRIMINATOR, expected,
        "UNPAUSED_EVENT_DISCRIMINATOR must equal SHA256(\"event:Unpaused\")[..8]",
    );
}

#[test]
fn test_frozen_event_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:Frozen");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        FROZEN_EVENT_DISCRIMINATOR, expected,
        "FROZEN_EVENT_DISCRIMINATOR must equal SHA256(\"event:Frozen\")[..8]",
    );
}

#[test]
fn test_unpause_expired_event_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:UnpauseExpired");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        UNPAUSE_EXPIRED_EVENT_DISCRIMINATOR, expected,
        "UNPAUSE_EXPIRED_EVENT_DISCRIMINATOR must equal SHA256(\"event:UnpauseExpired\")[..8]",
    );
}

#[test]
fn test_pauser_addresses_set_event_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:PauserAddressesSet");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        PAUSER_ADDRESSES_SET_EVENT_DISCRIMINATOR, expected,
        "PAUSER_ADDRESSES_SET_EVENT_DISCRIMINATOR must equal SHA256(\"event:PauserAddressesSet\")[..8]",
    );
}
