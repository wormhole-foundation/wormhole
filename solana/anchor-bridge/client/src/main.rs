use borsh::BorshSerialize;
use bridge::{
    api,
    client,
    instruction,
    types,
};
use clap::Clap;
use solana_client::{
    rpc_client::RpcClient,
    rpc_config::RpcSendTransactionConfig,
};
use solana_program::{
    pubkey::Pubkey,
    system_instruction,
    system_program,
    sysvar,
};
use solana_sdk::{
    commitment_config::CommitmentConfig,
    signature::{
        read_keypair_file,
        Signer as SolSigner,
    },
    transaction::Transaction,
};
use solitaire_client::{
    AccEntry,
    Signer,
    ToInstruction,
};

use std::{
    error,
    mem::size_of,
};

#[derive(Clap)]
pub struct Opts {
    #[clap(long)]
    bridge_address: Pubkey,
}

pub type ErrBox = Box<dyn error::Error>;

pub const DEFAULT_MESSAGE_FEE: u64 = 42;
pub const DEFAULT_GUARDIAN_SET_EXPIRATION_TIME: u32 = 42;

fn main() -> Result<(), ErrBox> {
    let opts = Opts::parse();

    let payer = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");

    // Keypair is not Clone
    let payer_for_tx = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");
    let url = "http://localhost:8899".to_owned();

    let client = RpcClient::new(url);

    let program_id = opts.bridge_address;

    use AccEntry::*;
    let init = api::InitializeAccounts {
        bridge: Derived(program_id.clone()),
        guardian_set: Derived(program_id.clone()),
        payer: Signer(payer),
    };

    let init_args = types::BridgeConfig {
        guardian_set_expiration_time: DEFAULT_GUARDIAN_SET_EXPIRATION_TIME,
        fee: DEFAULT_MESSAGE_FEE,
    };

    let ix_data = init_args.try_to_vec()?;

    let (ix, signers) = init.to_ix(program_id, ix_data.as_slice())?;
    let (recent_blockhash, _) = client.get_recent_blockhash()?;
    println!("Instruction ready.");
    println!(
        "Signing for {} signer(s): {:?}",
        signers.len(),
        signers.iter().map(|s| s.pubkey()).collect::<Vec<_>>()
    );

    let mut tx = Transaction::new_with_payer(&[ix], Some(&payer_for_tx.pubkey()));

    tx.try_sign(&signers.iter().collect::<Vec<_>>(), recent_blockhash)?;
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
