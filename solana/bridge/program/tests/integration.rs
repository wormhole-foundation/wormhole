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


#[test]
fn run_integration_tests() {
    let (ref payer, ref client, ref program) = common::setup();
    let (public_keys, secret_keys) = common::generate_keys(19);

    common::initialize(client, program, payer, &*public_keys);

    test_bridge_messages(&public_keys, &secret_keys);
    test_guardian_set_change(&public_keys, &secret_keys);
}

fn test_bridge_messages(public_keys: &[[u8; 20]], secret_keys: &[SecretKey]) {
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists.
    let nonce = 12397;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::new();

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone());

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
    common::verify_signatures(client, program, payer, body, body_hash, &secret_keys, 0);
    common::post_vaa(client, program, payer, vaa);
}

fn test_guardian_set_change(public_keys: &[[u8; 20]], secret_keys: &[SecretKey]) {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref payer, ref client, ref program) = common::setup();

    // Data/Nonce used for emitting a message we want to prove exists.
    let nonce = 12397;
    let message = b"Prove Me".to_vec();
    let emitter = Keypair::new();

    // Post the message, publishing the data for guardian consumption.
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone());

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
    common::verify_signatures(client, program, payer, body, body_hash, &secret_keys, 0);
    common::post_vaa(client, program, payer, vaa);

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, new_secret_keys) = common::generate_keys(3);
    let nonce = 12398;
    let message = GovernancePayloadGuardianSetChange {
        new_guardian_set_index: 1,
        new_guardian_set: new_public_keys.clone(),
    }.try_to_vec().unwrap();

    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone());
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 0);
    common::verify_signatures(client, program, payer, body, body_hash, &secret_keys, 0);
    common::post_vaa(client, program, payer, vaa);

    common::upgrade_guardian_set(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        0,
        1,
    );

    // Submit the message a second time with a new nonce.
    let nonce = 12399;
    let message_key = common::post_message(client, program, payer, &emitter, nonce, message.clone());

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, body_hash) = common::generate_vaa(&emitter, message.clone(), nonce, 1);
    common::verify_signatures(client, program, payer, body, body_hash, &new_secret_keys, 1);
    common::post_vaa(client, program, payer, vaa);
}
