use libsecp256k1::SecretKey;
use rand::Rng;
use solana_program::{
    pubkey::Pubkey,
    system_instruction,
};
use solana_program_test::tokio;
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
use solana_program_test::ProgramTestContext;

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

async fn initialize() -> (Context, ProgramTestContext, Pubkey) {
    let (public_keys, secret_keys) = common::generate_keys(6);
    let context = Context {
        public: public_keys,
        secret: secret_keys,
        seq: Sequencer {
            sequences: std::collections::HashMap::new(),
        },
    };
    let (mut test_ctx, program) = common::setup().await;

    // Use a timestamp from a few seconds earlier for testing to simulate thread::sleep();
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 10;

    common::initialize(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &context.public,
        500,
    )
    .await
    .unwrap();

    // Verify the initial bridge state is as expected.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, &program);
    let guardian_set_key = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
        &GuardianSetDerivationData { index: 0 },
        &program,
    );

    // Fetch account states.
    let bridge: BridgeData = common::get_account_data(&mut test_ctx.banks_client, bridge_key).await;
    let guardian_set: GuardianSetData =
        common::get_account_data(&mut test_ctx.banks_client, guardian_set_key).await;

    // Bridge Config should be as expected.
    assert_eq!(bridge.guardian_set_index, 0);
    assert_eq!(bridge.config.guardian_set_expiration_time, 2_000_000_000);
    assert_eq!(bridge.config.fee, 500);

    // Guardian set account must also be as expected.
    assert_eq!(guardian_set.index, 0);
    assert_eq!(guardian_set.keys, context.public);
    assert!(guardian_set.creation_time as u64 > now);

    (context, test_ctx, program)
}

#[tokio::test]
async fn bridge_messages() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut _context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);
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
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

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

#[test]
fn test_message_account_closed_discriminator_matches_sha256() {
    let hash = solana_program::hash::hash(b"event:MessageAccountClosed");
    let expected = &hash.to_bytes()[..8];
    assert_eq!(
        bridge::MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR, expected,
        "MESSAGE_ACCOUNT_CLOSED_DISCRIMINATOR must equal SHA256(\"event:MessageAccountClosed\")[..8]"
    );
}

#[test]
fn test_event_ix_tag_matches_sha256() {
    // EVENT_IX_TAG should equal SHA256("anchor:event")[0..8] read as a big-endian u64.
    let hash = solana_program::hash::hash(b"anchor:event");
    let mut tag_bytes = [0u8; 8];
    tag_bytes.copy_from_slice(&hash.to_bytes()[..8]);
    let expected = u64::from_be_bytes(tag_bytes);

    assert_eq!(
        solitaire::EVENT_IX_TAG,
        expected,
        "EVENT_IX_TAG must equal SHA256(\"anchor:event\")[..8] as BE u64"
    );
    assert_eq!(
        solitaire::EVENT_IX_TAG_LE,
        expected.to_le_bytes(),
        "EVENT_IX_TAG_LE must be the little-endian encoding"
    );
}

/// Verify that the event CPI guard rejects external callers. Only the program
/// itself (via invoke_signed with the event authority PDA) should be able to
/// invoke the event CPI path.
#[tokio::test]
async fn test_event_cpi_guard_rejects_external_call() {
    let (ref mut _context, ref mut test_ctx, ref program) = initialize().await;
    let (client, payer) = (&mut test_ctx.banks_client, &test_ctx.payer);

    let (event_authority, _) =
        Pubkey::find_program_address(&[solitaire::EVENT_AUTHORITY_SEED], program);

    // Case 1: Pass the correct event authority PDA, but it can't be a signer
    // because only the program itself can sign for its own PDAs.
    let ix_unsigned_pda = solana_program::instruction::Instruction {
        program_id: *program,
        accounts: vec![solana_program::instruction::AccountMeta::new_readonly(
            event_authority,
            false,
        )],
        data: solitaire::EVENT_IX_TAG_LE.to_vec(),
    };
    let result = common::execute(
        client,
        payer,
        &[payer],
        &[ix_unsigned_pda],
        CommitmentLevel::Processed,
    )
    .await;
    assert!(
        result.is_err(),
        "Event CPI must reject when event authority PDA is not a signer"
    );

    // Case 2: Pass a random keypair that IS a signer, but is not the PDA.
    let fake_authority = Keypair::new();
    let ix_wrong_signer = solana_program::instruction::Instruction {
        program_id: *program,
        accounts: vec![solana_program::instruction::AccountMeta::new_readonly(
            fake_authority.pubkey(),
            true,
        )],
        data: solitaire::EVENT_IX_TAG_LE.to_vec(),
    };
    let result = common::execute(
        client,
        payer,
        &[payer, &fake_authority],
        &[ix_wrong_signer],
        CommitmentLevel::Processed,
    )
    .await;
    assert!(
        result.is_err(),
        "Event CPI must reject when signer is not the event authority PDA"
    );
}

// ---------------------------------------------------------------------------
// Rent Reclamation Tests
// ---------------------------------------------------------------------------

/// Submission time 29 days in the past -- still inside the 30-day retention window.
fn recent_submission_time() -> u32 {
    (std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 29 * 24 * 60 * 60) as u32
}

/// Submission time 31 days in the past, just barely outside the 30-day retention window.
fn old_submission_time() -> u32 {
    (std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_secs()
        - 31 * 24 * 60 * 60) as u32
}

#[tokio::test]
async fn test_post_vaa_sets_submission_time() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let client = &mut test_ctx.banks_client;
    let payer = &test_ctx.payer;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [0u8; 32].to_vec();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Post a message and verify signatures.
    common::post_message(
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

    let (vaa, body, _) = common::generate_vaa(&emitter, message, nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(client, program, payer, body, &context.secret, 0)
        .await
        .unwrap();
    common::post_vaa(client, program, payer, signature_set, vaa)
        .await
        .unwrap();

    // Verify submission_time is set on the PostedVAA.
    let message_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );
    let posted_vaa: PostedVAAData = common::get_account_data(client, message_key).await;
    assert!(
        posted_vaa.message.submission_time > 0,
        "PostVAA should set submission_time"
    );
}

#[tokio::test]
async fn test_close_posted_message_rejects_recent() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;
    let client = &mut test_ctx.banks_client;
    let payer = &test_ctx.payer;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [1u8; 32].to_vec();

    // Post a message (reliable, "msg" prefix).
    let message_key = common::post_message(
        client, program, payer, &emitter, None, nonce, message, 10_000,
    )
    .await
    .unwrap();

    // Immediately try to close -- should fail (within retention window).
    let result = common::close_posted_message(client, program, payer, message_key).await;
    assert!(result.is_err(), "Should reject closing a recent message");
}

/// Verify the retention boundary: a message whose submission_time is 29 days ago
/// (inside the 30-day window) should still be rejected.
#[tokio::test]
async fn test_close_posted_message_rejects_within_retention_window() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [9u8; 32].to_vec();

    let message_key = common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message,
        10_000,
    )
    .await
    .unwrap();

    // Set submission_time to 29 days ago -- still inside the 30-day retention window.
    common::set_submission_time(test_ctx, message_key, recent_submission_time()).await;

    let result = common::close_posted_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        message_key,
    )
    .await;
    assert!(
        result.is_err(),
        "Should reject closing a message still within the 30-day retention window"
    );
}

#[tokio::test]
async fn test_close_posted_message_happy_path() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [2u8; 32].to_vec();
    let fee_collector = FeeCollector::<'_>::key(None, program);

    // Post a message.
    let message_key = common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message,
        10_000,
    )
    .await
    .unwrap();

    // Record balances before close.
    let fee_collector_before =
        common::get_account_balance(&mut test_ctx.banks_client, fee_collector).await;
    let message_lamports =
        common::get_account_balance(&mut test_ctx.banks_client, message_key).await;

    // Set submission_time to 0 (epoch) so it passes the 30-day retention check.
    common::set_submission_time(test_ctx, message_key, old_submission_time()).await;

    // Close the message.
    common::close_posted_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        message_key,
    )
    .await
    .unwrap();

    // Verify: fee_collector received the lamports.
    let fee_collector_after =
        common::get_account_balance(&mut test_ctx.banks_client, fee_collector).await;
    assert_eq!(
        fee_collector_after,
        fee_collector_before + message_lamports,
        "Fee collector should receive message account lamports"
    );

    // Verify: message account is closed (empty data or doesn't exist).
    let msg_data = common::get_account_data_raw(&mut test_ctx.banks_client, message_key).await;
    assert!(
        msg_data.is_none() || msg_data.unwrap().iter().all(|&b| b == 0),
        "Message account should be closed"
    );

    // Verify: bridge.last_lamports is updated.
    let bridge_key = Bridge::<'_, { AccountState::Uninitialized }>::key(None, program);
    let bridge: BridgeData = common::get_account_data(&mut test_ctx.banks_client, bridge_key).await;
    assert_eq!(
        bridge.last_lamports, fee_collector_after,
        "bridge.last_lamports should be updated"
    );
}

#[tokio::test]
async fn test_close_posted_message_unreliable() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [3u8; 32].to_vec();
    let message_key = Keypair::new();

    // Post an unreliable message ("msu" prefix).
    common::post_message_unreliable(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        &message_key,
        nonce,
        message,
        10_000,
    )
    .await
    .unwrap();

    // Set submission_time to 0 so it passes the retention check.
    common::set_submission_time(test_ctx, message_key.pubkey(), old_submission_time()).await;

    // Close should succeed for "msu" prefix too.
    common::close_posted_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        message_key.pubkey(),
    )
    .await
    .unwrap();

    // Verify account is closed.
    let data = common::get_account_data_raw(&mut test_ctx.banks_client, message_key.pubkey()).await;
    assert!(
        data.is_none() || data.unwrap().iter().all(|&b| b == 0),
        "Unreliable message account should be closed"
    );
}

#[tokio::test]
async fn test_close_signature_set_and_posted_vaa_happy_path() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [4u8; 32].to_vec();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());
    let fee_collector = FeeCollector::<'_>::key(None, program);

    // Post message, verify signatures, post VAA.
    common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _) = common::generate_vaa(&emitter, message, nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        body,
        &context.secret,
        0,
    )
    .await
    .unwrap();
    common::post_vaa(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        signature_set,
        vaa,
    )
    .await
    .unwrap();

    let posted_vaa_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );

    // Record balances.
    let fee_collector_before =
        common::get_account_balance(&mut test_ctx.banks_client, fee_collector).await;
    let sig_lamports = common::get_account_balance(&mut test_ctx.banks_client, signature_set).await;
    let vaa_lamports =
        common::get_account_balance(&mut test_ctx.banks_client, posted_vaa_key).await;

    // Set submission_time to 0 on the PostedVAA so it passes the retention check.
    common::set_submission_time(test_ctx, posted_vaa_key, old_submission_time()).await;

    // Close both.
    common::close_signature_set_and_posted_vaa(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        signature_set,
        posted_vaa_key,
        0,
    )
    .await
    .unwrap();

    // Verify fee_collector received all lamports.
    let fee_collector_after =
        common::get_account_balance(&mut test_ctx.banks_client, fee_collector).await;
    assert_eq!(
        fee_collector_after,
        fee_collector_before + sig_lamports + vaa_lamports,
        "Fee collector should receive both accounts' lamports"
    );

    // Verify both accounts are closed.
    let sig_data = common::get_account_data_raw(&mut test_ctx.banks_client, signature_set).await;
    assert!(
        sig_data.is_none() || sig_data.unwrap().iter().all(|&b| b == 0),
        "Signature set should be closed"
    );
    let vaa_data = common::get_account_data_raw(&mut test_ctx.banks_client, posted_vaa_key).await;
    assert!(
        vaa_data.is_none() || vaa_data.unwrap().iter().all(|&b| b == 0),
        "PostedVAA should be closed"
    );
}

#[tokio::test]
async fn test_close_signature_set_and_posted_vaa_rejects_recent() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [5u8; 32].to_vec();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Full flow: post message, verify sigs, post VAA.
    common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (vaa, body, _) = common::generate_vaa(&emitter, message, nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        body,
        &context.secret,
        0,
    )
    .await
    .unwrap();
    common::post_vaa(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        signature_set,
        vaa,
    )
    .await
    .unwrap();

    let posted_vaa_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );

    // Immediately try to close -- should fail (recent VAA).
    let result = common::close_signature_set_and_posted_vaa(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        signature_set,
        posted_vaa_key,
        0,
    )
    .await;
    assert!(
        result.is_err(),
        "Should reject closing recent sig set + VAA"
    );
}

#[tokio::test]
async fn test_close_signature_set_no_vaa_active_guardian_rejects() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [6u8; 32].to_vec();
    let sequence = context.seq.next(emitter.pubkey().to_bytes());

    // Post message and verify signatures, but do NOT post VAA.
    common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    let (_, body, _) = common::generate_vaa(&emitter, message, nonce, sequence, 0, 1);
    let signature_set = common::verify_signatures(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        body,
        &context.secret,
        0,
    )
    .await
    .unwrap();

    // The PostedVAA PDA doesn't exist (never posted).
    let posted_vaa_key = PostedVAA::<'_, { AccountState::MaybeInitialized }>::key(
        &PostedVAADerivationData {
            payload_hash: body.to_vec(),
        },
        program,
    );

    // Try to close with active guardian set -- should fail.
    let result = common::close_signature_set_and_posted_vaa(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        signature_set,
        posted_vaa_key,
        0,
    )
    .await;
    assert!(
        result.is_err(),
        "Should reject closing sig set when guardian set is still active"
    );
}

#[tokio::test]
async fn test_fee_enforcement_after_close() {
    let (ref mut context, ref mut test_ctx, ref program) = initialize().await;

    let emitter = Keypair::new();
    let nonce = rand::thread_rng().gen();
    let message = [7u8; 32].to_vec();

    // Post a message.
    let message_key = common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce,
        message.clone(),
        10_000,
    )
    .await
    .unwrap();

    // Set submission_time to 0 so it passes retention check, then close.
    common::set_submission_time(test_ctx, message_key, old_submission_time()).await;

    common::close_posted_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        message_key,
    )
    .await
    .unwrap();

    // Now post another message -- fee should still be enforced.
    let nonce2 = rand::thread_rng().gen();
    let message2 = [8u8; 32].to_vec();
    let result = common::post_message(
        &mut test_ctx.banks_client,
        program,
        &test_ctx.payer,
        &emitter,
        None,
        nonce2,
        message2,
        10_000,
    )
    .await;
    assert!(
        result.is_ok(),
        "Should still be able to post messages after close"
    );
}
