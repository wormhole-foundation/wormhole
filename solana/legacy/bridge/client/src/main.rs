#![allow(incomplete_features)]
#![feature(adt_const_params)]

use std::{
    fmt::Display,
    mem::size_of,
    process::exit,
};

use borsh::BorshDeserialize;
use bridge::accounts::{
    Bridge,
    BridgeData,
    FeeCollector,
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
        values_of,
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
    system_instruction::transfer,
    transaction::Transaction,
};
use solitaire::{
    processors::seeded::Seeded,
    AccountState,
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
fn command_deploy_bridge(
    config: &Config,
    bridge: &Pubkey,
    initial_guardians: Vec<[u8; 20]>,
    guardian_expiration: u32,
    message_fee: u64,
) -> CommmandResult {
    println!("Initializing Wormhole bridge {}", bridge);

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<BridgeData>())?;

    let ix = bridge::instructions::initialize(
        *bridge,
        config.owner.pubkey(),
        message_fee,
        guardian_expiration,
        initial_guardians.as_slice(),
    )
    .unwrap();
    println!("config account: {}, ", ix.accounts[0].pubkey);
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

// [`get_recent_blockhash`] is deprecated, but devnet deployment hangs using the
// recommended method, so allowing deprecated here. This is only the client, so
// no risk.
#[allow(deprecated)]
fn command_post_message(
    config: &Config,
    bridge: &Pubkey,
    nonce: u32,
    payload: Vec<u8>,
    commitment: bridge::types::ConsistencyLevel,
    proxy: Option<Pubkey>,
) -> CommmandResult {
    println!("Posting a message to the wormhole");

    // Fetch the message fee
    let bridge_config_account = config
        .rpc_client
        .get_account(&Bridge::<'_, { AccountState::Initialized }>::key(
            None, bridge,
        ))?;
    let bridge_config = BridgeData::try_from_slice(bridge_config_account.data.as_slice())?;
    let fee = bridge_config.config.fee;
    println!("Message fee: {} lamports", fee);

    let transfer_ix = transfer(
        &config.owner.pubkey(),
        &FeeCollector::key(None, bridge),
        fee,
    );

    let message = Keypair::new();
    let ix = match proxy {
        Some(p) => cpi_poster::instructions::post_message(
            p,
            *bridge,
            config.owner.pubkey(),
            config.owner.pubkey(),
            message.pubkey(),
            nonce,
            payload,
            commitment,
        )
        .unwrap(),
        None => bridge::instructions::post_message(
            *bridge,
            config.owner.pubkey(),
            config.owner.pubkey(),
            message.pubkey(),
            nonce,
            payload,
            commitment,
        )
        .unwrap(),
    };
    let mut transaction =
        Transaction::new_with_payer(&[transfer_ix, ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(transaction.message()))?;
    transaction.sign(
        &[&config.fee_payer, &config.owner, &message],
        recent_blockhash,
    );
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
            SubCommand::with_name("upgrade-authority")
                .about("Get the derived signer used for contract upgrades")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the bridge program address"),
                )
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
                        .help("Specify the bridge program address"),
                )
                .arg(
                    Arg::with_name("guardian")
                        .validator(is_hex)
                        .value_name("GUARDIAN_ADDRESS")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .require_delimiter(true)
                        .help("Addresses of the initial guardians, comma delimited."),
                )
                .arg(
                    Arg::with_name("guardian_set_expiration")
                        .validator(is_u32)
                        .value_name("GUARDIAN_SET_EXPIRATION")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Time in seconds after which a guardian set expires after an update"),
                )
                .arg(
                    Arg::with_name("message_fee")
                        .validator(is_u64)
                        .value_name("MESSAGE_FEE")
                        .takes_value(true)
                        .index(4)
                        .required(true)
                        .help("Initial message posting fee"),
                ),
        )
        .subcommand(
            SubCommand::with_name("post-message")
                .about("Post a message via Wormhole")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Specify the bridge program address"),
                )
                .arg(
                    Arg::with_name("nonce")
                        .validator(is_u32)
                        .value_name("NONCE")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Nonce of the message"),
                )
                .arg(
                    Arg::with_name("consistency_level")
                        .value_name("CONSISTENCY_LEVEL")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Consistency (Commitment) level at which the VAA should be produced <FINALIZED|CONFIRMED>"),
                )
                .arg(
                    Arg::with_name("data")
                        .validator(is_hex)
                        .value_name("DATA")
                        .takes_value(true)
                        .index(4)
                        .required(true)
                        .help("Payload of the message"),
                )
                .arg(
                    Arg::with_name("proxy")
                        .long("proxy")
                        .validator(is_pubkey_or_keypair)
                        .value_name("PROXY")
                        .takes_value(true)
                        .help("CPI Proxy to use"),
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
            let initial_guardians = values_of::<String>(arg_matches, "guardian").unwrap();
            let initial_data = initial_guardians
                .into_iter()
                .map(|key| hex::decode(key).unwrap());
            let guardians: Vec<[u8; 20]> = initial_data
                .into_iter()
                .map(|key| {
                    let mut guardian = [0u8; 20];
                    guardian.copy_from_slice(&key);
                    guardian
                })
                .collect::<Vec<[u8; 20]>>();
            let guardian_expiration: u32 =
                value_of(arg_matches, "guardian_set_expiration").unwrap();
            let msg_fee: u64 = value_of(arg_matches, "message_fee").unwrap();

            command_deploy_bridge(&config, &bridge, guardians, guardian_expiration, msg_fee)
        }
        ("upgrade-authority", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let upgrade_auth = <Derive<Info<'_>, "upgrade">>::key(None, &bridge);
            println!("Upgrade Key: {}", upgrade_auth);

            Ok(None)
        }
        ("post-message", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let data_str: String = value_of(arg_matches, "data").unwrap();
            let data = hex::decode(data_str).unwrap();
            let nonce: u32 = value_of(arg_matches, "nonce").unwrap();
            let consistency_level: String = value_of(arg_matches, "consistency_level").unwrap();
            let proxy = pubkey_of(arg_matches, "proxy");

            command_post_message(
                &config,
                &bridge,
                nonce,
                data,
                match consistency_level.to_lowercase().as_str() {
                    "finalized" => bridge::types::ConsistencyLevel::Finalized,
                    "confirmed" => bridge::types::ConsistencyLevel::Confirmed,
                    _ => {
                        eprintln!("Invalid commitment level");
                        exit(1);
                    }
                },
                proxy,
            )
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
