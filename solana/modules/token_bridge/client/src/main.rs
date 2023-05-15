#![allow(incomplete_features)]
#![feature(adt_const_params)]

use std::{
    fmt::Display,
    mem::size_of,
    process::exit,
};

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
    pubkey::Pubkey,
    signature::{
        read_keypair_file,
        Keypair,
        Signer,
    },
    transaction::Transaction,
};
use solitaire::{
    processors::seeded::Seeded,
    Derive,
    Info,
};

struct Config {
    rpc_client: RpcClient,
    owner: Keypair,
    fee_payer: Keypair,
    commitment_config: CommitmentConfig,
}

type Error = Box<dyn std::error::Error>;
type CommmandResult = Result<Option<Transaction>, Error>;

// [`get_recent_blockhash`] is deprecated, but devnet deployment hangs using the
// recommended method, so allowing deprecated here. This is only the client, so
// no risk.
#[allow(deprecated)]
fn command_init_bridge(config: &Config, bridge: &Pubkey, core_bridge: &Pubkey) -> CommmandResult {
    println!("Initializing Token bridge {}", bridge);

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<token_bridge::types::Config>())?;

    let ix = token_bridge::instructions::initialize(*bridge, config.owner.pubkey(), *core_bridge)
        .unwrap();
    println!("config account: {}, ", ix.accounts[1].pubkey);
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_create_meta(
    config: &Config,
    mint: &Pubkey,
    name: String,
    symbol: String,
    uri: String,
) -> CommmandResult {
    println!("Creating meta for mint {}", mint);

    let meta_acc = Pubkey::find_program_address(
        &[
            "metadata".as_bytes(),
            spl_token_metadata::id().as_ref(),
            mint.as_ref(),
        ],
        &spl_token_metadata::id(),
    )
    .0;
    println!("Meta account: {}", meta_acc);
    let ix = spl_token_metadata::instruction::create_metadata_accounts_v3(
        spl_token_metadata::id(),
        meta_acc,
        *mint,
        config.owner.pubkey(),
        config.owner.pubkey(),
        config.owner.pubkey(),
        name,
        symbol,
        uri,
        None,
        0,
        false,
        false,
        None,
        None,
        None,
    );
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let recent_blockhash = config.rpc_client.get_latest_blockhash()?;
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
                arg.default_value(config_file)
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
        .subcommand(
            SubCommand::with_name("emitter")
                .about("Get the derived emitter used for contract messages")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the token bridge program address"),
                ),
        )
        .subcommand(
            SubCommand::with_name("metadata")
                .about("Get the derived metadata associated with token mints")
                .arg(
                    Arg::with_name("mint")
                        .long("mint")
                        .value_name("MINT_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the token mint to derive metadata for"),
                ),
        )
        .subcommand(
            SubCommand::with_name("create-meta")
                .about("Create token metadata")
                .arg(
                    Arg::with_name("mint")
                        .long("mint")
                        .value_name("MINT")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the mint address"),
                )
                .arg(
                    Arg::with_name("name")
                        .long("name")
                        .value_name("NAME")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Name of the token"),
                )
                .arg(
                    Arg::with_name("symbol")
                        .long("symbol")
                        .value_name("SYMBOL")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Symbol of the token"),
                )
                .arg(
                    Arg::with_name("uri")
                        .long("uri")
                        .value_name("URI")
                        .takes_value(true)
                        .index(4)
                        .required(true)
                        .help("URI of the token metadata"),
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
        ("create-meta", Some(arg_matches)) => {
            let mint = pubkey_of(arg_matches, "mint").unwrap();
            let name: String = value_of(arg_matches, "name").unwrap();
            let symbol: String = value_of(arg_matches, "symbol").unwrap();
            let uri: String = value_of(arg_matches, "uri").unwrap();

            command_create_meta(&config, &mint, name, symbol, uri)
        }
        ("emitter", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let emitter = <Derive<Info<'_>, "emitter">>::key(None, &bridge);
            println!("Emitter Key: {}", emitter);

            Ok(None)
        }
        ("metadata", Some(arg_matches)) => {
            let mint = pubkey_of(arg_matches, "mint").unwrap();
            let meta_acc = Pubkey::find_program_address(
                &[
                    "metadata".as_bytes(),
                    spl_token_metadata::id().as_ref(),
                    mint.as_ref(),
                ],
                &spl_token_metadata::id(),
            )
            .0;
            let meta_info = spl_token_metadata::utils::meta_deser_unchecked(
                &mut config
                    .rpc_client
                    .get_account(&meta_acc)
                    .unwrap()
                    .data
                    .as_slice(),
            )
            .unwrap();
            println!("Key: {:?}", meta_info.key);
            println!("Mint: {}", meta_info.mint);
            println!("Metadata Key: {}", meta_acc);
            println!("Update Authority: {}", meta_info.update_authority);
            println!("Name: {}", meta_info.data.name);
            println!("Symbol: {}", meta_info.data.symbol);
            println!("URI: {}", meta_info.data.uri);
            println!("Mutable: {}", meta_info.is_mutable);

            Ok(None)
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
                        max_retries: None,
                        min_context_slot: None,
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
