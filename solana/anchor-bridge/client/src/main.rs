use bridge::{
    api,
    client,
    instruction,
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
use solitaire::{
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

    let ix_data = vec![];

    let (ix, signers) = init.to_ix(program_id, ix_data.as_slice())?;
    let (recent_blockhash, _) = client.get_recent_blockhash()?;

    let mut tx = Transaction::new_with_payer(&[ix], Some(&payer_for_tx.pubkey()));

    tx.sign(&signers.iter().collect::<Vec<_>>(), recent_blockhash);

    let signature = client.send_and_confirm_transaction_with_spinner_and_config(
        &tx,
        CommitmentConfig::processed(),
        RpcSendTransactionConfig {
            skip_preflight: false,
            preflight_commitment: None,
            encoding: None,
        },
    )?;
    println!("Signature: {}", signature);

    Ok(())
}

// fn initialize_bridge(client: &Client, bridge_address: Pubkey) -> Result<()> {
//     let program = client.program(bridge_address);

//     let guardian_set_key = Keypair::generate(&mut OsRng);
//     let state_key = Keypair::generate(&mut OsRng);

//     program
//         .state_request()
//         .instruction(system_instruction::create_account(
//             &program.payer(),
//             &guardian_set_key.pubkey(),
//             program.rpc().get_minimum_balance_for_rent_exemption(500)?,
//             500,
//             &program.id(),
//         ))
//         .instruction(system_instruction::create_account(
//             &program.payer(),
//             &state_key.pubkey(),
//             program.rpc().get_minimum_balance_for_rent_exemption(500)?,
//             500,
//             &program.id(),
//         ))
//         .signer(&guardian_set_key)
//         // .signer(&state_key)
//         .accounts(Initialize {
//             payer: program.payer(),
//             guardian_set: guardian_set_key.pubkey(),
//             state: state_key.pubkey(),
//             system_program: system_program::id(),
//             clock: sysvar::clock::id(),
//             rent: sysvar::rent::id(),
//         })
//         .new(New {
//             data: InitializeData {
//                 len_guardians: 0,
//                 initial_guardian_keys: [[0u8; 20]; MAX_LEN_GUARDIAN_KEYS],
//                 config: BridgeConfig {
//                     guardian_set_expiration_time: 0u32,
//                 },
//             },
//         })
//         .send()?;

//     Ok(())
// }
