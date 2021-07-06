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
use secp256k1::{
    Message as Secp256k1Message,
    PublicKey,
    SecretKey,
};
use sha3::Digest;
use solana_client::{
    client_error::ClientError,
    rpc_client::RpcClient,
    rpc_config::RpcSendTransactionConfig,
};
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
    commitment_config::CommitmentConfig,
    secp256k1_instruction::new_secp256k1_instruction,
    signature::{
        read_keypair_file,
        Keypair,
        Signature,
        Signer,
    },
    transaction::Transaction,
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
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        MessageDerivationData,
        Sequence,
        SequenceDerivationData,
        SignatureSet,
        SignatureSetDerivationData,
    },
    instruction,
    instructions,
    types::{
        BridgeConfig,
        PostedMessage,
        SequenceTracker,
    },
    Initialize,
    InitializeData,
    PostMessageData,
    PostVAAData,
    UninitializedMessage,
    VerifySignaturesData,
};

use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

pub use helpers::*;

/// Simple API wrapper for quickly preparing and sending transactions.
fn execute(
    client: &RpcClient,
    payer: &Keypair,
    signers: &[&Keypair],
    instructions: &[Instruction],
) -> Result<Signature, ClientError> {
    let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
    let recent_blockhash = client.get_recent_blockhash().unwrap().0;
    transaction.sign(&signers.to_vec(), recent_blockhash);
    client.send_and_confirm_transaction_with_spinner_and_config(
        &transaction,
        CommitmentConfig::processed(),
        RpcSendTransactionConfig {
            skip_preflight: true,
            preflight_commitment: None,
            encoding: None,
        },
    )
}

mod helpers {
    use super::*;

    /// Initialize the test environment, spins up a solana-test-validator in the background so that
    /// each test has a fresh environment to work within.
    pub fn setup() -> (Keypair, RpcClient, Pubkey) {
        let payer = env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string());
        let rpc_address = env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string());
        let payer = read_keypair_file(payer).unwrap();
        let rpc = RpcClient::new(rpc_address);
        let program = env::var("BRIDGE_PROGRAM")
            .unwrap_or("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o".to_string())
            .parse::<Pubkey>()
            .unwrap();
        (payer, rpc, program)
    }

    /// Fetch account data, the loop is there to re-attempt until data is available.
    pub fn get_account_data<T: BorshDeserialize>(
        client: &RpcClient,
        account: &Pubkey,
    ) -> Option<T> {
        for _ in 0..5 {
            if let Ok(account) = client.get_account(account) {
                return Some(T::try_from_slice(&account.data).unwrap());
            }
            std::thread::sleep(std::time::Duration::from_millis(2000));
        }
        None
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
        guardian_set_index: u32,
    ) -> (PostVAAData, Vec<u8>, [u8; 32]) {
        let mut vaa = PostVAAData {
            version: 0,
            guardian_set_index,

            // Body part
            emitter_chain: 1,
            emitter_address: emitter.pubkey().to_bytes(),
            sequence: 0,
            payload: data,
            timestamp: SystemTime::now()
                .duration_since(SystemTime::UNIX_EPOCH)
                .unwrap()
                .as_secs() as u32,
            nonce,
        };

        // Hash data, the thing we wish to actually sign.
        let body = {
            let mut v = Cursor::new(Vec::new());
            v.write_u32::<BigEndian>(vaa.timestamp).unwrap();
            v.write_u32::<BigEndian>(vaa.nonce).unwrap();
            v.write_u16::<BigEndian>(vaa.emitter_chain).unwrap();
            v.write(&vaa.emitter_address).unwrap();
            v.write_u64::<BigEndian>(vaa.sequence).unwrap();
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

        (vaa, body, body_hash)
    }

    pub fn transfer(
        client: &RpcClient,
        from: &Keypair,
        to: &Pubkey,
        lamports: u64,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            from,
            &[from],
            &[system_instruction::transfer(&from.pubkey(), to, lamports)],
        )
    }

    pub fn initialize(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        initial_guardians: &[[u8; 20]],
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::initialize(
                *program,
                payer.pubkey(),
                500,
                500,
                2_000_000_000,
                initial_guardians,
            )
            .unwrap()],
        )
    }

    pub fn post_message(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        nonce: u32,
        data: Vec<u8>,
        fee: u64,
        persist: bool,
    ) -> Result<Pubkey, ClientError> {
        // Transfer money into the fee collector as it needs a balance/must exist.
        let fee_collector = FeeCollector::<'_>::key(None, program);

        // Capture the resulting message, later functions will need this.
        let (message_key, instruction) = instructions::post_message(
            *program,
            payer.pubkey(),
            emitter.pubkey(),
            nonce,
            data,
            persist,
        )
        .unwrap();

        execute(
            client,
            payer,
            &[payer, emitter],
            &[
                system_instruction::transfer(&payer.pubkey(), &fee_collector, fee),
                instruction,
            ],
        )?;

        Ok(message_key)
    }

    pub fn verify_signatures(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        body: Vec<u8>,
        body_hash: [u8; 32],
        secret_keys: &[SecretKey],
        guardian_set_version: u32,
    ) -> Result<(), ClientError> {
        // Push Secp256k1 instructions for each signature we want to verify.
        for (i, key) in secret_keys.iter().enumerate() {
            // Set this signers signature position as present at 0.
            let mut signers = [-1; 19];
            signers[i] = 0;

            execute(
                client,
                payer,
                &[payer],
                &vec![
                    new_secp256k1_instruction(&key, &body),
                    instructions::verify_signatures(
                        *program,
                        payer.pubkey(),
                        guardian_set_version,
                        VerifySignaturesData {
                            hash: body_hash,
                            initial_creation: true,
                            signers,
                        },
                    )
                    .unwrap(),
                ],
            )?;
        }
        Ok(())
    }

    pub fn post_vaa(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        vaa: PostVAAData,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::post_vaa(*program, payer.pubkey(), vaa)],
        )
    }

    pub fn upgrade_contract(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        payload_message: Pubkey,
        spill: Pubkey,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::upgrade_contract(
                *program,
                payer.pubkey(),
                payload_message,
                spill,
            )],
        )
    }

    pub fn upgrade_guardian_set(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        payload_message: Pubkey,
        emitter: Pubkey,
        old_index: u32,
        new_index: u32,
        sequence: u64,
    ) -> Result<Signature, ClientError> {
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
        )
    }

    pub fn set_fees(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        message: Pubkey,
        emitter: Pubkey,
        sequence: u64,
    ) -> Result<Signature, ClientError> {
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
        )
    }

    pub fn transfer_fees(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        message: Pubkey,
        emitter: Pubkey,
        sequence: u64,
        recipient: Pubkey,
    ) -> Result<Signature, ClientError> {
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
        )
    }
}
