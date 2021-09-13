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
        BridgeConfig,
        BridgeData,
        FeeCollector,
        GuardianSet,
        GuardianSetData,
        GuardianSetDerivationData,
        MessageData,
        PostedVAA,
        PostedVAAData,
        PostedVAADerivationData,
        SequenceTracker,
        SignatureSet,
        SignatureSetData,
    },
    instruction,
    instructions::hash_vaa,
    types::{
        ConsistencyLevel,
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        GovernancePayloadTransferFees,
        GovernancePayloadUpgrade,
    },
    Initialize,
    PostVAA,
    PostVAAData,
    SerializeGovernancePayload,
    Signature,
};
use primitive_types::U256;
use solana_sdk::hash::hashv;

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
    test_invalid_emitter(&mut context);
    test_duplicate_messages_fail(&mut context);
    test_guardian_set_change(&mut context);
    test_guardian_set_change_fails(&mut context);
    test_set_fees(&mut context);
    test_set_fees_fails(&mut context);
    test_free_fees(&mut context);
    test_transfer_fees(&mut context);
    test_transfer_fees_fails(&mut context);
    test_transfer_too_much(&mut context);
    test_transfer_total_fails(&mut context);
}

fn test_initialize(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Use a timestamp from a few seconds earlier for testing to simulate thread::sleep();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 10;

    common::initialize(client, program, payer, &*context.public.clone(), 500);
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

    // Guardian set account must also be as expected.
    assert_eq!(guardian_set.index, 0);
    assert_eq!(guardian_set.keys, context.public);
    assert!(guardian_set.creation_time as u64 > now);
}

fn test_bridge_messages(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists. Run this twice to make sure
    // that duplicate data does not clash.
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();

    for _ in 0..2 {
        let nonce = rand::thread_rng().gen();
        let sequence = context.seq.next(emitter.pubkey().to_bytes());

        // Post the message, publishing the data for guardian consumption.
        let message_key = common::post_message(
            client,
            program,
            payer,
            &emitter,
            nonce,
            message.clone(),
            10_000,
        )
        .unwrap();

        // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
        let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
        let signature_set =
            common::verify_signatures(client, program, payer, body, &context.secret, 0).unwrap();
        common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
        common::sync(client, payer);

        // Fetch chain accounts to verify state.
        let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
        let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

        // Verify on chain Message
        assert_eq!(posted_message.0.vaa_version, 0);
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
        assert_eq!(signatures.hash, body);
        assert_eq!(signatures.guardian_set_index, 0);

        for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
            assert_eq!(*signature, true);
        }
    }

    // Prepare another message with no data in its message to confirm it succeeds.
    let nonce = rand::thread_rng().gen();
    let message = b"".to_vec();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::sync(client, payer);

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    // Verify on chain Message
    assert_eq!(posted_message.0.vaa_version, 0);
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
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert_eq!(*signature, true);
    }
}

fn test_invalid_emitter(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Generate a message we want to persist.
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let fee_collector = FeeCollector::key(None, &program);

    let msg_account = Keypair::new();
    // Manually send a message that isn't signed by the emitter, which should be rejected to
    // prevent fraudulant transactions sent on behalf of an emitter.
    let mut instruction = bridge::instructions::post_message(
        *program,
        payer.pubkey(),
        emitter.pubkey(),
        msg_account.pubkey(),
        nonce,
        message,
        ConsistencyLevel::Confirmed,
    )
    .unwrap();

    // Modify account list to not require the emitter signs.
    instruction.accounts[2].is_signer = false;

    // Executing this should fail.
    assert!(common::execute(
        client,
        payer,
        &[payer],
        &[
            system_instruction::transfer(&payer.pubkey(), &fee_collector, 10_000),
            instruction,
        ],
        solana_sdk::commitment_config::CommitmentConfig::processed(),
    )
    .is_err());
}

fn test_duplicate_messages_fail(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // We'll use the following nonce/message/emitter/sequence twice.
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    // Second should fail due to duplicate derivations.
    assert!(common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .is_err());
}

fn test_guardian_set_change(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Use a timestamp from a few seconds earlier for testing to simulate thread::sleep();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 10;

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(1);

    let nonce = rand::thread_rng().gen();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    println!("{}", emitter.pubkey().to_string());
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::upgrade_guardian_set(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        0,
        1,
        sequence,
    )
    .unwrap();
    common::sync(client, payer);

    // Derive keys for accounts we want to check.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let guardian_set_key = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 1 },
        &program,
    );

    // Fetch account states.
    let bridge: BridgeData = common::get_account_data(client, &bridge_key);
    let guardian_set: GuardianSetData = common::get_account_data(client, &guardian_set_key);

    // Confirm the bridge now has a new guardian set, and no other fields have shifted.
    assert_eq!(bridge.guardian_set_index, 1);
    assert_eq!(bridge.config.guardian_set_expiration_time, 2_000_000_000);
    assert_eq!(bridge.config.fee, 500);

    // Verify Created Guardian Set
    assert_eq!(guardian_set.index, 1);
    assert_eq!(guardian_set.keys, new_public_keys);
    assert!(guardian_set.creation_time as u64 > now);

    // Submit the message a second time with a new nonce.
    let nonce = rand::thread_rng().gen();
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    context.public = new_public_keys;
    context.secret = new_secret_keys;

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::sync(client, payer);

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    // Verify on chain Message
    assert_eq!(posted_message.0.vaa_version, 0);
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
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 1);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert_eq!(*signature, true);
    }
}

fn test_guardian_set_change_fails(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Use a random emitter key to confirm the bridge rejects transactions from non-governance key.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(6);
    let nonce = rand::thread_rng().gen();
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
        sequence,
    )
    .is_err());
}

fn test_set_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(100),
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .unwrap();
    common::sync(client, payer);

    // Fetch Bridge to check on-state value.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let fee_collector = FeeCollector::key(None, &program);
    let bridge: BridgeData = common::get_account_data(client, &bridge_key);
    assert_eq!(bridge.config.fee, 100);

    // Check that posting a new message fails with too small a fee.
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;
    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    assert!(
        common::post_message(client, program, payer, &emitter, nonce, message.clone(), 50).is_err()
    );
    common::sync(client, payer);

    assert_eq!(
        client.get_account(&fee_collector).unwrap().lamports,
        account_balance,
    );

    // And succeeds with the new.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        100,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::sync(client, payer);

    // Verify that the fee collector was paid.
    assert_eq!(
        client.get_account(&fee_collector).unwrap().lamports,
        account_balance + 100,
    );

    // And that the new message is on chain.
    let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    // Verify on chain Message
    assert_eq!(posted_message.0.vaa_version, 0);
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
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 1);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert_eq!(*signature, true);
    }
}

fn test_set_fees_fails(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Use a random key to confirm only the governance key is respected.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(100),
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    assert!(common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .is_err());
    common::sync(client, payer);
}

fn test_free_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Set Fees to 0.
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee { fee: U256::from(0) }
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .unwrap();
    common::sync(client, payer);

    // Fetch Bridge to check on-state value.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let fee_collector = FeeCollector::key(None, &program);
    let bridge: BridgeData = common::get_account_data(client, &bridge_key);
    assert_eq!(bridge.config.fee, 0);

    // Check that posting a new message is free.
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let message_key =
        common::post_message(client, program, payer, &emitter, nonce, message.clone(), 0).unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::sync(client, payer);

    // Verify that the fee collector was paid.
    assert_eq!(
        client.get_account(&fee_collector).unwrap().lamports,
        account_balance,
    );

    // And that the new message is on chain.
    let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    // Verify on chain Message
    assert_eq!(posted_message.0.vaa_version, 0);
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
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 1);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert_eq!(*signature, true);
    }
}

fn test_transfer_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let recipient = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, &program);
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .unwrap();
    common::sync(client, payer);
}

fn test_transfer_fees_fails(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Use an invalid emitter.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let recipient = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, &program);
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();

    assert!(common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .is_err());
}

fn test_transfer_too_much(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let recipient = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100_000_000_000u64.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, &program);
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        nonce,
        message.clone(),
        10_000,
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();

    // Should fail to transfer.
    assert!(common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .is_err());
}

fn test_foreign_bridge_messages(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Verify the VAA generated on a foreign chain.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 2);

    // Derive where we expect created accounts to be.
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: hash_vaa(&vaa).to_vec(),
        },
        &program,
    );

    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::sync(client, payer);

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, &message_key);
    let signatures: SignatureSetData = common::get_account_data(client, &signature_set);

    assert_eq!(posted_message.0.vaa_version, 0);
    assert_eq!(posted_message.0.vaa_signature_account, signature_set);
    assert_eq!(posted_message.0.nonce, nonce);
    assert_eq!(posted_message.0.sequence, sequence);
    assert_eq!(posted_message.0.emitter_chain, 2);
    assert_eq!(posted_message.0.payload, message);
    assert_eq!(
        posted_message.0.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert_eq!(*signature, true);
    }
}

fn test_transfer_total_fails(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Be sure any previous tests have fully committed.
    common::sync(client, payer);

    let fee_collector = FeeCollector::key(None, &program);
    let account_balance = client.get_account(&fee_collector).unwrap().lamports;

    // Prepare to remove total balance, adding 10_000 to include the fee we're about to pay.
    let recipient = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: (account_balance + 10_000).into(),
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();

    // Transferring total fees should fail, to prevent the account being de-allocated.
    assert!(common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .is_err());
    common::sync(client, payer);

    // The fee should have been paid, but other than that the balance should be exactly the same,
    // I.E non-zero.
    assert_eq!(
        client.get_account(&fee_collector).unwrap().lamports,
        account_balance + 10_000
    );
}

fn test_upgrade_contract(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(1);

    // New Contract Address
    let new_contract = Pubkey::new_unique();

    let nonce = rand::thread_rng().gen();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let message = GovernancePayloadUpgrade {
        new_contract: new_contract.clone(),
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
    )
    .unwrap();

    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0, 1);
    let signature_set =
        common::verify_signatures(client, program, payer, body, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa).unwrap();
    common::upgrade_contract(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        new_contract,
        Pubkey::new_unique(),
        sequence,
    )
    .unwrap();
    common::sync(client, payer);
}
