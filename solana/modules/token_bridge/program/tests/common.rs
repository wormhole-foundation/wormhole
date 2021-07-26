#![allow(warnings)]

use solana_sdk::rent::Rent;
use spl_token::state::Mint;
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

use token_bridge::{
    accounts::*,
    instruction,
    instructions,
    types::*,
    Initialize,
};

use solitaire::{
    processors::seeded::Seeded,
    AccountState,
};

pub use helpers::*;

/// Simple API wrapper for quickly preparing and sending transactions.
pub fn execute(
    client: &RpcClient,
    payer: &Keypair,
    signers: &[&Keypair],
    instructions: &[Instruction],
    commitment_level: CommitmentConfig,
) -> Result<Signature, ClientError> {
    let mut transaction = Transaction::new_with_payer(instructions, Some(&payer.pubkey()));
    let recent_blockhash = client.get_recent_blockhash().unwrap().0;
    transaction.sign(&signers.to_vec(), recent_blockhash);
    client.send_and_confirm_transaction_with_spinner_and_config(
        &transaction,
        commitment_level,
        RpcSendTransactionConfig {
            skip_preflight: true,
            preflight_commitment: None,
            encoding: None,
        },
    )
}

mod helpers {
    use bridge::types::PostedMessage;
    use token_bridge::{CompleteNativeData, TransferNativeData};

    use super::*;

    /// Initialize the test environment, spins up a solana-test-validator in the background so that
    /// each test has a fresh environment to work within.
    pub fn setup() -> (Keypair, RpcClient, Pubkey, Pubkey) {
        let payer = env::var("BRIDGE_PAYER").unwrap_or("./payer.json".to_string());
        let rpc_address = env::var("BRIDGE_RPC").unwrap_or("http://127.0.0.1:8899".to_string());
        let payer = read_keypair_file(payer).unwrap();
        let rpc = RpcClient::new(rpc_address);

        let (program, token_program) = (
            env::var("BRIDGE_PROGRAM")
                .unwrap_or("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o".to_string())
                .parse::<Pubkey>()
                .unwrap(),
            env::var("TOKEN_BRIDGE_PROGRAM")
                .unwrap_or("GatRhT513sQ6VcavtKVHv12S71pcpthjcgVFpmKkTs6H".to_string())
                .parse::<Pubkey>()
                .unwrap(),
        );

        (payer, rpc, program, token_program)
    }

    /// Wait for a single transaction to fully finalize, guaranteeing chain state has been
    /// confirmed. Useful for consistently fetching data during state checks.
    pub fn sync(client: &RpcClient, payer: &Keypair) {
        execute(
            client,
            payer,
            &[payer],
            &[system_instruction::transfer(
                &payer.pubkey(),
                &payer.pubkey(),
                1,
            )],
            CommitmentConfig::finalized(),
        )
        .unwrap();
    }

    /// Fetch account data, the loop is there to re-attempt until data is available.
    pub fn get_account_data<T: BorshDeserialize>(client: &RpcClient, account: &Pubkey) -> T {
        let account = client.get_account(account).unwrap();
        T::try_from_slice(&account.data).unwrap()
    }

    pub fn initialize_bridge(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
    ) -> Result<Signature, ClientError> {
        let initial_guardians = &[[1u8; 20]];
        execute(
            client,
            payer,
            &[payer],
            &[bridge::instructions::initialize(
                *program,
                payer.pubkey(),
                50,
                2_000_000_000,
                initial_guardians,
            )
            .unwrap()],
            CommitmentConfig::processed(),
        )
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
            CommitmentConfig::processed(),
        )
    }

    pub fn initialize(
        client: &RpcClient,
        program: &Pubkey,
        payer: &Keypair,
        bridge: &Pubkey,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::initialize(*program, payer.pubkey(), *bridge).unwrap()],
            CommitmentConfig::processed(),
        )
    }

    pub fn attest(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        payer: &Keypair,
        mint: Pubkey,
        mint_meta: Pubkey,
        nonce: u32,
    ) -> Result<Signature, ClientError> {
        let mint_data = Mint::unpack(
            &client.get_account(&mint)?.data
        ).expect("Could not unpack Mint");

        execute(
            client,
            payer,
            &[payer],
            &[instructions::attest(
                *program,
                *bridge,
                payer.pubkey(),
                mint,
                mint_data,
                mint_meta,
                nonce,
            )
            .unwrap()],
            CommitmentConfig::processed(),
        )
    }

    pub fn transfer_native(
        client: &RpcClient,
        program: &Pubkey,
        bridge: &Pubkey,
        payer: &Keypair,
        from: &Keypair,
        mint: Pubkey,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer],
            &[instructions::transfer_native(
                *program,
                *bridge,
                payer.pubkey(),
                from.pubkey(),
                mint,
                TransferNativeData {
                    nonce: 0,
                    amount: 0,
                    fee: 0,
                    target_address: [0u8; 32],
                    target_chain: 1,
                },
            )
            .unwrap()],
            CommitmentConfig::processed(),
        )
    }

    pub fn create_mint(
        client: &RpcClient,
        payer: &Keypair,
        mint_authority: &Pubkey,
        mint: &Keypair,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, mint],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &mint.pubkey(),
                    Rent::default().minimum_balance(spl_token::state::Mint::LEN),
                    spl_token::state::Mint::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_mint(
                    &spl_token::id(),
                    &mint.pubkey(),
                    mint_authority,
                    None,
                    8,
                ).unwrap(),
            ],
            CommitmentConfig::processed(),
        )
    }

    pub fn create_token_account(
        client: &RpcClient,
        payer: &Keypair,
        token_acc: &Keypair,
        token_authority: Pubkey,
        mint: Pubkey,
    ) -> Result<Signature, ClientError> {
        execute(
            client,
            payer,
            &[payer, token_acc],
            &[
                solana_sdk::system_instruction::create_account(
                    &payer.pubkey(),
                    &token_acc.pubkey(),
                    Rent::default().minimum_balance(spl_token::state::Account::LEN),
                    spl_token::state::Account::LEN as u64,
                    &spl_token::id(),
                ),
                spl_token::instruction::initialize_account(
                    &spl_token::id(),
                    &token_acc.pubkey(),
                    &mint,
                    &token_authority,
                ).unwrap(),
            ],
            CommitmentConfig::processed(),
        )
    }
}
