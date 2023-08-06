use libsecp256k1::SecretKey;
use rand::Rng;
use solana_program::{
    pubkey::Pubkey,
    system_instruction,
};
use solana_program_test::{
    tokio,
    BanksClient,
};
use solana_sdk::{
    commitment_config::CommitmentLevel,
    signature::{
        Keypair,
        Signer,
    },
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

use bridge::{
    accounts::{
        Bridge,
        BridgeData,
        FeeCollector,
        GuardianSet,
        GuardianSetData,
        GuardianSetDerivationData,
        PostedVAA,
        PostedVAAData,
        PostedVAADerivationData,
        SignatureSetData,
    },
    instructions,
    types::{
        ConsistencyLevel,
        GovernancePayloadGuardianSetChange,
        GovernancePayloadSetMessageFee,
        GovernancePayloadTransferFees,
        GovernancePayloadUpgrade,
    },
    SerializeGovernancePayload,
};
use primitive_types::U256;
use solana_program::rent::Rent;

mod common;

// The pubkey corresponding to this key is "CiByUvEcx7w2HA4VHcPCBUAFQ73Won9kB36zW9VjirSr" and needs
// to be exported as the `EMITTER_ADDRESS` environment variable when building the program bpf in
// order for the governance related tests to pass.
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
}

async fn initialize() -> (Context, BanksClient, Keypair, Pubkey) {
    let (public_keys, secret_keys) = common::generate_keys(6);
    let context = Context {
        public: public_keys,
        secret: secret_keys,
        seq: Sequencer {
            sequences: std::collections::HashMap::new(),
        },
    };
    let (mut client, payer, program) = common::setup().await;

    // Use a timestamp from a few seconds earlier for testing to simulate thread::sleep();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 10;

    common::initialize(&mut client, program, &payer, &context.public, 500)
        .await
        .unwrap();

    // Verify the initial bridge state is as expected.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let guardian_set_key = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 0 },
        &program,
    );

    // Fetch account states.
    let bridge: BridgeData = common::get_account_data(&mut client, bridge_key).await;
    let guardian_set: GuardianSetData =
        common::get_account_data(&mut client, guardian_set_key).await;

    // Bridge Config should be as expected.
    assert_eq!(bridge.guardian_set_index, 0);
    assert_eq!(bridge.config.guardian_set_expiration_time, 2_000_000_000);
    assert_eq!(bridge.config.fee, 500);

    // Guardian set account must also be as expected.
    assert_eq!(guardian_set.index, 0);
    assert_eq!(guardian_set.keys, context.public);
    assert!(guardian_set.creation_time as u64 > now);

    (context, client, payer, program)
}

#[tokio::test]
async fn bridge_messages() {
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Data/Nonce used for emitting a message we want to prove exists. Run this twice to make sure
    // that duplicate data does not clash.
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();

    for _ in 0..2 {
        let nonce = rand::thread_rng().gen();

        // Post the message, publishing the data for guardian consumption.
        let sequence = context.seq.next(emitter.pubkey().to_bytes());
        let message_key = common::post_message(
            client,
            program,
            payer,
            &emitter,
            None,
            nonce,
            message.clone(),
            10_000,
        )
        .await
        .unwrap();

        let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
        assert_eq!(posted_message.message.vaa_version, 0);
        assert_eq!(posted_message.message.consistency_level, 1);
        assert_eq!(posted_message.message.nonce, nonce);
        assert_eq!(posted_message.message.sequence, sequence);
        assert_eq!(posted_message.message.emitter_chain, 1);
        assert_eq!(
            &posted_message.message.emitter_address,
            emitter.pubkey().as_ref()
        );
        assert_eq!(posted_message.message.payload, message);
        assert_eq!(
            posted_message.message.emitter_address,
            emitter.pubkey().to_bytes()
        );

        // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
        let (vaa, body, _body_hash) =
            common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
        let vaa_time = vaa.timestamp;

        let signature_set =
            common::verify_signatures(client, program, payer, body, &context.secret, 0)
                .await
                .unwrap();

        // Derive where we expect the posted VAA to be stored.
        let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
            &PostedVAADerivationData {
                payload_hash: body.to_vec(),
            },
            program,
        );
        common::post_vaa(client, program, payer, signature_set, vaa)
            .await
            .unwrap();

        // Fetch chain accounts to verify state.
        let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
        let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

        // Verify on chain Message
        assert_eq!(posted_message.message.vaa_version, 0);
        assert_eq!(
            posted_message.message.consistency_level,
            ConsistencyLevel::Confirmed as u8
        );
        assert_eq!(posted_message.message.vaa_time, vaa_time);
        assert_eq!(posted_message.message.vaa_signature_account, signature_set);
        assert_eq!(posted_message.message.nonce, nonce);
        assert_eq!(posted_message.message.sequence, sequence);
        assert_eq!(posted_message.message.emitter_chain, 1);
        assert_eq!(
            &posted_message.message.emitter_address,
            emitter.pubkey().as_ref()
        );
        assert_eq!(posted_message.message.payload, message);
        assert_eq!(
            posted_message.message.emitter_address,
            emitter.pubkey().to_bytes()
        );

        // Verify on chain Signatures
        assert_eq!(signatures.hash, body);
        assert_eq!(signatures.guardian_set_index, 0);

        for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
            assert!(*signature);
        }
    }

    // Prepare another message with no data in its message to confirm it succeeds.
    let nonce = rand::thread_rng().gen();
    let message = b"".to_vec();

    // Post the message, publishing the data for guardian consumption.
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(posted_message.message.consistency_level, 1);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let vaa_time = vaa.timestamp;
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();

    // Derive where we expect the posted VAA to be stored.
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

    // Verify on chain Message
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(
        posted_message.message.consistency_level,
        ConsistencyLevel::Confirmed as u8
    );
    assert_eq!(posted_message.message.vaa_time, vaa_time);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert!(*signature);
    }
}

// Make sure that posting messages with account reuse works and only accepts messages with the same
// length.
#[tokio::test]
async fn test_bridge_messages_unreliable() {
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Data/Nonce used for emitting a message we want to prove exists. Run this twice to make sure
    // that duplicate data does not clash.
    let emitter = Keypair::new();
    let message_key = Keypair::new();

    for _ in 0..2 {
        let nonce = rand::thread_rng().gen();
        let message: [u8; 32] = rand::thread_rng().gen();
        let sequence = context.seq.next(emitter.pubkey().to_bytes());

        // Post the message, publishing the data for guardian consumption.
        common::post_message_unreliable(
            client,
            program,
            payer,
            &emitter,
            &message_key,
            nonce,
            message.to_vec(),
            10_000,
        )
        .await
        .unwrap();

        // Verify on chain Message
        let posted_message: PostedVAAData =
            common::get_account_data(client, message_key.pubkey()).await;
        assert_eq!(posted_message.message.vaa_version, 0);
        assert_eq!(posted_message.message.nonce, nonce);
        assert_eq!(posted_message.message.sequence, sequence);
        assert_eq!(posted_message.message.emitter_chain, 1);
        assert_eq!(posted_message.message.payload, message);
        assert_eq!(
            posted_message.message.emitter_address,
            emitter.pubkey().to_bytes()
        );

        // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
        let (vaa, body, _body_hash) =
            common::generate_vaa(&emitter, message.to_vec(), nonce, sequence, 0, 1);
        let signature_set =
            common::verify_signatures(client, program, payer, body, &context.secret, 0)
                .await
                .unwrap();
        common::post_vaa(client, program, payer, signature_set, vaa)
            .await
            .unwrap();
        let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
            &PostedVAADerivationData {
                payload_hash: body.to_vec(),
            },
            program,
        );

        // Fetch chain accounts to verify state.
        let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
        let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

        // Verify on chain vaa
        assert_eq!(posted_message.message.vaa_version, 0);
        assert_eq!(posted_message.message.vaa_signature_account, signature_set);
        assert_eq!(posted_message.message.nonce, nonce);
        assert_eq!(posted_message.message.sequence, sequence);
        assert_eq!(posted_message.message.emitter_chain, 1);
        assert_eq!(posted_message.message.payload, message);
        assert_eq!(
            posted_message.message.emitter_address,
            emitter.pubkey().to_bytes()
        );

        // Verify on chain Signatures
        assert_eq!(signatures.hash, body);
        assert_eq!(signatures.guardian_set_index, 0);

        for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
            assert!(*signature);
        }
    }

    // Make sure that posting a message with a different length fails (<len)
    let nonce = rand::thread_rng().gen();
    let message: [u8; 16] = rand::thread_rng().gen();

    assert!(common::post_message_unreliable(
        client,
        program,
        payer,
        &emitter,
        &message_key,
        nonce,
        message.to_vec(),
        10_000,
    )
    .await
    .is_err());

    // Make sure that posting a message with a different length fails (>len)
    let nonce = rand::thread_rng().gen();
    let message: [u8; 128] = [0u8; 128];

    assert!(common::post_message_unreliable(
        client,
        program,
        payer,
        &emitter,
        &message_key,
        nonce,
        message.to_vec(),
        10_000,
    )
    .await
    .is_err());
}

#[tokio::test]
async fn test_bridge_messages_unreliable_do_not_override_reliable() {
    let (ref mut _context, ref mut client, ref payer, ref program) = initialize().await;

    let emitter = Keypair::new();
    let message_key = Keypair::new();

    let nonce = rand::thread_rng().gen();
    let message: [u8; 32] = rand::thread_rng().gen();

    // Post the message using the reliable method
    common::post_message(
        client,
        program,
        payer,
        &emitter,
        Some(&message_key),
        nonce,
        message.to_vec(),
        10_000,
    )
    .await
    .unwrap();

    // Make sure that posting an unreliable message to the same message account fails
    assert!(common::post_message_unreliable(
        client,
        program,
        payer,
        &emitter,
        &message_key,
        nonce,
        message.to_vec(),
        10_000,
    )
    .await
    .is_err());
}

#[tokio::test]
async fn bridge_works_after_transfer_fees() {
    // This test aims to ensure that the bridge remains operational after the
    // fees have been transferred out.
    // NOTE: the bridge is initialised to take a minimum of 500 in fees.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let fee_collector = FeeCollector::key(None, program);
    let initial_balance = common::get_account_balance(client, fee_collector).await;

    // First, post a message that transfer out 500 lamports.
    // Since posting the message itself costs 500 lamports, this should result
    // in the fee collector's account being reset to the initial amount (which
    // is the rent exemption amount)
    {
        let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
        let sequence = context.seq.next(emitter.pubkey().to_bytes());

        let nonce = rand::thread_rng().gen();
        let message = GovernancePayloadTransferFees {
            amount: 500u128.into(),
            to: payer.pubkey().to_bytes(),
        }
        .try_to_vec()
        .unwrap();

        let message_key = common::post_message(
            client,
            program,
            payer,
            &emitter,
            None,
            nonce,
            message.clone(),
            500,
        )
        .await
        .unwrap();

        common::transfer_fees(
            client,
            program,
            payer,
            message_key,
            emitter.pubkey(),
            payer.pubkey(),
            sequence,
        )
        .await
        .unwrap();
    }

    // Ensure that the account has the same amount of money as we started with
    let account_balance = common::get_account_balance(client, fee_collector).await;
    assert_eq!(account_balance, initial_balance);

    // Next, make sure that we can still post a message.
    {
        let emitter = Keypair::new();

        let nonce = rand::thread_rng().gen();
        let message: [u8; 32] = rand::thread_rng().gen();

        common::post_message(
            client,
            program,
            payer,
            &emitter,
            None,
            nonce,
            message.to_vec(),
            500,
        )
        .await
        .unwrap();
    }
}

// Make sure that solitaire can claim accounts that already hold lamports so the protocol can't be
// DoSd by someone funding derived accounts making CreateAccount fail.
#[tokio::test]
async fn test_bridge_message_prefunded_account() {
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Data/Nonce used for emitting a message we want to prove exists. Run this twice to make sure
    // that duplicate data does not clash.
    let payload = [0u8; 32].to_vec();
    let emitter = Keypair::new();

    let nonce = rand::thread_rng().gen();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Post the message, publishing the data for guardian consumption.
    // Transfer money into the fee collector as it needs a balance/must exist.
    let fee_collector = FeeCollector::<'_>::key(None, program);

    let message = Keypair::new();

    // Fund the message account
    common::execute(
        client,
        payer,
        &[payer],
        &[system_instruction::transfer(
            &payer.pubkey(),
            &message.pubkey(),
            // This is enough to cover the base rent but not enough for account storage
            Rent::default().minimum_balance(0),
        )],
        CommitmentLevel::Processed,
    )
    .await
    .unwrap();

    // Capture the resulting message, later functions will need this.
    let instruction = instructions::post_message(
        *program,
        payer.pubkey(),
        emitter.pubkey(),
        message.pubkey(),
        nonce,
        payload.clone(),
        ConsistencyLevel::Confirmed,
    )
    .unwrap();

    common::execute(
        client,
        payer,
        &[payer, &emitter, &message],
        &[
            system_instruction::transfer(&payer.pubkey(), &fee_collector, 10_000),
            instruction,
        ],
        CommitmentLevel::Processed,
    )
    .await
    .unwrap();

    // Verify on chain Message
    let posted_message: PostedVAAData = common::get_account_data(client, message.pubkey()).await;
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(posted_message.message.payload, payload);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );
}

#[tokio::test]
async fn invalid_emitter() {
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Generate a message we want to persist.
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let _sequence = context.seq.next(emitter.pubkey().to_bytes());

    let fee_collector = FeeCollector::key(None, program);

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
        &[payer, &msg_account],
        &[
            system_instruction::transfer(&payer.pubkey(), &fee_collector, 10_000),
            instruction,
        ],
        solana_sdk::commitment_config::CommitmentLevel::Processed,
    )
    .await
    .is_err());
}

#[tokio::test]
async fn guardian_set_change() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

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
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(posted_message.message.consistency_level, 1);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let vaa_time = vaa.timestamp;
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();
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
    .await
    .unwrap();

    // Derive keys for accounts we want to check.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, program);
    let guardian_set_key = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 1 },
        program,
    );

    // Fetch account states.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let bridge: BridgeData = common::get_account_data(client, bridge_key).await;
    let guardian_set: GuardianSetData = common::get_account_data(client, guardian_set_key).await;

    // Verify on chain Message
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(
        posted_message.message.consistency_level,
        ConsistencyLevel::Confirmed as u8
    );
    assert_eq!(posted_message.message.vaa_time, vaa_time);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

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
    let _message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    context.public = new_public_keys;
    context.secret = new_secret_keys;

    // Emulate Guardian behaviour, verifying the data and publishing signatures/VAA.
    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 1, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 1)
        .await
        .unwrap();
    let vaa_time = vaa.timestamp;
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

    // Verify on chain Message
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(
        posted_message.message.consistency_level,
        ConsistencyLevel::Confirmed as u8
    );
    assert_eq!(posted_message.message.vaa_time, vaa_time);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 1);

    for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert!(*signature);
    }
}

#[tokio::test]
async fn guardian_set_change_fails() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Use a random emitter key to confirm the bridge rejects transactions from non-governance key.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Upgrade the guardian set with a new set of guardians.
    let (new_public_keys, _new_secret_keys) = common::generate_keys(6);
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
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

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
    .await
    .is_err());
}

#[tokio::test]
async fn set_fees() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(100u128),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();
    common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .await
    .unwrap();

    // Fetch Bridge to check on-state value.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, program);
    let fee_collector = FeeCollector::key(None, program);
    let bridge: BridgeData = common::get_account_data(client, bridge_key).await;
    assert_eq!(bridge.config.fee, 100);

    // Check that posting a new message fails with too small a fee.
    let account_balance = common::get_account_balance(client, fee_collector).await;
    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    assert!(common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        50
    )
    .await
    .is_err());

    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        account_balance,
    );

    // And succeeds with the new.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let _message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        100,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    let vaa_time = vaa.timestamp;
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Verify that the fee collector was paid.
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        account_balance + 100,
    );

    // And that the new message is on chain.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

    // Verify on chain Message
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(
        posted_message.message.consistency_level,
        ConsistencyLevel::Confirmed as u8
    );
    assert_eq!(posted_message.message.vaa_time, vaa_time);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert!(*signature);
    }
}

#[tokio::test]
async fn set_fees_fails() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Use a random key to confirm only the governance key is respected.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(100u128),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();
    assert!(common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .await
    .is_err());
}

#[tokio::test]
async fn free_fees() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Set Fees to 0.
    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadSetMessageFee {
        fee: U256::from(0u128),
    }
    .try_to_vec()
    .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();
    common::set_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        sequence,
    )
    .await
    .unwrap();

    // Fetch Bridge to check on-state value.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, program);
    let fee_collector = FeeCollector::key(None, program);
    let bridge: BridgeData = common::get_account_data(client, bridge_key).await;
    assert_eq!(bridge.config.fee, 0);

    // Check that posting a new message is free.
    let account_balance = common::get_account_balance(client, fee_collector).await;
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let _message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        0,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    let vaa_time = vaa.timestamp;
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Verify that the fee collector was paid.
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        account_balance,
    );

    // And that the new message is on chain.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

    // Verify on chain Message
    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(
        posted_message.message.consistency_level,
        ConsistencyLevel::Confirmed as u8
    );
    assert_eq!(posted_message.message.vaa_time, vaa_time);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 1);
    assert_eq!(
        &posted_message.message.emitter_address,
        emitter.pubkey().as_ref()
    );
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert!(*signature);
    }
}

#[tokio::test]
async fn transfer_fees() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100u128.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, program);

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    let previous_balance = common::get_account_balance(client, fee_collector).await;

    common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .await
    .unwrap();
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        previous_balance - 100
    );
}

#[tokio::test]
async fn transfer_fees_fails() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // Use an invalid emitter.
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100u128.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, program);

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    let previous_balance = common::get_account_balance(client, fee_collector).await;

    assert!(common::transfer_fees(
        client,
        program,
        payer,
        message_key,
        emitter.pubkey(),
        payer.pubkey(),
        sequence,
    )
    .await
    .is_err());
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        previous_balance
    );
}

#[tokio::test]
async fn transfer_too_much() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    let nonce = rand::thread_rng().gen();
    let message = GovernancePayloadTransferFees {
        amount: 100_000_000_000u64.into(),
        to: payer.pubkey().to_bytes(),
    }
    .try_to_vec()
    .unwrap();

    // Fetch accounts for chain state checking.
    let fee_collector = FeeCollector::key(None, program);

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    let previous_balance = common::get_account_balance(client, fee_collector).await;

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
    .await
    .is_err());
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        previous_balance
    );
}

#[tokio::test]
async fn foreign_bridge_messages() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let emitter = Keypair::new();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Verify the VAA generated on a foreign chain.
    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 2);

    // Derive where we expect created accounts to be.
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );

    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Fetch chain accounts to verify state.
    let posted_message: PostedVAAData = common::get_account_data(client, message_key).await;
    let signatures: SignatureSetData = common::get_account_data(client, signature_set).await;

    assert_eq!(posted_message.message.vaa_version, 0);
    assert_eq!(posted_message.message.vaa_signature_account, signature_set);
    assert_eq!(posted_message.message.nonce, nonce);
    assert_eq!(posted_message.message.sequence, sequence);
    assert_eq!(posted_message.message.emitter_chain, 2);
    assert_eq!(posted_message.message.payload, message);
    assert_eq!(
        posted_message.message.emitter_address,
        emitter.pubkey().to_bytes()
    );

    // Verify on chain Signatures
    assert_eq!(signatures.hash, body);
    assert_eq!(signatures.guardian_set_index, 0);

    for (signature, _secret_key) in signatures.signatures.iter().zip(context.secret.iter()) {
        assert!(*signature);
    }
}

#[tokio::test]
async fn transfer_total_fails() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Be sure any previous tests have fully committed.

    let fee_collector = FeeCollector::key(None, program);
    let account_balance = common::get_account_balance(client, fee_collector).await;

    // Prepare to remove total balance, adding 10_000 to include the fee we're about to pay.
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
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

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
    .await
    .is_err());

    // The fee should have been paid, but other than that the balance should be exactly the same,
    // I.E non-zero.
    assert_eq!(
        common::get_account_balance(client, fee_collector).await,
        account_balance + 10_000
    );
}

// `solana-program-test` doesn't use an upgradeable loader so it's not currently possible to test
// the contract upgrade logic this way. See https://github.com/solana-labs/solana/issues/22950 for
// more details. This test is here mainly as a reference in case the issue above gets fixed, at
// which point this test can be re-enabled.
#[tokio::test]
#[ignore]
async fn upgrade_contract() {
    // Initialize a wormhole bridge on Solana to test with.
    let (ref mut context, ref mut client, ref payer, ref program) = initialize().await;

    // New Contract Address
    // let new_contract = Pubkey::new_unique();
    let new_contract = *program;

    let nonce = rand::thread_rng().gen();
    let emitter = Keypair::from_bytes(&GOVERNANCE_KEY).unwrap();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let message = GovernancePayloadUpgrade { new_contract }
        .try_to_vec()
        .unwrap();

    let message_key = common::post_message(
        client,
        program,
        payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _body_hash) =
        common::generate_vaa(&emitter, message.clone(), nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();
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
    .await
    .unwrap();
}
