use bridge::{
    accounts::{
        PostedVAA,
        PostedVAADerivationData,
    },
    SerializePayload,
};

use libsecp256k1::SecretKey;
use nft_bridge::{
    accounts::{
        ConfigAccount,
        WrappedDerivationData,
        WrappedMint,
    },
    messages::{
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    },
    types::Config,
};
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
use std::str::FromStr;

mod common;

const GOVERNANCE_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44, 196,
    163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201, 174, 108,
    197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170, 141, 231, 147,
    88, 97, 199,
];

struct Context {
    /// Guardian secret keys.
    guardian_keys: Vec<SecretKey>,

    /// Address of the core bridge contract.
    bridge: Pubkey,

    /// Shared RPC client for tests to make transactions with.
    client: BanksClient,

    /// Payer key with a ton of lamports to ease testing with.
    payer: Keypair,

    /// Address of the token bridge itself that we wish to test.
    nft_bridge: Pubkey,

    /// Keypairs for mint information, required in multiple tests.
    mint_authority: Keypair,
    mint: Keypair,

    /// Keypairs for test token accounts.
    token_authority: Keypair,
    token_account: Keypair,
    metadata_account: Pubkey,
}

async fn set_up() -> Result<Context, TransportError> {
    let (guardians, guardian_keys) = common::generate_keys(6);

    let (mut client, payer, bridge, nft_bridge) = common::setup().await;

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
    let mut context = Context {
        guardian_keys,
        bridge,
        client,
        payer,
        nft_bridge,
        mint_authority: Keypair::new(),
        mint,
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

    // Create an SPL metadata account for native NFTs.
    common::create_spl_metadata(
        &mut context.client,
        &context.payer,
        context.metadata_account,
        &context.mint_authority,
        context.mint.pubkey(),
        context.payer.pubkey(),
        "Non-Fungible Token".into(),
        "NFT".into(),
        "https://example.com".into(),
    )
    .await?;

    // Mint an NFT.
    common::mint_tokens(
        &mut context.client,
        &context.payer,
        &context.mint_authority,
        &context.mint,
        &context.token_account.pubkey(),
        1,
    )
    .await?;

    // Initialize the nft bridge.
    common::initialize(
        &mut context.client,
        context.nft_bridge,
        &context.payer,
        context.bridge,
    )
    .await
    .unwrap();

    // Verify NFT Bridge State
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &nft_bridge);
    let config: Config = common::get_account_data(&mut context.client, config_key)
        .await
        .unwrap();
    assert_eq!(config.wormhole_bridge, bridge);

    Ok(context)
}

#[tokio::test]
async fn transfer_native() {
    let Context {
        ref payer,
        ref mut client,
        bridge,
        nft_bridge,
        ref mint,
        ref token_account,
        ref token_authority,
        ..
    } = set_up().await.unwrap();

    let message = &Keypair::new();

    common::transfer_native(
        client,
        nft_bridge,
        bridge,
        payer,
        message,
        token_account,
        token_authority,
        mint.pubkey(),
    )
    .await
    .unwrap();
}

async fn register_chain(context: &mut Context) {
    let Context {
        ref payer,
        ref mut client,
        ref bridge,
        ref nft_bridge,
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
        *nft_bridge,
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
        nft_bridge,
        mint_authority: _,
        ref mint,
        ref token_account,
        ref token_authority,
        ref guardian_keys,
        ..
    } = context;

    // Do an initial transfer so that the bridge account owns the NFT.
    let message = &Keypair::new();
    common::transfer_native(
        client,
        nft_bridge,
        bridge,
        payer,
        message,
        token_account,
        token_authority,
        mint.pubkey(),
    )
    .await
    .unwrap();

    let nonce = rand::thread_rng().gen();

    let token_address = [1u8; 32];
    let token_chain = 1;
    let token_id = U256::from_big_endian(&mint.pubkey().to_bytes());

    let associated_addr = spl_associated_token_account::get_associated_token_address(
        &token_authority.pubkey(),
        &mint.pubkey(),
    );

    let payload = PayloadTransfer {
        token_address,
        token_chain,
        symbol: "NFT".into(),
        name: "Non-Fungible Token".into(),
        token_id,
        uri: "https://example.com".to_string(),
        to: associated_addr.to_bytes(),
        to_chain: 1,
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
        nft_bridge,
        bridge,
        message_key,
        vaa,
        payload,
        payer,
        token_authority.pubkey(),
        mint.pubkey(),
    )
    .await
    .unwrap();
}

#[tokio::test]
async fn transfer_wrapped() {
    let mut context = set_up().await.unwrap();
    register_chain(&mut context).await;

    let Context {
        ref payer,
        ref mut client,
        bridge,
        nft_bridge,
        mint_authority: _,
        mint: _,
        token_account: _,
        ref token_authority,
        ref guardian_keys,
        metadata_account: _,
        ..
    } = context;

    let nonce = rand::thread_rng().gen();

    let token_chain = 2;
    let token_address = [7u8; 32];
    let token_id = U256::from_big_endian(&[0x2cu8; 32]);

    let wrapped_mint_key = WrappedMint::<'_, { AccountState::Uninitialized }>::key(
        &WrappedDerivationData {
            token_chain,
            token_address,
            token_id,
        },
        &nft_bridge,
    );
    let associated_addr = spl_associated_token_account::get_associated_token_address(
        &token_authority.pubkey(),
        &wrapped_mint_key,
    );

    let symbol = "UUC";
    let name = "External Token";
    let uri = "https://example.com";
    let payload = PayloadTransfer {
        token_address,
        token_chain,
        symbol: symbol.into(),
        name: name.into(),
        token_id,
        uri: uri.into(),
        to: associated_addr.to_bytes(),
        to_chain: 1,
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

    common::complete_wrapped(
        client,
        nft_bridge,
        bridge,
        message_key,
        vaa.clone(),
        payload.clone(),
        token_authority.pubkey(),
        payer,
    )
    .await
    .unwrap();

    // What this actually does is initialize the spl token metadata so that we can then burn the NFT
    // in the future but of course, we can't call it something useful like
    // `initialize_wrapped_nft_metadata`.
    common::complete_wrapped_meta(client, nft_bridge, bridge, message_key, vaa, payload, payer)
        .await
        .unwrap();

    // Now transfer the wrapped nft back, which will burn it.
    let message = &Keypair::new();
    common::transfer_wrapped(
        client,
        nft_bridge,
        bridge,
        payer,
        message,
        associated_addr,
        token_authority,
        token_chain,
        token_address,
        token_id,
    )
    .await
    .unwrap();
}
