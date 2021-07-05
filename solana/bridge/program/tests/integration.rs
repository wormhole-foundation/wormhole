#![allow(warnings)]

use borsh::BorshSerialize;
use secp256k1::Message;
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
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use std::{
    convert::TryInto,
    io::{
        Cursor,
        Write,
    },
};
use std::time::{
    Duration,
    SystemTime,
};
use hex_literal::hex;
use secp256k1::{
    PublicKey,
    SecretKey,
};
use sha3::Digest;

use bridge::{
    accounts::GuardianSetDerivationData,
    instruction,
    types::{
        BridgeConfig,
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        PostedMessage,
        PostedMessageData,
        SequenceTracker,
        SignatureSet,
    },
    Initialize,
    PostVAAData,
    SerializePayload,
    Signature,
};

mod common;

const GOV_KEY: [u8; 64] = [
    240, 133, 120, 113, 30, 67, 38, 184, 197, 72, 234, 99, 241, 21, 58, 225, 41, 157, 171, 44,
    196, 163, 134, 236, 92, 148, 110, 68, 127, 114, 177, 0, 173, 253, 199, 9, 242, 142, 201,
    174, 108, 197, 18, 102, 115, 0, 31, 205, 127, 188, 191, 56, 171, 228, 20, 247, 149, 170,
    141, 231, 147, 88, 97, 199,
];

struct Context {
    public: Vec<[u8; 20]>,
    secret: Vec<SecretKey>,
}

#[test]
fn run_integration_tests() {
    let (ref payer, ref client, ref program) = common::setup();
    let (public_keys, secret_keys) = common::generate_keys(6);
    let mut context = Context {
        public: public_keys,
        secret: secret_keys,
    };

    common::initialize(client, program, payer, &*context.public.clone());

    // Tests are currently unhygienic as It's difficult to wrap `solana-test-validator` within the
    // integration tests so for now we work around it by simply chain-calling our tests.
    test_bridge_messages(&mut context);
    test_guardian_set_change(&mut context);
    test_guardian_set_change_fails(&mut context);
    test_set_fees(&mut context);
}

fn test_bridge_messages(context: &mut Context) {
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists.
    let nonce = 12397;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::new();

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone(), 10_000).unwrap();

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
}

fn test_guardian_set_change(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists.
    let nonce = 12397;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::from_bytes(&GOV_KEY).unwrap();

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone(), 10_000).unwrap();

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 0).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(6);

    let nonce = 12398;
    let message = GovernancePayloadGuardianSetChange {
        new_guardian_set_index: 1,
        new_guardian_set: new_public_keys.clone(),
    }.try_to_vec().unwrap();

    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone(), 10_000).unwrap();
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
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
    ).unwrap();

    // Submit the message a second time with a new nonce.
    let nonce = 12399;
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone(), 10_000).unwrap();

    context.public = new_public_keys;
    context.secret = new_secret_keys;

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1);
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
    }.try_to_vec().unwrap();

    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone(), 10_000).unwrap();
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1);

    assert!(common::upgrade_guardian_set(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        1,
        2,
        0,
    ).is_err());
}

fn test_set_fees(context: &mut Context) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();
    let emitter = Keypair::from_bytes(&GOV_KEY).unwrap();

    let nonce = 12401;
    let message = GovernancePayloadSetMessageFee { fee: 100 }
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
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &context.secret, 1).unwrap();
    common::post_vaa(client, program, payer, vaa).unwrap();
    common::set_fees(client, program, payer, message_key, emitter.pubkey(), 3).unwrap();

    // Check that posting a new message fails with too small a fee.
    let emitter = Keypair::new();
    let nonce = 12402;
    let message = b"Fail to Pay".to_vec();
    assert!(
        common::post_message(client, program, payer, &emitter, nonce, message.clone(), 50).is_err()
    );

    // And succeeds with the new.
    let emitter = Keypair::new();
    let nonce = 12402;
    let message = b"Fail to Pay".to_vec();
    common::post_message(client, program, payer, &emitter, nonce, message.clone(), 100).unwrap();
}
