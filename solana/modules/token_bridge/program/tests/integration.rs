#![allow(warnings)]

use borsh::BorshSerialize;
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use hex_literal::hex;
use rand::Rng;
use secp256k1::{
    Message as Secp256k1Message,
    PublicKey,
    SecretKey,
};
use sha3::Digest;
use solana_client::rpc_client::RpcClient;
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
use solana_sdk::{
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};
use std::{
    convert::TryInto,
    io::{
        Cursor,
        Write,
    },
    time::{
        Duration,
        SystemTime,
    },
};

use bridge::{
    accounts::{
        Bridge,
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        Message,
        MessageDerivationData,
        SignatureSet,
    },
    instruction,
    types::{
        BridgeConfig,
        BridgeData,
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        GovernancePayloadTransferFees,
        GuardianSetData,
        PostedMessage,
        PostedMessageData,
        SequenceTracker,
        SignatureSet as SignatureSetData,
    },
    Initialize,
    PostVAA,
    PostVAAData,
    SerializePayload,
    Signature,
};
use primitive_types::U256;
use std::{
    collections::HashMap,
    str::FromStr,
    time::UNIX_EPOCH,
};
use token_bridge::{
    accounts::{
        WrappedDerivationData,
        WrappedMint,
    },
    messages::{
        PayloadAssetMeta,
        PayloadGovernanceRegisterChain,
        PayloadTransfer,
    },
    types::Address,
};

mod common;

const GOVERNANCE_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44, 196,
    163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201, 174, 108,
    197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170, 141, 231, 147,
    88, 97, 199,
];

struct Context {
    /// Address of the core bridge contract.
    bridge: Pubkey,

    /// Shared RPC client for tests to make transactions with.
    client: RpcClient,

    /// Payer key with a ton of lamports to ease testing with.
    payer: Keypair,

    /// Track nonces throughout the tests.
    seq: Sequencer,

    /// Address of the token bridge itself that we wish to test.
    token_bridge: Pubkey,

    /// Keypairs for mint information, required in multiple tests.
    mint_authority: Keypair,
    mint: Keypair,
    mint_meta: Keypair,

    /// Keypairs for test token accounts.
    token_authority: Keypair,
    token_account: Keypair,
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

#[test]
fn run_integration_tests() {
    let (payer, client, bridge, token_bridge) = common::setup();

    // Setup a Bridge to test against.
    println!("Bridge: {}", bridge);
    common::initialize_bridge(&client, &bridge, &payer);

    // Context for test environment.
    let mut context = Context {
        seq: Sequencer {
            sequences: HashMap::new(),
        },
        bridge,
        client,
        payer,
        token_bridge,
        mint_authority: Keypair::new(),
        mint: Keypair::new(),
        mint_meta: Keypair::new(),
        token_account: Keypair::new(),
        token_authority: Keypair::new(),
    };

    // Create a mint for use within tests.
    common::create_mint(
        &context.client,
        &context.payer,
        &context.mint_authority.pubkey(),
        &context.mint,
    )
    .unwrap();

    // Create Token accounts for use within tests.
    common::create_token_account(
        &context.client,
        &context.payer,
        &context.token_account,
        context.token_authority.pubkey(),
        context.mint.pubkey(),
    )
    .unwrap();

    // Mint tokens
    common::mint_tokens(
        &context.client,
        &context.payer,
        &context.mint_authority,
        &context.mint,
        &context.token_account.pubkey(),
        1000,
    )
    .unwrap();

    // Initialize the bridge and verify the bridges state.
    test_initialize(&mut context);
    test_transfer_native(&mut context);
    test_attest(&mut context);
    test_register_chain(&mut context);
    test_transfer_native_in(&mut context);
    let wrapped = test_create_wrapped(&mut context);
    let wrapped_acc = Keypair::new();
    common::create_token_account(
        &context.client,
        &context.payer,
        &wrapped_acc,
        context.token_authority.pubkey(),
        wrapped,
    )
    .unwrap();
    test_transfer_wrapped_in(&mut context, wrapped_acc.pubkey());
    test_transfer_wrapped(&mut context, wrapped_acc.pubkey());
}

fn test_attest(context: &mut Context) -> () {
    println!("Attest");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
        ref bridge,
        ref token_bridge,
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ..
    } = context;

    common::attest(
        client,
        token_bridge,
        bridge,
        payer,
        mint.pubkey(),
        mint_meta.pubkey(),
        0,
    )
    .unwrap();
}

fn test_transfer_native(context: &mut Context) -> () {
    println!("Transfer Native");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
        ref bridge,
        ref token_bridge,
        ref mint_authority,
        ref mint,
        ref mint_meta,
        ref token_account,
        ref token_authority,
        ..
    } = context;

    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        token_account,
        token_authority,
        mint.pubkey(),
        100,
    )
    .unwrap();
}

fn test_transfer_wrapped(context: &mut Context, token_account: Pubkey) -> () {
    println!("Transfer Wrapped");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
        ref bridge,
        ref token_bridge,
        ref mint_authority,
        ref token_authority,
        ..
    } = context;

    common::transfer_wrapped(
        client,
        token_bridge,
        bridge,
        payer,
        token_account,
        token_authority,
        2,
        [1u8; 32],
        10000000,
    )
    .unwrap();
}

fn test_register_chain(context: &mut Context) -> () {
    println!("Register Chain");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
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
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let payload = PayloadGovernanceRegisterChain {
        chain: 2,
        endpoint_address: [0u8; 32],
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, _, _) = common::generate_vaa(emitter.pubkey().to_bytes(), 1, message, nonce, 0);
    let message_key = common::post_vaa(client, bridge, payer, vaa.clone()).unwrap();

    common::register_chain(
        client,
        token_bridge,
        bridge,
        &message_key,
        vaa,
        payload,
        payer,
    )
    .unwrap();
}

fn test_transfer_native_in(context: &mut Context) -> () {
    println!("TransferNativeIn");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
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

    let payload = PayloadTransfer {
        amount: U256::from(100),
        token_address: mint.pubkey().to_bytes(),
        token_chain: 1,
        to: token_account.pubkey().to_bytes(),
        to_chain: 1,
        fee: U256::from(0),
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, _, _) = common::generate_vaa([0u8; 32], 2, message, nonce, 1);
    let message_key = common::post_vaa(client, bridge, payer, vaa.clone()).unwrap();

    common::complete_transfer_native(
        client,
        token_bridge,
        bridge,
        &message_key,
        vaa,
        payload,
        payer,
    )
    .unwrap();
}

fn test_transfer_wrapped_in(context: &mut Context, to: Pubkey) -> () {
    println!("TransferWrapped");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
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

    let payload = PayloadTransfer {
        amount: U256::from(100000000),
        token_address: [1u8; 32],
        token_chain: 2,
        to: to.to_bytes(),
        to_chain: 1,
        fee: U256::from(0),
    };
    let message = payload.try_to_vec().unwrap();

    let (vaa, _, _) = common::generate_vaa([0u8; 32], 2, message, nonce, rand::thread_rng().gen());
    let message_key = common::post_vaa(client, bridge, payer, vaa.clone()).unwrap();

    common::complete_transfer_wrapped(
        client,
        token_bridge,
        bridge,
        &message_key,
        vaa,
        payload,
        payer,
    )
    .unwrap();
}

fn test_create_wrapped(context: &mut Context) -> (Pubkey) {
    println!("CreateWrapped");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
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

    let (vaa, _, _) = common::generate_vaa([0u8; 32], 2, message, nonce, 2);
    let message_key = common::post_vaa(client, bridge, payer, vaa.clone()).unwrap();

    common::create_wrapped(
        client,
        token_bridge,
        bridge,
        &message_key,
        vaa,
        payload,
        payer,
    )
    .unwrap();

    return WrappedMint::<'_, { AccountState::Initialized }>::key(
        &WrappedDerivationData {
            token_chain: 2,
            token_address: [1u8; 32],
        },
        token_bridge,
    );
}

fn test_initialize(context: &mut Context) {
    println!("Initialize");
    use token_bridge::{
        accounts::ConfigAccount,
        types::Config,
    };

    let Context {
        ref payer,
        ref client,
        ref bridge,
        ref token_bridge,
        ..
    } = context;

    common::initialize(client, token_bridge, payer, &bridge).unwrap();

    // Verify Token Bridge State
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &token_bridge);
    let config: Config = common::get_account_data(client, &config_key);
    assert_eq!(config.wormhole_bridge, *bridge);
}
