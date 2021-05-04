use anchor_bridge::{
    accounts::Initialize,
    instruction::state::New,
    BridgeConfig,
    InitializeData,
    MAX_LEN_GUARDIAN_KEYS,
};
use anchor_client::{
    solana_sdk::{
        account_info::AccountInfo,
        commitment_config::CommitmentConfig,
        instruction::AccountMeta,
        pubkey::Pubkey,
        signature::{read_keypair_file, Keypair, Signer},
        system_instruction,
        system_program,
        sysvar,
    },
    Client,
    Cluster,
    EventContext,
};
use anyhow::Result;
use clap::Clap;
use rand::rngs::OsRng;

use std::time::Duration;

#[derive(Clap)]
pub struct Opts {
    #[clap(long)]
    bridge_address: Pubkey,
}

fn main() -> Result<()> {
    let opts = Opts::parse();

    // Wallet and cluster params.
    let payer = read_keypair_file(&*shellexpand::tilde("~/.config/solana/id.json"))
        .expect("Example requires a keypair file");
    let url = Cluster::Custom(
        "http://localhost:8899".to_owned(),
        "ws://localhost:8900".to_owned(),
    );

    let client = Client::new_with_options(url, payer, CommitmentConfig::processed());

    initialize_bridge(&client, opts.bridge_address)?;
    Ok(())
}

fn initialize_bridge(client: &Client, bridge_address: Pubkey) -> Result<()> {
    let program = client.program(bridge_address);

    let guardian_set_key = Keypair::generate(&mut OsRng);
    let state_key = Keypair::generate(&mut OsRng);

    program
        .state_request()
        .instruction(system_instruction::create_account(
            &program.payer(),
            &guardian_set_key.pubkey(),
            program.rpc().get_minimum_balance_for_rent_exemption(500)?,
            500,
            &program.id(),
        ))
        .instruction(system_instruction::create_account(
            &program.payer(),
            &state_key.pubkey(),
            program.rpc().get_minimum_balance_for_rent_exemption(500)?,
            500,
            &program.id(),
        ))
        .signer(&guardian_set_key)
        // .signer(&state_key)
        .accounts(Initialize {
            payer: program.payer(),
            guardian_set: guardian_set_key.pubkey(),
            state: state_key.pubkey(),
            system_program: system_program::id(),
            clock: sysvar::clock::id(),
            rent: sysvar::rent::id(),
        })
        .new(New {
            data: InitializeData {
                len_guardians: 0,
                initial_guardian_keys: [[0u8; 20]; MAX_LEN_GUARDIAN_KEYS],
                config: BridgeConfig {
                    guardian_set_expiration_time: 0u32,
                },
            },
        })
        .send()?;

    Ok(())
}
