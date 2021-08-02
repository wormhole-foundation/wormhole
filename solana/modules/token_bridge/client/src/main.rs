#![feature(const_generics)]
#![allow(warnings)]

use std::{
    fmt::Display,
    mem::size_of,
    process::exit,
};

use borsh::BorshDeserialize;
use clap::{
    crate_description,
    crate_name,
    crate_version,
    value_t,
    App,
    AppSettings,
    Arg,
    SubCommand,
};
use hex;
use solana_clap_utils::{
    input_parsers::{
        keypair_of,
        pubkey_of,
        value_of,
    },
    input_validators::{
        is_keypair,
        is_pubkey_or_keypair,
        is_url,
    },
};
use solana_client::{
    rpc_client::RpcClient,
    rpc_config::RpcSendTransactionConfig,
};
use solana_sdk::{
    commitment_config::{
        CommitmentConfig,
        CommitmentLevel,
    },
    native_token::*,
    program_error::ProgramError::AccountAlreadyInitialized,
    pubkey::Pubkey,
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    system_instruction::transfer,
    transaction::Transaction,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
    Info,
};
use solitaire_client::Derive;

struct Config {
    rpc_client: RpcClient,
    owner: Keypair,
    fee_payer: Keypair,
    commitment_config: CommitmentConfig,
}

type Error = Box<dyn std::error::Error>;
type CommmandResult = Result<Option<Transaction>, Error>;

fn command_init_bridge(config: &Config, bridge: &Pubkey, core_bridge: &Pubkey) -> CommmandResult {
    println!("Initializing Token bridge {}", bridge);

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<token_bridge::types::Config>())?;

    let ix = token_bridge::instructions::initialize(*bridge, config.owner.pubkey(), *core_bridge)
        .unwrap();
    println!("config account: {}, ", ix.accounts[1].pubkey.to_string());
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn main() {
    let matches = App::new(crate_name!())
        .about(crate_description!())
        .version(crate_version!())
        .setting(AppSettings::SubcommandRequiredElseHelp)
        .arg({
            let arg = Arg::with_name("config_file")
                .short("C")
                .long("config")
                .value_name("PATH")
                .takes_value(true)
                .global(true)
                .help("Configuration file to use");
            if let Some(ref config_file) = *solana_cli_config::CONFIG_FILE {
                arg.default_value(&config_file)
            } else {
                arg
            }
        })
        .arg(
            Arg::with_name("json_rpc_url")
                .long("url")
                .value_name("URL")
                .takes_value(true)
                .validator(is_url)
                .help("JSON RPC URL for the cluster.  Default from the configuration file."),
        )
        .arg(
            Arg::with_name("owner")
                .long("owner")
                .value_name("KEYPAIR")
                .validator(is_keypair)
                .takes_value(true)
                .help(
                    "Specify the contract payer account. \
                     This may be a keypair file, the ASK keyword. \
                     Defaults to the client keypair.",
                ),
        )
        .arg(
            Arg::with_name("fee_payer")
                .long("fee-payer")
                .value_name("KEYPAIR")
                .validator(is_keypair)
                .takes_value(true)
                .help(
                    "Specify the fee-payer account. \
                     This may be a keypair file, the ASK keyword. \
                     Defaults to the client keypair.",
                ),
        )
        .subcommand(
            SubCommand::with_name("create-bridge")
                .about("Create a new bridge")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the token bridge program address"),
                )
                .arg(
                    Arg::with_name("core-bridge")
                        .validator(is_pubkey_or_keypair)
                        .value_name("CORE_BRIDGE_KEY")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Address of the Wormhole core bridge program"),
                ),
        )
        .get_matches();

    let config = {
        let cli_config = if let Some(config_file) = matches.value_of("config_file") {
            solana_cli_config::Config::load(config_file).unwrap_or_default()
        } else {
            solana_cli_config::Config::default()
        };
        let json_rpc_url = value_t!(matches, "json_rpc_url", String)
            .unwrap_or_else(|_| cli_config.json_rpc_url.clone());

        let client_keypair = || {
            read_keypair_file(&cli_config.keypair_path).unwrap_or_else(|err| {
                eprintln!("Unable to read {}: {}", cli_config.keypair_path, err);
                exit(1)
            })
        };

        let owner = keypair_of(&matches, "owner").unwrap_or_else(client_keypair);
        let fee_payer = keypair_of(&matches, "fee_payer").unwrap_or_else(client_keypair);

        Config {
            rpc_client: RpcClient::new(json_rpc_url),
            owner,
            fee_payer,
            commitment_config: CommitmentConfig::processed(),
        }
    };

    let _ = match matches.subcommand() {
        ("create-bridge", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let core_bridge = pubkey_of(arg_matches, "core-bridge").unwrap();

            command_init_bridge(&config, &bridge, &core_bridge)
        }

        _ => unreachable!(),
    }
    .and_then(|transaction| {
        if let Some(transaction) = transaction {
            let signature = config
                .rpc_client
                .send_and_confirm_transaction_with_spinner_and_config(
                    &transaction,
                    config.commitment_config,
                    RpcSendTransactionConfig {
                        skip_preflight: true,
                        preflight_commitment: None,
                        encoding: None,
                    },
                )?;
            println!("Signature: {}", signature);
        }
        Ok(())
    })
    .map_err(|err| {
        eprintln!("{}", err);
        exit(1);
    });
}

pub fn is_u8<T>(amount: T) -> Result<(), String>
where
    T: AsRef<str> + Display,
{
    if amount.as_ref().parse::<u8>().is_ok() {
        Ok(())
    } else {
        Err(format!(
            "Unable to parse input amount as integer, provided: {}",
            amount
        ))
    }
}

pub fn is_u32<T>(amount: T) -> Result<(), String>
where
    T: AsRef<str> + Display,
{
    if amount.as_ref().parse::<u32>().is_ok() {
        Ok(())
    } else {
        Err(format!(
            "Unable to parse input amount as integer, provided: {}",
            amount
        ))
    }
}

pub fn is_u64<T>(amount: T) -> Result<(), String>
where
    T: AsRef<str> + Display,
{
    if amount.as_ref().parse::<u64>().is_ok() {
        Ok(())
    } else {
        Err(format!(
            "Unable to parse input amount as integer, provided: {}",
            amount
        ))
    }
}

pub fn is_hex<T>(value: T) -> Result<(), String>
where
    T: AsRef<str> + Display,
{
    hex::decode(value.to_string())
        .map(|_| ())
        .map_err(|e| format!("{}", e))
}

fn check_fee_payer_balance(config: &Config, required_balance: u64) -> Result<(), Error> {
    let balance = config
        .rpc_client
        .get_balance_with_commitment(
            &config.fee_payer.pubkey(),
            CommitmentConfig {
                commitment: CommitmentLevel::Processed,
            },
        )?
        .value;
    if balance < required_balance {
        Err(format!(
            "Fee payer, {}, has insufficient balance: {} required, {} available",
            config.fee_payer.pubkey(),
            lamports_to_sol(required_balance),
            lamports_to_sol(balance)
        )
        .into())
    } else {
        Ok(())
    }
}
