#![allow(warnings)]

use borsh::BorshSerialize;
use bridge::{
    accounts::{
        Bridge,
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        PostedVAA,
        PostedVAADerivationData,
        SignatureSet,
    },
    instruction,
    types::{
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        GovernancePayloadTransferFees,
    },
    BridgeConfig,
    BridgeData,
    GuardianSetData,
    Initialize,
    MessageData,
    PostVAA,
    PostVAAData,
    PostedVAAData,
    SequenceTracker,
    SerializePayload,
    Signature,
    SignatureSet as SignatureSetData,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use hex_literal::hex;
use libsecp256k1::{
    Message as Secp256k1Message,
    PublicKey,
    SecretKey,
};
use primitive_types::U256;
use rand::Rng;
use sha3::Digest;
use solana_program::{
    borsh::try_from_slice_unchecked,
    hash,
    instruction::{
        AccountMeta,
        Instruction,
    },
    program_pack::Pack,
    pubkey::Pubkey,
    system_instruction::{
        self,
        create_account,
    },
    system_program,
    sysvar,
};
use solana_program_test::{
    tokio,
    BanksClient,
};
use solana_sdk::{
    commitment_config::CommitmentLevel,
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
    transport::TransportError,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use spl_token::state::Mint;
use std::{
    collections::HashMap,
    convert::TryInto,
    io::{
        Cursor,
        Write,
    },
    str::FromStr,
    time::{
        Duration,
        SystemTime,
        UNIX_EPOCH,
    },
};
use token_bridge::{
    accounts::{
        ConfigAccount,
        EmitterAccount,
        WrappedDerivationData,
        WrappedMint,
    },
    messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    },
    types::{
        Address,
        Config,
    },
};

mod common;

const GOVERNANCE_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44, 196,
    163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201, 174, 108,
    197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170, 141, 231, 147,
    88, 97, 199,
];

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

    let (metadata_key, metadata_bump_seed) = Pubkey::find_program_address(
        metadata_seeds,
        &Pubkey::from_str("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s").unwrap(),
    );

    // Token Bridge Meta
    use token_bridge::accounts::WrappedTokenMeta;
    let metadata_account = WrappedTokenMeta::<'_, { AccountState::Uninitialized }>::key(
        &token_bridge::accounts::WrappedMetaDerivationData {
            mint_key: mint_pubkey.clone(),
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
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ref token_account,
        ref token_authority,
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
    let mut msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(&msg_derivation_data, bridge);

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
        &token_bridge,
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
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ref metadata_account,
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

    let emitter_key = EmitterAccount::key(None, &token_bridge);
    let account = client
        .get_account_with_commitment(mint.pubkey(), CommitmentLevel::Processed)
        .await
        .unwrap()
        .unwrap();
    let mint_data = Mint::unpack(&account.data).unwrap();
    let payload = PayloadAssetMeta {
        token_address: mint.pubkey().to_bytes(),
        token_chain: 1,
        decimals: mint_data.decimals,
        symbol: "USD".to_string(),
        name: "Bitcoin".to_string(),
    };
    let payload = payload.try_to_vec().unwrap();
}

#[tokio::test]
async fn transfer_native() {
    let Context {
        ref payer,
        ref mut client,
        bridge,
        token_bridge,
        ref mint_authority,
        ref mint,
        ref mint_meta,
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
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ref token_authority,
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
    let signature_set = common::verify_signatures(client, &bridge, payer, body, guardian_keys, 0)
        .await
        .unwrap();
    common::post_vaa(client, *bridge, payer, signature_set, vaa.clone())
        .await
        .unwrap();

    let mut msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(&msg_derivation_data, bridge);

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
        ref mint_authority,
        ref mint,
        ref mint_meta,
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
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(&msg_derivation_data, &bridge);

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
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ref token_account,
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
    let mut msg_derivation_data = &PostedVAADerivationData {
        payload_hash: body.to_vec(),
    };
    let message_key =
        PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(&msg_derivation_data, &bridge);

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
