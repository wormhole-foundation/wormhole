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
        SignatureSetDerivationData,
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
use std::collections::HashMap;
use std::str::FromStr;
use std::time::UNIX_EPOCH;


mod common;

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
        &context.mint
    ).unwrap();

    // Create Token accounts for use within tests.
    common::create_token_account(
        &context.client,
        &context.payer,
        &context.token_account,
        context.token_authority.pubkey(),
        context.mint.pubkey(),
    ).unwrap();

    common::sync(&context.client, &context.payer);

    // Initialize the bridge and verify the bridges state.
    test_initialize(&mut context);
    test_transfer_native(&mut context);
    test_attest(&mut context);
    test_complete_native(&mut context);
    //test_transfer_wrapped(&mut context);
    //test_complete_wrapped(&mut context);
    //test_register_chain(&mut context);
    //test_create_wrapped(&mut context);
}

fn test_attest(context: &mut Context) -> () {
    println!("Attest");
    use token_bridge::{
        accounts::ConfigAccount,
        types::{
            Config,
            FeeStructure,
        },
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
        types::{
            Config,
            FeeStructure,
        },
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
        ..
    } = context;

    common::transfer_native(
        client,
        token_bridge,
        bridge,
        payer,
        token_account,
        mint.pubkey(),
    )
    .unwrap();
}

fn test_initialize(context: &mut Context) {
    println!("Initialize");
    use token_bridge::{
        accounts::ConfigAccount,
        types::{
            Config,
            FeeStructure,
        },
    };

    let Context {
        ref payer,
        ref client,
        ref bridge,
        ref token_bridge,
        ..
    } = context;

    common::initialize(client, token_bridge, payer, &bridge).unwrap();
    common::sync(client, payer);

    // Verify Token Bridge State
    let config_key = ConfigAccount::<'_, { AccountState::Uninitialized }>::key(None, &token_bridge);
    let config: Config = common::get_account_data(client, &config_key);
    assert_eq!(config.wormhole_bridge, *bridge);
    assert_eq!(config.fees.usd_ephemeral, 0);
    assert_eq!(config.fees.usd_persistent, 0);
}
