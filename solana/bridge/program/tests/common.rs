#![allow(warnings)]

use secp256k1::SecretKey;
use borsh::BorshSerialize;
use solana_client::rpc_client::RpcClient;
use solana_client::rpc_config::RpcSendTransactionConfig;
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
        GuardianSet,
        GuardianSetDerivationData,
        SignatureSet,
        SignaturesSetDerivationData,
        Message,
        MessageDerivationData,
    },
    instruction,
    types::{
        BridgeConfig,
        PostedMessage,
        SequenceTracker,
    },
    Initialize,
    PostMessageData,
    PostVAAData,
    UninitializedMessage,
    VerifySignaturesData,
};

use solitaire::processors::seeded::Seeded;
use solitaire::AccountState;

pub use helpers::*;
pub use instructions::*;

mod helpers {
    use super::*;

    fn rpc_send(client: &RpcClient, tx: Transaction) {
        client.send_and_confirm_transaction_with_spinner_and_config(
            &tx,
            CommitmentConfig::processed(),
            RpcSendTransactionConfig {
                skip_preflight: true,
                preflight_commitment: None,
                encoding: None,
            },
        ).unwrap();
    }

    pub fn setup() -> (Keypair, RpcClient, Pubkey) {
        let payer = read_keypair_file(env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string())).unwrap();
        let rpc = RpcClient::new(env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string()));
        let program = env::var("BRIDGE_PROGRAM")
            .unwrap_or("6mFKdAtUBVbsQ5dgvBrUkn1Pixb7BMTUtVKj4dpwrmQs".to_string())
            .parse::<Pubkey>()
            .unwrap();
        (payer, rpc, program)
    }

    pub fn transfer(client: &RpcClient, from: &Keypair, to: &Pubkey, lamports: u64) {
        let signers = vec![from];
        let instructions = [system_instruction::transfer(&from.pubkey(), to, lamports)];
        let mut transaction = Transaction::new_with_payer(&instructions, Some(&from.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;
        transaction.sign(&signers, recent_blockhash);
        rpc_send(client, transaction);
    }

    pub fn initialize(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        guardian_set: GuardianSetDerivationData,
    ) {
        let (bridge, _) = Pubkey::find_program_address(&["Bridge".as_ref()], program);
        let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
            &guardian_set,
            &program,
        );

        let signers = vec![payer];
        let instructions = [instructions::create_initialize(
            *program,
            payer.pubkey(),
            bridge,
            guardian_set,
        )];

        let mut transaction = Transaction::new_with_payer(&instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;

        transaction.sign(&signers, recent_blockhash);
        rpc_send(client, transaction);
    }

    pub fn post_message(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        emitter: &Keypair,
        message: PostedMessage,
        sequence: u64,
    ) {
        let (bridge, _) = Pubkey::find_program_address(&["Bridge".as_ref()], program);
        let (fee_vault, _) = Pubkey::find_program_address(&["Fees".as_ref()], program);
        let (fee_collector, _) = Pubkey::find_program_address(&["fee_collector".as_ref()], program);
        let (sequence_key, _) = Pubkey::find_program_address(&[&emitter.pubkey().to_bytes()], program);

        let message_key = UninitializedMessage::<'_>::key(&MessageDerivationData {
            emitter_key: emitter.pubkey().to_bytes(),
            emitter_chain: message.emitter_chain,
            nonce: message.nonce,
            payload: message.payload.clone(),
        }, program);

        println!("PostMessage: Derived Keys:");
        println!("Bridge:        {}", bridge);
        println!("Fee Vault:     {}", fee_vault);
        println!("Fee Collector: {}", fee_collector);
        println!("Sequence:      {}", sequence_key);
        println!("Message:       {}", message_key);

        // Top up the fee collector with some base funds.
        transfer(client, payer, &fee_collector, 1000000);

        let signers = vec![payer, &emitter];
        let instructions = [instructions::create_post_message(
            *program,
            payer.pubkey(),
            bridge,
            message_key,
            emitter.pubkey(),
            sequence_key,
            fee_collector,
        )];

        let mut transaction = Transaction::new_with_payer(&instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;

        transaction.sign(&signers, recent_blockhash);
        rpc_send(client, transaction);
    }

    pub fn verify_signatures(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        body: Vec<u8>,
        body_hash: [u8; 32],
        secret_key: SecretKey,
        guardian_set: GuardianSetDerivationData,
    ) {
        let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
            &guardian_set,
            &program,
        );

        let signatures = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
            &SignaturesSetDerivationData {
                hash: body_hash,
            },
            &program,
        );

        println!("VerifySignatures: Derived Keys:");
        println!("GuardianSet: {}", guardian_set);
        println!("Signatures:  {}", signatures);

        let signers = vec![payer];
        let instructions = [
            new_secp256k1_instruction(
                &secret_key,
                &body,
            ),

            instructions::create_verify_signatures(
                *program,
                payer.pubkey(),
                guardian_set,
                signatures,
                body_hash,
            )
        ];

        let mut transaction = Transaction::new_with_payer(&instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;

        transaction.sign(&signers, recent_blockhash);
        rpc_send(client, transaction);
    }

    pub fn post_vaa(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        body_hash: [u8; 32],
        message: Vec<u8>,
        emitter: &Keypair,
        guardian_set: GuardianSetDerivationData,
        vaa: PostVAAData,
    ) {
        let (bridge, _) = Pubkey::find_program_address(&["Bridge".as_ref()], program);
        let guardian_set = GuardianSet::<'_, { AccountState::Uninitialized }>::key(
            &guardian_set,
            &program,
        );

        let signatures = SignatureSet::<'_, { AccountState::Uninitialized }>::key(
            &SignaturesSetDerivationData {
                hash: body_hash,
            },
            &program,
        );

        let message = Message::<'_, { AccountState::MaybeInitialized }>::key(
            &MessageDerivationData {
                emitter_key: emitter.pubkey().to_bytes(),
                emitter_chain: 1,
                nonce: 0,
                payload: message,
            },
            &program,
        );

        println!("PostVAA: Derived Key:");
        println!("GuardianSet: {}", guardian_set);
        println!("Signatures:  {}", signatures);

        let signers = vec![payer];
        let instructions = [instructions::create_post_vaa(
            *program,
            payer.pubkey(),
            guardian_set,
            bridge,
            signatures,
            message,
            vaa,
        )];

        let mut transaction = Transaction::new_with_payer(&instructions, Some(&payer.pubkey()));
        let recent_blockhash = client.get_recent_blockhash().unwrap().0;

        transaction.sign(&signers, recent_blockhash);
        rpc_send(client, transaction);
    }
}

mod instructions {
    use super::*;

    pub fn create_initialize(
        program_id: Pubkey,
        payer: Pubkey,
        bridge: Pubkey,
        guardian_set: Pubkey,
    ) -> Instruction {
        Instruction {
            program_id,

            accounts: vec![
                AccountMeta::new(bridge, false),
                AccountMeta::new(guardian_set, false),
                AccountMeta::new(payer, true),
                AccountMeta::new_readonly(sysvar::rent::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
            ],

            data: instruction::Instruction::Initialize(BridgeConfig {
                fee: 500,
                guardian_set_expiration_time: 1_000_000
                    + SystemTime::now()
                        .duration_since(SystemTime::UNIX_EPOCH)
                        .unwrap()
                        .as_secs() as u32,
            })
            .try_to_vec()
            .unwrap(),
        }
    }

    pub fn create_post_message(
        program_id: Pubkey,
        payer: Pubkey,
        bridge: Pubkey,
        message: Pubkey,
        emitter: Pubkey,
        sequence: Pubkey,
        fee_collector: Pubkey,
    ) -> Instruction {
        Instruction {
            program_id,

            accounts: vec![
                AccountMeta::new(bridge, false),
                AccountMeta::new(message, false),
                AccountMeta::new(emitter, true),
                AccountMeta::new(sequence, false),
                AccountMeta::new(payer, true),
                AccountMeta::new(fee_collector, false),
                AccountMeta::new_readonly(sysvar::clock::id(), false),
                AccountMeta::new_readonly(sysvar::rent::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
            ],

            data: instruction::Instruction::PostMessage(PostMessageData {
                nonce: 0,
                payload: vec![],
            })
            .try_to_vec()
            .unwrap(),
        }
    }

    pub fn create_verify_signatures(
        program_id: Pubkey,
        payer: Pubkey,
        guardian_set: Pubkey,
        signature_set: Pubkey,
        hash: [u8; 32],
    ) -> Instruction {
        let mut signers = [-1; 19];
        signers[0] = 0;

        Instruction {
            program_id,

            accounts: vec![
                AccountMeta::new(payer, true),
                AccountMeta::new(guardian_set, false),
                AccountMeta::new(signature_set, false),
                AccountMeta::new_readonly(sysvar::instructions::id(), false),
                AccountMeta::new_readonly(sysvar::rent::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
            ],

            data: instruction::Instruction::VerifySignatures(VerifySignaturesData {
                hash,
                signers,
                initial_creation: true,
            })
            .try_to_vec()
            .unwrap(),
        }
    }

    pub fn create_post_vaa(
        program_id: Pubkey,
        payer: Pubkey,
        guardian_set: Pubkey,
        bridge_info: Pubkey,
        signature_set: Pubkey,
        message: Pubkey,
        vaa: PostVAAData,
    ) -> Instruction {
        Instruction {
            program_id,

            accounts: vec![
                AccountMeta::new(guardian_set, false),
                AccountMeta::new(bridge_info, false),
                AccountMeta::new(signature_set, false),
                AccountMeta::new(message, false),
                AccountMeta::new(payer, true),
                AccountMeta::new_readonly(sysvar::clock::id(), false),
                AccountMeta::new_readonly(sysvar::rent::id(), false),
                AccountMeta::new_readonly(solana_program::system_program::id(), false),
            ],

            data: instruction::Instruction::PostVAA(vaa)
            .try_to_vec()
            .unwrap(),
        }
    }
}
