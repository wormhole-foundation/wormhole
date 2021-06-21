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
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use std::io::{
    Cursor,
    Write,
};
use std::time::{
    Duration,
    SystemTime,
};
use sha3::Digest;

use bridge::{
    accounts::GuardianSetDerivationData,
    instruction,
    types::{
        BridgeConfig,
        PostedMessage,
        SequenceTracker,
        SignatureSet,
    },
    Initialize,
    PostVAAData,
    Signature,
};

mod common;

#[test]
fn test_bridge_messages() {
    let (ref payer, ref client, ref program) = common::setup();

    // Emitting Entity needs a keypair for signing.
    let emitter = Keypair::new();

    // Data we want to verify exists.
    let data = vec![];

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
        &emitter,
        create_message(data.clone()),
        0,
    );

    let signatures = Keypair::new();

    common::verify_signatures(
        client,
        program,
        payer,
        &signatures,
        GuardianSetDerivationData { index: 0 },
    );

    // Guardians sign, verify, and we produce VAA data here.
    let vaa = guardian_sign_round(
        &signatures,
        &emitter,
        data,
    );

    // Post VAA
    common::post_vaa(
        client,
        program,
        payer,
        GuardianSetDerivationData { index: 0 },
        vaa,
    );

    // // Verify a Signature
    // common::verify_signature(client, program, payer);
}

/// A utility function for emulating what the guardians should be doing, I.E, detecting a message
/// is on the chain and creating a signature set for it.
fn guardian_sign_round(
    signatures: &Keypair,
    emitter: &Keypair,
    data: Vec<u8>
) -> PostVAAData {
    let mut vaa = PostVAAData {
        version: 0,
        guardian_set_index: 0,
        signatures: vec![],

        // Body part
        nonce: 0,
        emitter_chain: 1,
        emitter_address: emitter.pubkey().to_bytes(),
        sequence: 0,
        payload: data,
        timestamp: SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap()
            .as_secs() as u32,
    };

    // Hash data, the thing we wish to actually sign.
    let body = {
        let mut v = Cursor::new(Vec::new());
        v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
        v.write_u32::<BigEndian>(vaa.nonce).unwrap();
        v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
        v.write(&vaa.emitter_address).unwrap();
        v.write(&vaa.payload).unwrap();
        v.into_inner()
    };

    // Hash this body, which is expected to be the same as the hash currently stored in the
    // signature account, binding that set of signatures to this VAA.
    let body_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write(body.as_slice()).unwrap();
        h.finalize().into()
    };

    let signatures = SignatureSet {
        hash: body_hash,
        guardian_set_index: 0,
        signatures: vec![
            [0u8; 32],
            [0u8; 32],
        ],
    };

    vaa
}

fn create_message(data: Vec<u8>) -> PostedMessage {
    PostedMessage {
        vaa_version: 0,
        vaa_time: 0,
        vaa_signature_account: Pubkey::new_unique(),

        nonce: 0,
        sequence: 0,
        emitter_chain: 1,
        emitter_address: [0u8; 32],
        payload: data,
        submission_time: SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap()
            .as_secs() as u32,
    }
}
