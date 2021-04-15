use std::{fmt::Display, mem::size_of, net::ToSocketAddrs, ops::Deref, process::exit};

use clap::{
    App, AppSettings, Arg, ArgMatches, crate_description, crate_name, crate_version, SubCommand,
    value_t, value_t_or_exit,
};
use hex;
use primitive_types::U256;
use rand::{
    CryptoRng,
    prelude::StdRng,
    RngCore, rngs::{mock::StepRng, ThreadRng},
};
use solana_account_decoder::{parse_token::TokenAccountType, UiAccountData};
use solana_clap_utils::{
    input_parsers::{keypair_of, pubkey_of, value_of},
    input_validators::{is_amount, is_keypair, is_pubkey_or_keypair, is_url},
};
use solana_client::{
    rpc_client::RpcClient, rpc_config::RpcSendTransactionConfig, rpc_request::TokenAccountsFilter,
};
use solana_sdk::{
    commitment_config::CommitmentConfig,
    native_token::*,
    pubkey::Pubkey,
    signature::{Keypair, read_keypair_file, Signer},
    system_instruction,
    transaction::Transaction,
};
use solana_sdk::program_pack::Pack;
use spl_token::native_mint;
use spl_token::state::Mint;

use spl_bridge::{instruction::*, state::*};

struct Config {
    rpc_client: RpcClient,
    fee_payer: Keypair,
    commitment_config: CommitmentConfig,
}

type Error = Box<dyn std::error::Error>;
type CommandResult = Result<Option<Transaction>, Error>;

fn command_deploy_bridge(
    config: &Config,
    bridge: &Pubkey,
    initial_guardian: Vec<[u8; 20]>,
) -> CommandResult {
    println!("Deploying bridge program {}", bridge);

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let ix = initialize(
        bridge,
        &config.fee_payer.pubkey(),
        initial_guardian,
        &BridgeConfig {
            guardian_set_expiration_time: 200000000,
        },
    )?;
    println!("bridge: {}, ", ix.accounts[2].pubkey.to_string());
    println!("payer: {}, ", ix.accounts[3].pubkey.to_string());
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption
            + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer], recent_blockhash);
    Ok(Some(transaction))
}

fn check_fee_payer_balance(config: &Config, required_balance: u64) -> Result<(), Error> {
    let balance = config.rpc_client.get_balance(&config.fee_payer.pubkey())?;
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

fn main() {
    let default_decimals = &format!("{}", native_mint::DECIMALS);

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
                .help("JSON RPC URL for the cluster. Default from the configuration file."),
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
        .subcommand(SubCommand::with_name("create-bridge")
            .about("Create a new bridge")
            .arg(
                Arg::with_name("bridge")
                    .long("bridge")
                    .value_name("BRIDGE_KEY")
                    .validator(is_pubkey_or_keypair)
                    .takes_value(true)
                    .index(1)
                    .required(true)
                    .help(
                        "Specify the bridge program address"
                    ),
            )
            .arg(
                Arg::with_name("guardian")
                    .validator(is_hex)
                    .value_name("GUARDIAN_ADDRESS")
                    .takes_value(true)
                    .index(2)
                    .required(true)
                    .help("Address of the initial guardian"),
            ))
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

        let fee_payer = keypair_of(&matches, "fee_payer").unwrap_or_else(client_keypair);

        Config {
            rpc_client: RpcClient::new(json_rpc_url),
            fee_payer,
            commitment_config: CommitmentConfig::processed(),
        }
    };

    solana_logger::setup_with_default("solana=info");

    let _ = match matches.subcommand() {
        ("create-bridge", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let initial_guardian: String = value_of(arg_matches, "guardian").unwrap();
            let initial_data = hex::decode(initial_guardian).unwrap();

            let mut guardian = [0u8; 20];
            guardian.copy_from_slice(&initial_data);
            command_deploy_bridge(&config, &bridge, vec![guardian])
        }
        _ => unreachable!(),
    }
        .and_then(|transaction| {
            if let Some(transaction) = transaction {
                // TODO: Upgrade to solana-client 1.3 and
                // `send_and_confirm_transaction_with_spinner_and_commitment()` with single
                // confirmation by default for better UX
                let signature = config
                    .rpc_client
                    .send_and_confirm_transaction_with_spinner_and_config(
                        &transaction,
                        config.commitment_config,
                        RpcSendTransactionConfig {
                            // TODO: move to https://github.com/solana-labs/solana/pull/11792
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

pub fn is_hex<T>(value: T) -> Result<(), String>
    where
        T: AsRef<str> + Display,
{
    hex::decode(value.to_string())
        .map(|_| ())
        .map_err(|e| format!("{}", e))
}
