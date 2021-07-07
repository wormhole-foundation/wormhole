#![allow(warnings)]

use rand::Rng;
use borsh::BorshSerialize;
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use hex_literal::hex;
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
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
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

mod common;

const GOVERNANCE_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44, 196,
    163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201, 174, 108,
    197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170, 141, 231, 147,
    88, 97, 199,
];

struct Context {
    public: Vec<[u8; 20]>,
    secret: Vec<SecretKey>,
    seq: Sequencer,
}

/// Small helper to track and provide sequences during tests. This is in particular needed for
/// guardian operations that require them for derivations.
struct Sequencer {
    sequences: std::collections::HashMap<[u8; 32], u64>,
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
    let (public_keys, secret_keys) = common::generate_keys(6);
    let mut context = Context {
        public: public_keys,
        secret: secret_keys,
        seq: Sequencer { 
            sequences: std::collections::HashMap::new(),
        },
    };

    // Initialize the bridge and verify the bridges state.
    test_initialize(&mut context);

    // Tests are currently unhygienic as It's difficult to wrap `solana-test-validator` within the
    // integration tests so for now we work around it by simply chain-calling our tests.
    test_bridge_messages(&mut context);
    test_foreign_bridge_messages(&mut context);
    test_persistent_bridge_messages(&mut context);
    test_guardian_set_change(&mut context);
    test_guardian_set_change_fails(&mut context);
    test_set_fees(&mut context);
    test_transfer_fees(&mut context);
}

fn test_initialize(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Use a timestamp from a few seconds earlier for testing to simulate thread::sleep();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs() - 10;

    common::initialize(client, program, payer, &*context.public.clone(), 500, 5000);
    common::sync(client, payer);

    // Verify the initial bridge state is as expected.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let guardian_set_key = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 0 },
        &program,
    );

    // Fetch account states.
    let bridge: BridgeData = common::get_account_data(client, &bridge_key);
    let guardian_set: GuardianSetData = common::get_account_data(client, &guardian_set_key);

    // Bridge Config should be as expected.
    assert_eq!(bridge.guardian_set_index, 0);
    assert_eq!(bridge.config.guardian_set_expiration_time, 2_000_000_000);
    assert_eq!(bridge.config.fee, 500);
    assert_eq!(bridge.config.fee_persistent, 5000);

    // Guardian set account must also be as expected.
    assert_eq!(guardian_set.index, 0);
    assert_eq!(guardian_set.keys, context.public);
    assert!(guardian_set.creation_time as u64 > now);
}

fn test_bridge_messages(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists.
    let nonce = 12397;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::new();

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
}

fn test_persistent_bridge_messages(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Generate a message we want to persist.
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Confirm that it fails if we try and save a message while paying below the expected fee.
    assert!(common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        4999,
        true,
    ).is_err());

    // Check current balance to verify the right fee is going to be taken.
    let fee_collector = FeeCollector::key(None, &program);
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;

    // Generate a message we want to persist.
    let message = [0u8; 32].to_vec();
    let nonce = rand::thread_rng().gen();

    // This time pay enough to confirm the message succeeds.
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        5000,
        true,
    )
    .unwrap();

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
    common::sync(client, payer);

    // Derive where we expect created accounts to be.
    let signature_set = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
        &SignatureSetDerivationData { hash: body_hash },
        &program,
    );

    // Fetch chain accounts to verify state.
    let posted_message: PostedMessage = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    // Verify on chain Message
    assert_eq!(posted_message.0.vaa_version, 0);
    assert_eq!(posted_message.0.persist, true);
    assert_eq!(posted_message.0.vaa_signature_account, signature_set);
    assert_eq!(posted_message.0.nonce, nonce);
    assert_eq!(posted_message.0.sequence, sequence);
    assert_eq!(posted_message.0.emitter_chain, 1);
    assert_eq!(posted_message.0.payload, message);
    assert_eq!(
        posted_message.0.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body_hash);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        // Sign message locally.
        let (local_sig, recover_id) = secp256k1::sign(&Secp256k1Message::parse(&body_hash), &secret_key);

        // Combine recoverify with signature to match 65 byte layout.
        let mut signature_bytes = [0u8; 65];
        signature_bytes[64] = recover_id.serialize();
        (&mut signature_bytes[0..64]).copy_from_slice(&local_sig.serialize());

        // Signature stored should on chain be as expected.
        assert_eq!(*signature, signature_bytes);
    }

    // Verify fee account gained a persistent fee.
    assert_eq!(
        client.get_account(&fee_collector).unwrap().lamports,
        account_balance + 5000,
    );
}

fn test_guardian_set_change(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(6);

    let nonce = 12398;
    let message = GovernancePayloadGuardianSetChange {
        new_guardian_set_index: 1,
        new_guardian_set: new_public_keys.clone(),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
    common::upgrade_guardian_set(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        0,
        1,
        1,
    )
    .unwrap();

    // Submit the message a second time with a new nonce.
    let nonce = 12399;
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();

    context.public = new_public_keys;
    context.secret = new_secret_keys;

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
}

fn test_guardian_set_change_fails(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::new();

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(6);
    let nonce = 12400;
    let message = GovernancePayloadGuardianSetChange {
        new_guardian_set_index: 2,
        new_guardian_set: new_public_keys.clone(),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);

    assert!(common::upgrade_guardian_set(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        1,
        2,
        0,
    )
    .is_err());
}

fn test_set_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOV_KEY).unwrap();

    let nonce = 12401;
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(100),
        persisted_fee: U256::from(100),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
    common::set_fees(client, program, payer, message_key, emitter.pubkey(), 3).unwrap();

    // Check that posting a new message fails with too small a fee.
    let emitter = Keypair::new();
    let nonce = 12402;
    let message = b"Fail to Pay".to_vec();
    assert!(common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        50,
        false
    )
    .is_err());

    // And succeeds with the new.
    let emitter = Keypair::new();
    let nonce = 12402;
    let message = b"Fail to Pay".to_vec();
    common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        100,
        false,
    )
    .unwrap();
}

fn test_transfer_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOV_KEY).unwrap();

    let nonce = 12403;
    let message = GovernancePayloadTransferFees {
        amount: 100.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
        false,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
    common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        4,
        payer.pubkey(),
    )
    .unwrap();
}

fn test_foreign_bridge_messages(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let nonce = 13832;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::new();

    // Verify the VAA generated on a foreign chain.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 2);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
}
