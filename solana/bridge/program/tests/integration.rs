#![allow(warnings)]

use borsh::BorshSerialize;
use secp256k1::{Message};

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
 
use std::convert::TryInto;
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

    // Guardians sign, verify, and we produce VAA data here.
    let (vaa, body, body_hash, secret_key) = guardian_sign_round(
        &emitter,
        data,
    );

    common::verify_signatures(
        client,
        program,
        payer,
        body,
        body_hash,
        secret_key,
        GuardianSetDerivationData { index: 0 },
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
    emitter: &Keypair,
    data: Vec<u8>
) -> (PostVAAData, Vec<u8>, [u8; 32], secp256k1::SecretKey) {
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

    // Public Key: 0x1d72877eb2d898738afe94c6101152ede0435de9
    let secret_key = secp256k1::SecretKey::parse(&[
      0x99, 0x70, 0x1c, 0x80, 0x5e, 0xf9, 0x38, 0xe1, 0x3f, 0x0e, 0x48, 0xf0, 0x9e, 0x2c, 0x32,
      0x78, 0x91, 0xc1, 0xd8, 0x47, 0x29, 0xd1, 0x52, 0xf3, 0x01, 0xe7, 0xe6, 0x2c, 0xbf, 0x1f,
      0x91, 0xc9
    ]).unwrap();

    let public_key = secp256k1::PublicKey::from_secret_key(&secret_key);
    println!("{}", hex::encode(&public_key.serialize()));

    // Hash this body, which is expected to be the same as the hash currently stored in the
    // signature account, binding that set of signatures to this VAA.
    let body_hash: [u8; 32] = {
        let mut h = sha3::Keccak256::default();
        h.write(body.as_slice()).unwrap();
        h.finalize().into()
    };

    println!("Ahs: {:?}", body_hash);

    // Sign the body hash of the VAA.
    let sig = secp256k1::sign(
        &Message::parse(&body_hash),
        &secret_key,
    );

    // Insert signature into VAA.
    let signature = sig.0.serialize();
    vaa.signatures.push(Signature {
        index: 0,
        r:     signature[0..32].try_into().unwrap(),
        s:     signature[32..64].try_into().unwrap(),
        v:     sig.1.serialize(),
    });

    (vaa, body, body_hash, secret_key)
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
