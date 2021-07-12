use borsh::BorshSerialize;
use bridge::{api, types};
use clap::Clap;
use solana_client::{rpc_client::RpcClient, rpc_config::RpcSendTransactionConfig};
use solana_program::pubkey::Pubkey;
use solana_sdk::{
    commitment_config::CommitmentConfig,
    signature::{read_keypair_file, Signer as SolSigner},
    transaction::Transaction,
};
use solitaire_client::{AccEntry, ToInstruction};

use bridge::accounts::{GuardianSet, GuardianSetDerivationData};
use solana_client::client_error::{ClientError, ClientErrorKind};
use solana_client::rpc_request::RpcError;
use solana_sdk::client::Client;
use solana_sdk::instruction::Instruction;
use solana_sdk::program_pack::Pack;
use solana_sdk::rent::Rent;
use solana_sdk::signature::Keypair;
use solitaire::{processors::seeded::Seeded, AccountState};
use solitaire_client::solana_sdk::account::ReadableAccount;
use spl_token::instruction::TokenInstruction::Transfer;
use spl_token::state::Mint;
use std::error;
use token_bridge::api::{AttestTokenData, TransferNativeData};

#[derive(Clap)]
pub struct Opts {
    #[clap(long)]
    bridge_address: Pubkey,
    #[clap(long)]
    token_address: Pubkey,
}

pub type ErrBox = Box<dyn error::Error>;

pub const DEFAULT_MESSAGE_FEE: u64 = 42;
pub const DEFAULT_GUARDIAN_SET_EXPIRATION_TIME: u32 = 42;

fn main() -> Result<(), ErrBox> {
    let opts = Opts::parse();

    let payer = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");

    // Keypair is not Clone
    let url = "http://localhost:8899".to_owned();

    let client = RpcClient::new_with_commitment(url, CommitmentConfig::processed());

    let program_id = opts.bridge_address;

    use AccEntry::*;
    let init = api::InitializeAccounts {
        bridge: Derived(program_id.clone()),
        guardian_set: Unprivileged(<GuardianSet<'_, { AccountState::Uninitialized }>>::key(
            &GuardianSetDerivationData { index: 0 },
            &program_id,
        )),
        payer: Signer(payer),
    };

    let init_args = bridge::instruction::Instruction::Initialize(types::BridgeConfig {
        guardian_set_expiration_time: DEFAULT_GUARDIAN_SET_EXPIRATION_TIME,
        fee: DEFAULT_MESSAGE_FEE,
    });

    let ix_data = init_args.try_to_vec()?;

    let payer = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");
    let (ix, signers) = init.gen_client_ix(program_id, ix_data.as_slice())?;
    send_ix_in_tx(&client, ix, &payer, signers.iter().collect())?;

    let payer = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");
    let token_program_id = opts.token_address;
    let init_token =
        token_bridge::instructions::initialize(token_program_id, payer.pubkey(), program_id)
            .unwrap();
    send_ix_in_tx(&client, init_token, &payer, vec![&payer])?;

    // Create a token
    let mint_authority = Keypair::new();
    let mint = Keypair::new();
    let init_mint_account = solana_sdk::system_instruction::create_account(
        &payer.pubkey(),
        &mint.pubkey(),
        Rent::default().minimum_balance(spl_token::state::Mint::LEN),
        spl_token::state::Mint::LEN as u64,
        &spl_token::id(),
    );
    let init_mint = spl_token::instruction::initialize_mint(
        &spl_token::id(),
        &mint.pubkey(),
        &mint_authority.pubkey(),
        None,
        8,
    )?;
    send_ix_in_tx(&client, init_mint_account, &payer, vec![&payer, &mint])?;
    send_ix_in_tx(&client, init_mint, &payer, vec![&payer])?;

    // Attest a token
    let rando = Keypair::new();
    let mint_data = get_mint(&client, &mint.pubkey())?;
    let attest_token = token_bridge::instructions::attest(
        token_program_id,
        program_id,
        payer.pubkey(),
        mint.pubkey(),
        mint_data,
        rando.pubkey(),
        0,
    )
    .unwrap();
    send_ix_in_tx(&client, attest_token, &payer, vec![&payer])?;

    // Create a token account
    let token_authority = Keypair::new();
    let token_acc = Keypair::new();
    let init_token_sys = solana_sdk::system_instruction::create_account(
        &payer.pubkey(),
        &token_acc.pubkey(),
        Rent::default().minimum_balance(spl_token::state::Account::LEN),
        spl_token::state::Account::LEN as u64,
        &spl_token::id(),
    );
    let init_token_account = spl_token::instruction::initialize_account(
        &spl_token::id(),
        &token_acc.pubkey(),
        &mint.pubkey(),
        &token_authority.pubkey(),
    )?;
    send_ix_in_tx(&client, init_token_sys, &payer, vec![&payer, &token_acc])?;
    send_ix_in_tx(&client, init_token_account, &payer, vec![&payer])?;

    // Mint tokens
    let mint_ix = spl_token::instruction::mint_to(
        &spl_token::id(),
        &mint.pubkey(),
        &token_acc.pubkey(),
        &mint_authority.pubkey(),
        &[],
        1000,
    )?;
    send_ix_in_tx(&client, mint_ix, &payer, vec![&payer, &mint_authority])?;

    // Give allowance
    let bridge_token_authority =
        token_bridge::accounts::AuthoritySigner::key(None, &token_program_id);
    let allowance_ix = spl_token::instruction::approve(
        &spl_token::id(),
        &token_acc.pubkey(),
        &bridge_token_authority,
        &token_authority.pubkey(),
        &[],
        1000,
    )?;
    send_ix_in_tx(
        &client,
        allowance_ix,
        &payer,
        vec![&payer, &token_authority],
    )?;

    // Transfer to ETH
    let transfer_eth = token_bridge::instructions::transfer_native(
        token_program_id,
        program_id,
        payer.pubkey(),
        token_acc.pubkey(),
        mint.pubkey(),
        TransferNativeData {
            nonce: 1,
            amount: 500,
            fee: 0,
            target_address: [2; 32],
            target_chain: 2,
        },
    )
    .unwrap();
    send_ix_in_tx(&client, transfer_eth, &payer, vec![&payer])?;
    let transfer_eth = token_bridge::instructions::transfer_native(
        token_program_id,
        program_id,
        payer.pubkey(),
        token_acc.pubkey(),
        mint.pubkey(),
        TransferNativeData {
            nonce: 2,
            amount: 500,
            fee: 0,
            target_address: [2; 32],
            target_chain: 2,
        },
    )
    .unwrap();
    send_ix_in_tx(&client, transfer_eth, &payer, vec![&payer])?;

    Ok(())
}

fn send_ix_in_tx(
    client: &RpcClient,
    ix: Instruction,
    payer: &Keypair,
    signers: Vec<&Keypair>,
) -> Result<(), ClientError> {
    let mut tx = Transaction::new_with_payer(&[ix], Some(&payer.pubkey()));

    let (recent_blockhash, _) = client.get_recent_blockhash()?;
    tx.try_sign(&signers, recent_blockhash)?;
    println!("Transaction signed.");

    let signature = client.send_and_confirm_transaction_with_spinner_and_config(
        &tx,
        CommitmentConfig::processed(),
        RpcSendTransactionConfig {
            skip_preflight: true,
            preflight_commitment: None,
            encoding: None,
        },
    )?;
    println!("Signature: {}", signature);

    Ok(())
}

fn get_mint(client: &RpcClient, mint: &Pubkey) -> Result<Mint, ClientError> {
    let acc = client.get_account(mint)?;
    Mint::unpack(acc.data()).map_err(|e| ClientError {
        request: None,
        kind: ClientErrorKind::Custom(String::from("Could not deserialize mint")),
    })
}
