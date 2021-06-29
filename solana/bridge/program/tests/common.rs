#![allow(warnings)]

use borsh::BorshSerialize;
use secp256k1::SecretKey;
use solana_client::{
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

use std::{
    env,
    time::{
        Duration,
        SystemTime,
    },
};

use solana_sdk::{
    commitment_config::CommitmentConfig,
    secp256k1_instruction::new_secp256k1_instruction,
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
};

use bridge::{
    accounts::{
        FeeCollector,
        GuardianSet,
        GuardianSetDerivationData,
        Message,
        MessageDerivationData,
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

mod helpers {
    use super::*;

    fn execute(
        client: &RpcClient,
        payer: &Keypair,
        signers: &[&Keypair],
        instructions: &[Instruction],
    ) {
        let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;
        transaction.sign(&signers.to_vec(), recent_blockhash);
        client
            .send_and_confirm_transaction_with_spinner_and_config(
                &transaction,
                CommitmentConfig::processed(),
                RpcSendTransactionConfig {
                    skip_preflight: true,
                    preflight_commitment: None,
                    encoding: None,
                },
            )
            .unwrap();
    }

    pub fn setup() -> (Keypair, RpcClient, Pubkey) {
        let payer =
            read_keypair_file(env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string()))
                .unwrap();
        let rpc =
            RpcClient::new(env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string()));
        let program = env::var("BRIDGE_PROGRAM")
            .unwrap_or("6mFKdAtUBVbsQ5dgvBrUkn1Pixb7BMTUtVKj4dpwrmQs".to_string())
            .parse::<Pubkey>()
            .unwrap();
        (payer, rpc, program)
    }

    pub fn transfer(client: &RpcClient, from: &Keypair, to: &Pubkey, lamports: u64) {
        execute(
            client,
            from,
            &[from],
            &[system_instruction::transfer(&from.pubkey(), to, lamports)],
        );
    }

    pub fn initialize(client: &RpcClient, program: &Pubkey, payer: &Keypair) {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::initialize(*program, payer.pubkey(), 500, 2_000_000_000).unwrap()],
        );
    }

    pub fn post_message(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        sequence: u64,
        data: Vec<u8>,
    ) {
        // Transfer money into the fee collector as it needs a balance/must exist.
        let fee_collector = FeeCollector::<'_>::key(None, program);
        transfer(client, payer, &fee_collector, 1000000);

        execute(
            client,
            payer,
            &[payer, emitter],
            &[instructions::post_message(
                *program,
                payer.pubkey(),
                emitter.pubkey(),
                0,
                data,
            )
                .unwrap()],
        );
    }

    pub fn verify_signatures(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        body: Vec<u8>,
        body_hash: [u8; 32],
        secret_key: SecretKey,
    ) {
        let mut signers = [-1; 19];
        signers[0] = 0;

        execute(
            client,
            payer,
            &[payer],
            &[
                new_secp256k1_instruction(&secret_key, &body),
                instructions::verify_signatures(*program, payer.pubkey(), 0, VerifySignaturesData {
                    hash: body_hash,
                    signers,
                    initial_creation: true,
                }).unwrap(),
            ],
        );
    }

    pub fn post_vaa(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        vaa: PostVAAData,
    ) {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::post_vaa(
                *program,
                payer.pubkey(),
                vaa,
            )],
        );
    }
}
