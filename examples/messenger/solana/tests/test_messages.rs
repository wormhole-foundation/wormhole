//! Test the Messenger program.
//!
//! You will need a `bridge.so` somewhere in the path of the tests. The easiest way to do this is
//! to build the Wormhole bridge and copy the `.so` into the current directory:
//!
//! ```
//! $ pushd <BRIDGE>
//! $ EMITTER_ADDRESS="0000000000000000000000000000000000000000000000000000000000000004" cargo build-bpf 
//! $ popd
//! $ cp <BRIDGE>/target/deploy/bridge.so .
//! ```
//!
//! This will give you a BPF ELF that the ProgramTest framework can deploy, if you don't do this
//! ProgramTest will try and compile the processor in native mode which cannot currently handle
//! accounts that change size, which Wormhole relies on.

use std::convert::TryInto;
use std::str::FromStr;

// Solana Requirements
use solana_program::pubkey::Pubkey;
use solana_program_test::{
    processor,
    tokio,
    ProgramTest,
    ProgramTestContext,
};
use solana_sdk::signature::Keypair;
use solana_sdk::signer::Signer;
use solana_sdk::transaction::Transaction;
use solana_sdk::secp256k1_instruction::new_secp256k1_instruction;

// Import necessary components from the Messenger Program so we can test them.
use messenger::Message;
use messenger::process_instruction;
use messenger::instruction::{
    send_message,
    recv_message,
    send_message_raw,
};

// We utilise the bridge_endpoint, which is re-exposed by the SDK, to run instructions against
// within the Solana program test framework.
use wormhole_sdk::Chain;
use wormhole_sdk::MessageData;
use wormhole_sdk::VAA;
use wormhole_sdk::PostVAAData;
use wormhole_sdk::VerifySignaturesData;
use wormhole_sdk::bridge_entrypoint;

// Borsh to deserialise the Wormhole Message account for asserting data.
use borsh::BorshDeserialize;

// Secp256k1 so we can produce a Guardian secret key to test with.
use secp256k1::SecretKey;


/// We need an address to deploy our Messenger program at, we just hardcode one for use throughout
/// the tests.
const ID: Pubkey = Pubkey::new_from_array([2u8; 32]);


#[tokio::test]
pub async fn test_publish_message() {
    // Guardian
    let guardian = hex::decode("B7f0900393F869eE15E00e01Dc71E7ba8590E51f").unwrap();
    let guardian = &guardian.try_into().unwrap();

    // Initialize Test Environment with instruction processors. This lets us load the wormhole
    // processor into scope so we can inspect whether the messages it emits are in fact as we
    // expect them to be.
    let mut context = {
        let mut test = ProgramTest::default();
        test.add_program("bridge", wormhole_sdk::id(), processor!(bridge_entrypoint));
        test.add_program("messenger", ID, processor!(process_instruction));
        test.start_with_context().await
    };

    // Initialize Wormhole
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[wormhole_sdk::instructions::initialize(
                wormhole_sdk::id(),
                context.payer.pubkey(),
                50,
                2_000_000_000,
                &[*guardian],
            ).unwrap()],
            Some(&context.payer.pubkey()),
            &[&context.payer],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    // Message & Emitter Account Keys
    let message = Keypair::new();
    let emitter = wormhole_sdk::emitter(&ID);

    // Submit a cross-chain message via Wormhole.
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[send_message(
                ID,
                context.payer.pubkey(),
                emitter.0,
                message.pubkey(),
                Message {
                    nick: "Alice".to_string(),
                    text: "Hello from Bob!".to_string(),
                },
                0,
            )],
            Some(&context.payer.pubkey()),
            &[&context.payer, &message],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    // We should now be able to find a Message on chain emitted by our contract. Let's verify it
    // contains the expected data.
    let message = MessageData::try_from_slice(
        &context
            .banks_client
            .get_account(message.pubkey())
            .await
            .unwrap()
            .unwrap()
            .data[3..]
    ).unwrap();

    assert_eq!(message.vaa_version, 0);
    assert_eq!(message.consistency_level, 32);
    assert_eq!(message.vaa_time, 0);
    assert_eq!(message.nonce, 0);
    assert_eq!(message.emitter_chain, 1);
    assert_eq!(message.emitter_address, emitter.0.to_bytes());
    assert_eq!(
        Message::try_from_slice(&message.payload).unwrap(),
        Message {
            nick: "Alice".to_string(),
            text: "Hello from Bob!".to_string(),
        }
    );

    // Simulate Guardian behaviour: detecting message, signing, posting VAA.
    let vaa = simulate_guardians(&mut context, &message).await;

    // We can now test the recv_message endpoint by submitting the signed VAA.
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[recv_message(
                ID,
                context.payer.pubkey(),
                vaa,
            )],
            Some(&context.payer.pubkey()),
            &[&context.payer],
            context.last_blockhash,
        ))
        .await
        .unwrap();
}

#[tokio::test]
pub async fn test_publish_message_raw() {
    // Guardian
    let guardian = hex::decode("966745cb54d907a93272dd154e1d1bb94b38c69b").unwrap();
    let guardian = &guardian.try_into().unwrap();

    // Initialize Test Environment with instruction processors. This lets us load the wormhole
    // processor into scope so we can inspect whether the messages it emits are in fact as we
    // expect them to be.
    let mut context = {
        let mut test = ProgramTest::default();
        test.add_program("bridge", wormhole_sdk::id(), processor!(bridge_entrypoint));
        test.add_program("messenger", ID, processor!(process_instruction));
        test.start_with_context().await
    };

    // Initialize Wormhole
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[wormhole_sdk::instructions::initialize(
                wormhole_sdk::id(),
                context.payer.pubkey(),
                50,
                2_000_000_000,
                &[*guardian],
            ).unwrap()],
            Some(&context.payer.pubkey()),
            &[&context.payer],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    // Message & Emitter Account Keys
    let message = Keypair::new();
    let emitter = wormhole_sdk::emitter(&ID);

    // Submit a cross-chain message via Wormhole.
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[send_message_raw(
                ID,
                context.payer.pubkey(),
                emitter.0,
                message.pubkey(),
                Message {
                    nick: "Alice".to_string(),
                    text: "Hello from Bob!".to_string(),
                },
                1,
            )],
            Some(&context.payer.pubkey()),
            &[&context.payer, &message],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    // Check Resulting Message on Chain
    let message = MessageData::try_from_slice(
        &context
            .banks_client
            .get_account(message.pubkey())
            .await
            .unwrap()
            .unwrap()
            .data[3..]
    ).unwrap();

    assert_eq!(message.vaa_version, 0);
    assert_eq!(message.consistency_level, 32);
    assert_eq!(message.vaa_time, 0);
    assert_eq!(message.nonce, 1);
    assert_eq!(message.emitter_chain, 1);
    assert_eq!(message.emitter_address, emitter.0.to_bytes());
    assert_eq!(
        Message::try_from_slice(&message.payload).unwrap(),
        Message {
            nick: "Alice".to_string(),
            text: "Hello from Bob!".to_string(),
        }
    );
}

pub async fn simulate_guardians(context: &mut ProgramTestContext, message: &MessageData) -> Pubkey {
    // Emulate Guardian signatures by signing manually. First we produce a VAA.
    let vaa = {
        let mut vaa           = VAA::default();
        vaa.timestamp         = message.submission_time;
        vaa.nonce             = message.nonce;
        vaa.emitter_chain     = Chain::Solana;
        vaa.emitter_address   = message.emitter_address;
        vaa.sequence          = message.sequence;
        vaa.consistency_level = message.consistency_level;
        vaa.payload           = message.payload.clone();
        vaa
    };

    // Hash the body to produce our message to sign.
    let body = vaa.digest().unwrap();

    // Place to store Signatures on Solana.
    let signatures = Keypair::new();
    let mut signers = [-1; 19];
    signers[0] = 0;

    // Verify Signatures
    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[
                new_secp256k1_instruction(
                    &SecretKey::parse(
                        (&*hex::decode("ff2f9d893e5c12618c442b34a98cfa3f646c402bf5e2a180ce761a7d8a43d452")
                            .unwrap())
                            .try_into()
                            .unwrap()
                    ).unwrap(),
                    &body
                ),
                wormhole_sdk::instructions::verify_signatures(
                    wormhole_sdk::id(),
                    context.payer.pubkey(),
                    0,
                    signatures.pubkey(),
                    VerifySignaturesData { signers },
                ).unwrap(),
            ],
            Some(&context.payer.pubkey()),
            &[&context.payer, &signatures],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    context
        .banks_client
        .process_transaction(Transaction::new_signed_with_payer(
            &[
                wormhole_sdk::instructions::post_vaa(
                    wormhole_sdk::id(),
                    context.payer.pubkey(),
                    signatures.pubkey(),
                    PostVAAData {
                        version:            0,
                        guardian_set_index: 0,
                        timestamp:          vaa.timestamp,
                        nonce:              vaa.nonce,
                        emitter_chain:      vaa.emitter_chain as u16,
                        emitter_address:    vaa.emitter_address,
                        sequence:           vaa.sequence,
                        consistency_level:  vaa.consistency_level,
                        payload:            vaa.payload,
                    },
                ),
            ],
            Some(&context.payer.pubkey()),
            &[&context.payer],
            context.last_blockhash,
        ))
        .await
        .unwrap();

    // Derive VAA Destination.
    Pubkey::find_program_address(
        &[b"PostedVAA", &body],
        &wormhole_sdk::id(),
    ).0
}
