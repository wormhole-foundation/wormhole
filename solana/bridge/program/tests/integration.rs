#![allow(warnings)]

use borsh::BorshSerialize;
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
use std::time::{
    Duration,
    SystemTime,
};

use bridge::{
    accounts::GuardianSetDerivationData,
    instruction,
    types::{
        BridgeConfig,
        PostedMessage,
        SequenceTracker,
    },
    Initialize,
};

mod common;

#[test]
fn test_bridge_messages() {
    let (ref payer, ref client, ref program) = common::setup();

    // Initialize the Bridge
    common::initialize(
        client,
        program,
        payer,
        GuardianSetDerivationData { index: 0 },
    );

    // Post a Message
    common::post_message(
        client,
        program,
        payer,
        PostedMessage {
            vaa_version: 0,
            vaa_time: 0,
            vaa_signature_account: Pubkey::new_unique(),

            nonce: 0,
            sequence: 0,
            emitter_chain: 1,
            emitter_address: [0u8; 32],
            payload: vec![],
            submission_time: SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs() as u32,
        },
        0,
    );

    guardian_sign_round();

    // Post VAA
    common::post_vaa(client, program, payer);

    // Verify a Signature
    common::verify_signature(client, program, payer);
}

/// Create a set of signatures for testing.
fn guardian_sign_round() {
}
