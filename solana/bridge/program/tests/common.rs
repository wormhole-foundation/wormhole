#![allow(warnings)]

use borsh::{
    BorshDeserialize,
    BorshSerialize,
};
use byteorder::{
    BigEndian,
    WriteBytesExt,
};
use hex_literal::hex;
use libsecp256k1::{
    Message as Secp256k1Message,
    PublicKey,
    SecretKey,
};
use sha3::Digest;
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
use solana_program_test::{
    BanksClient,
    ProgramTest,
};
use solana_sdk::{
    commitment_config::CommitmentLevel,
    hash::Hash,
    secp256k1_instruction::new_secp256k1_instruction,
    signature::{
        read_keypair_file,
        Keypair,
        Signature,
        Signer,
    },
    signers::Signers,
    transaction::Transaction,
    transport::TransportError,
};
use std::{
    convert::TryInto,
    env,
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
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        PostedVAAData,
        PostedVAADerivationData,
        Sequence,
        SequenceDerivationData,
        SequenceTracker,
        SignatureSet,
    },
    instruction,
    instructions,
    types::ConsistencyLevel,
    Initialize,
    InitializeData,
    PostMessageData,
    PostVAAData,
    UninitializedMessage,
    VerifySignaturesData,
};

use solitaire::{
    processors::seeded::Seeded,
    AccountSize,
    AccountState,
};

pub use helpers::*;

/// Simple API wrapper for quickly preparing and sending transactions.
pub async fn execute<T: Signers>(
    client: &mut BanksClient,
    payer: &Keypair,
    signers: &T,
    instructions: &[Instruction],
    commitment_level: CommitmentLevel,
) -> Result<(), TransportError> {
    let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
    let recent_blockhash = client.get_latest_blockhash().await?;
    transaction.sign(signers, recent_blockhash);

    client
        .process_transaction_with_commitment(transaction, commitment_level)
        .await
}

mod helpers {
    use bridge::{
        BridgeData,
        GuardianSetData,
    };
    use solana_program::rent::Rent;
    use solana_program_test::{
        processor,
        ProgramTestContext,
    };
    use solana_sdk::account::Account;
    use solitaire::{
        CreationLamports,
        Derive,
    };

    use super::*;

    /// Initialize the test environment, spins up a solana-test-validator in the background so that
    /// each test has a fresh environment to work within.
    pub async fn setup() -> (BanksClient, Keypair, Pubkey) {
        let program = env::var("BRIDGE_PROGRAM")
            .unwrap_or("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o".to_string())
            .parse::<Pubkey>()
            .unwrap();
        let mut builder = ProgramTest::new(
            "bridge",
            program,
            processor!(instruction::solitaire),
        );

        let (client, payer, _) = builder.start().await;

        (client, payer, program)
    }

    /// Wait for a single transaction to fully finalize, guaranteeing chain state has been
    /// confirmed. Useful for consistently fetching data during state checks.
    pub async fn sync(client: &mut BanksClient, payer: &Keypair) {
        let payer_key = payer.pubkey();
        execute(
            client,
            payer,
            &[payer],
            &[system_instruction::transfer(&payer_key, &payer_key, 1)],
            CommitmentLevel::Confirmed,
        )
        .await
        .unwrap();
    }

    /// Fetch account data, the loop is there to re-attempt until data is available.
    pub async fn get_account_data<T: BorshDeserialize>(
        client: &mut BanksClient,
        account: Pubkey,
    ) -> T {
        let account = client.get_account(account).await.unwrap().unwrap();
        T::try_from_slice(&account.data).unwrap()
    }

    /// Fetch account balance
    pub async fn get_account_balance(client: &mut BanksClient, account: Pubkey) -> u64 {
        client
            .get_account(account)
            .await
            .unwrap()
            .unwrap()
            .lamports
    }

    /// Generate `count` secp256k1 private keys, along with their ethereum-styled public key
    /// encoding: 0x0123456789ABCDEF01234
    pub fn generate_keys(count: u8) -> (Vec<[u8; 20]>, Vec<SecretKey>) {
        use rand::Rng;
        use sha3::Digest;

        let mut rng = rand::thread_rng();

        // Generate Guardian Keys
        let secret_keys: Vec<SecretKey> = std::iter::repeat_with(|| SecretKey::random(&mut rng))
            .take(count as usize)
            .collect();

        (
            secret_keys
                .iter()
                .map(|key| {
                    let public_key = PublicKey::from_secret_key(&key);
                    let mut h = sha3::Keccak256::default();
                    h.write(&public_key.serialize()[1..]).unwrap();
                    let key: [u8; 32] = h.finalize().into();
                    let mut address = [0u8; 20];
                    address.copy_from_slice(&key[12..]);
                    address
                })
                .collect(),
            secret_keys,
        )
    }

    /// Utility function for generating VAA's from message data.
    pub fn generate_vaa(
        emitter: &Keypair,
        data: Vec<u8>,
        nonce: u32,
        sequence: u64,
        guardian_set_index: u32,
        emitter_chain: u16,
    ) -> (PostVAAData, [u8; 32], [u8; 32]) {
        let mut vaa = PostVAAData {
            version: 0,
            guardian_set_index,

            // Body part
            emitter_chain,
            emitter_address: emitter.pubkey().to_bytes(),
            sequence,
            payload: data,
            timestamp: SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs() as u32,
            nonce,
            consistency_level: ConsistencyLevel::Confirmed as u8,
        };

        // Hash data, the thing we wish to actually sign.
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
            v.write_u32::<BigEndian>(vaa.nonce).unwrap();
            v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
            v.write(&vaa.emitter_address).unwrap();
            v.write_u64::<BigEndian>(vaa.sequence).unwrap();
            v.write_u8(vaa.consistency_level).unwrap();
            v.write(&vaa.payload).unwrap();
            v.into_inner()
        };

        // Hash this body, which is expected to be the same as the hash currently stored in the
        // signature account, binding that set of signatures to this VAA.
        let body: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write(body.as_slice()).unwrap();
            h.finalize().into()
        };

        let body_hash: [u8; 32] = {
            let mut h = sha3::Keccak256::default();
            h.write(&body).unwrap();
            h.finalize().into()
        };

        (vaa, body, body_hash)
    }

    pub async fn transfer(
        client: &mut BanksClient,
        from: &Keypair,
        to: &Pubkey,
        lamports: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            from,
            &[from],
            &[system_instruction::transfer(&from.pubkey(), to, lamports)],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn initialize(
        client: &mut BanksClient,
        program: Pubkey,
        payer: &Keypair,
        initial_guardians: &[[u8; 20]],
        fee: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::initialize(
                program,
                payer.pubkey(),
                fee,
                2_000_000_000,
                initial_guardians,
            )
            .unwrap()],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn post_message(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        nonce: u32,
        data: Vec<u8>,
        fee: u64,
    ) -> Result<Pubkey, TransportError> {
        // Transfer money into the fee collector as it needs a balance/must exist.
        let fee_collector = FeeCollector::<'_>::key(None, program);

        let message = Keypair::new();

        // Capture the resulting message, later functions will need this.
        let instruction = instructions::post_message(
            *program,
            payer.pubkey(),
            emitter.pubkey(),
            message.pubkey(),
            nonce,
            data,
            ConsistencyLevel::Confirmed,
        )
        .unwrap();

        execute(
            client,
            payer,
            &[payer, emitter, &message],
            &[
                system_instruction::transfer(&payer.pubkey(), &fee_collector, fee),
                instruction,
            ],
            CommitmentLevel::Processed,
        )
        .await?;

        Ok(message.pubkey())
    }

    pub async fn verify_signatures(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        body: [u8; 32],
        secret_keys: &[SecretKey],
        guardian_set_version: u32,
    ) -> Result<Pubkey, TransportError> {
        let signature_set = Keypair::new();
        let tx_signers = [payer, &signature_set];
        // Push Secp256k1 instructions for each signature we want to verify.
        for (i, key) in secret_keys.iter().enumerate() {
            // Set this signers signature position as present at 0.
            let mut signers = [-1; 19];
            signers[i] = 0;

            execute(
                client,
                payer,
                &tx_signers,
                &[
                    new_secp256k1_instruction(key, &body),
                    instructions::verify_signatures(
                        *program,
                        payer.pubkey(),
                        guardian_set_version,
                        signature_set.pubkey(),
                        VerifySignaturesData { signers },
                    )
                    .unwrap(),
                ],
                CommitmentLevel::Processed,
            )
            .await?;
        }

        Ok(signature_set.pubkey())
    }

    pub async fn post_vaa(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        signature_set: Pubkey,
        vaa: PostVAAData,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::post_vaa(
                *program,
                payer.pubkey(),
                signature_set,
                vaa,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn upgrade_guardian_set(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        payload_message: Pubkey,
        emitter: Pubkey,
        old_index: u32,
        new_index: u32,
        sequence: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::upgrade_guardian_set(
                *program,
                payer.pubkey(),
                payload_message,
                emitter,
                old_index,
                new_index,
                sequence,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn upgrade_contract(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        payload_message: Pubkey,
        emitter: Pubkey,
        new_contract: Pubkey,
        spill: Pubkey,
        sequence: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::upgrade_contract(
                *program,
                payer.pubkey(),
                payload_message,
                emitter,
                new_contract,
                spill,
                sequence,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn set_fees(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        message: Pubkey,
        emitter: Pubkey,
        sequence: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::set_fees(
                *program,
                payer.pubkey(),
                message,
                emitter,
                sequence,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }

    pub async fn transfer_fees(
        client: &mut BanksClient,
        program: &Pubkey,
        payer: &Keypair,
        message: Pubkey,
        emitter: Pubkey,
        recipient: Pubkey,
        sequence: u64,
    ) -> Result<(), TransportError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::transfer_fees(
                *program,
                payer.pubkey(),
                message,
                emitter,
                sequence,
                recipient,
            )],
            CommitmentLevel::Processed,
        )
        .await
    }
}
