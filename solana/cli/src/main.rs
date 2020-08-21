use std::fmt::Display;
use std::net::{SocketAddr, ToSocketAddrs};
use std::str::FromStr;
use std::{mem::size_of, process::exit};

use clap::{
    crate_description, crate_name, crate_version, value_t, value_t_or_exit, App, AppSettings, Arg,
    SubCommand,
};
use hex;
use primitive_types::U256;
use solana_account_decoder::{parse_token::TokenAccountType, UiAccountData};
use solana_clap_utils::input_parsers::value_of;
use solana_clap_utils::input_validators::is_derivation;
use solana_clap_utils::{
    input_parsers::{keypair_of, pubkey_of},
    input_validators::{is_amount, is_keypair, is_pubkey_or_keypair, is_url},
};
use solana_client::client_error::ClientError;
use solana_client::{rpc_client::RpcClient, rpc_request::TokenAccountsFilter};
use solana_sdk::system_instruction::create_account;
use solana_sdk::{
    commitment_config::CommitmentConfig,
    native_token::*,
    pubkey::Pubkey,
    signature::{read_keypair_file, Keypair, Signer},
    system_instruction,
    transaction::Transaction,
};
use spl_token::{
    self,
    instruction::*,
    native_mint,
    state::{Account, Mint},
};

use spl_bridge::instruction::*;
use spl_bridge::state::*;

use crate::faucet::request_and_confirm_airdrop;

mod faucet;

struct Config {
    rpc_client: RpcClient,
    owner: Keypair,
    fee_payer: Keypair,
    commitment_config: CommitmentConfig,
}

type Error = Box<dyn std::error::Error>;
type CommmandResult = Result<Option<Transaction>, Error>;

fn command_deploy_bridge(
    config: &Config,
    bridge: &Pubkey,
    initial_guardian: Vec<[u8; 20]>,
) -> CommmandResult {
    println!("Deploying bridge program");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let ix = initialize(
        bridge,
        &config.owner.pubkey(),
        initial_guardian,
        &BridgeConfig {
            vaa_expiration_time: 200000000,
            token_program: spl_token::id(),
        },
    )?;
    println!("bridge: {}, ", ix.accounts[2].pubkey.to_string());
    println!("payer: {}, ", ix.accounts[3].pubkey.to_string());
    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_lock_tokens(
    config: &Config,
    bridge: &Pubkey,
    account: Pubkey,
    token: Pubkey,
    amount: u64,
    to_chain: u8,
    target: ForeignAddress,
    nonce: u32,
) -> CommmandResult {
    println!("Initiating transfer to foreign chain");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let bridge_key = Bridge::derive_bridge_id(bridge)?;

    // Check whether we can find wrapped asset meta for the given token
    let wrapped_key = Bridge::derive_wrapped_meta_id(bridge, &bridge_key, &token)?;
    let asset_meta = match config.rpc_client.get_account(&wrapped_key) {
        Ok(v) => {
            let wrapped_meta: &WrappedAssetMeta =
                Bridge::unpack_unchecked_immutable(v.data.as_slice())?;
            AssetMeta {
                address: wrapped_meta.address,
                chain: wrapped_meta.chain,
            }
        }
        Err(e) => AssetMeta {
            address: token.to_bytes(),
            chain: CHAIN_ID_SOLANA,
        },
    };

    let mut instructions = vec![
        approve(
            &spl_token::id(),
            &account,
            &bridge_key,
            &config.owner.pubkey(),
            &[],
            amount,
        )?,
        transfer_out(
            bridge,
            &config.owner.pubkey(),
            &account,
            &token,
            &TransferOutPayload {
                amount: U256::from(amount),
                chain_id: to_chain,
                asset: asset_meta,
                target,
                nonce,
            },
        )?,
    ];

    println!(
        "custody: {}, ",
        instructions[1].accounts[8].pubkey.to_string()
    );

    let mut transaction =
        Transaction::new_with_payer(&instructions.as_slice(), Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_create_wrapped_asset(
    config: &Config,
    bridge: &Pubkey,
    meta: AssetMeta,
) -> CommmandResult {
    println!("Creating wrapped asset");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let ix = create_wrapped(bridge, &config.owner.pubkey(), meta)?;

    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_submit_vaa(config: &Config, bridge: &Pubkey, vaa: &[u8]) -> CommmandResult {
    println!("Submitting VAA");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let ix = post_vaa(bridge, &config.owner.pubkey(), vaa.to_vec())?;

    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
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

fn check_owner_balance(config: &Config, required_balance: u64) -> Result<(), Error> {
    let balance = config.rpc_client.get_balance(&config.owner.pubkey())?;
    if balance < required_balance {
        Err(format!(
            "Owner, {}, has insufficient balance: {} required, {} available",
            config.owner.pubkey(),
            lamports_to_sol(required_balance),
            lamports_to_sol(balance)
        )
        .into())
    } else {
        Ok(())
    }
}

fn command_create_token(config: &Config, decimals: u8) -> CommmandResult {
    let token = Keypair::new();
    println!("Creating token {}", token.pubkey());

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let mut transaction = Transaction::new_with_payer(
        &[
            system_instruction::create_account(
                &config.fee_payer.pubkey(),
                &token.pubkey(),
                minimum_balance_for_rent_exemption,
                size_of::<Mint>() as u64,
                &spl_token::id(),
            ),
            initialize_mint(
                &spl_token::id(),
                &token.pubkey(),
                None,
                Some(&config.owner.pubkey()),
                0,
                decimals,
            )?,
        ],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(
        &[&config.fee_payer, &config.owner, &token],
        recent_blockhash,
    );
    Ok(Some(transaction))
}

fn command_create_account(config: &Config, token: Pubkey) -> CommmandResult {
    let account = Keypair::new();
    println!("Creating account {}", account.pubkey());

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Account>())?;

    let mut transaction = Transaction::new_with_payer(
        &[
            system_instruction::create_account(
                &config.fee_payer.pubkey(),
                &account.pubkey(),
                minimum_balance_for_rent_exemption,
                size_of::<Account>() as u64,
                &spl_token::id(),
            ),
            initialize_account(
                &spl_token::id(),
                &account.pubkey(),
                &token,
                &config.owner.pubkey(),
            )?,
        ],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(
        &[&config.fee_payer, &config.owner, &account],
        recent_blockhash,
    );
    Ok(Some(transaction))
}

fn command_assign(config: &Config, account: Pubkey, new_owner: Pubkey) -> CommmandResult {
    println!(
        "Assigning {}\n  Current owner: {}\n  New owner: {}",
        account,
        config.owner.pubkey(),
        new_owner
    );

    let mut transaction = Transaction::new_with_payer(
        &[set_owner(
            &spl_token::id(),
            &account,
            &new_owner,
            &config.owner.pubkey(),
            &[],
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_transfer(
    config: &Config,
    sender: Pubkey,
    ui_amount: f64,
    recipient: Pubkey,
) -> CommmandResult {
    println!(
        "Transfer {} tokens\n  Sender: {}\n  Recipient: {}",
        ui_amount, sender, recipient
    );

    let sender_token_balance = config
        .rpc_client
        .get_token_account_balance_with_commitment(&sender, config.commitment_config)?
        .value;

    let amount = spl_token::ui_amount_to_amount(ui_amount, sender_token_balance.decimals);

    let mut transaction = Transaction::new_with_payer(
        &[transfer(
            &spl_token::id(),
            &sender,
            &recipient,
            &config.owner.pubkey(),
            &[],
            amount,
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_burn(config: &Config, source: Pubkey, ui_amount: f64) -> CommmandResult {
    println!("Burn {} tokens\n  Source: {}", ui_amount, source);

    let source_token_balance = config
        .rpc_client
        .get_token_account_balance_with_commitment(&source, config.commitment_config)?
        .value;

    let amount = spl_token::ui_amount_to_amount(ui_amount, source_token_balance.decimals);
    let mut transaction = Transaction::new_with_payer(
        &[burn(
            &spl_token::id(),
            &source,
            &config.owner.pubkey(),
            &[],
            amount,
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_mint(
    config: &Config,
    token: Pubkey,
    ui_amount: f64,
    recipient: Pubkey,
) -> CommmandResult {
    println!(
        "Minting {} tokens\n  Token: {}\n  Recipient: {}",
        ui_amount, token, recipient
    );

    let recipient_token_balance = config
        .rpc_client
        .get_token_account_balance_with_commitment(&recipient, config.commitment_config)?
        .value;
    let amount = spl_token::ui_amount_to_amount(ui_amount, recipient_token_balance.decimals);

    let mut transaction = Transaction::new_with_payer(
        &[mint_to(
            &spl_token::id(),
            &token,
            &recipient,
            &config.owner.pubkey(),
            &[],
            amount,
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_wrap(config: &Config, sol: f64) -> CommmandResult {
    let account = Keypair::new();
    let lamports = sol_to_lamports(sol);
    println!("Wrapping {} SOL into {}", sol, account.pubkey());

    let mut transaction = Transaction::new_with_payer(
        &[
            system_instruction::create_account(
                &config.owner.pubkey(),
                &account.pubkey(),
                lamports,
                size_of::<Account>() as u64,
                &spl_token::id(),
            ),
            initialize_account(
                &spl_token::id(),
                &account.pubkey(),
                &native_mint::id(),
                &config.owner.pubkey(),
            )?,
        ],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_owner_balance(config, lamports)?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(
        &[&config.fee_payer, &config.owner, &account],
        recent_blockhash,
    );
    Ok(Some(transaction))
}

fn command_unwrap(config: &Config, address: Pubkey) -> CommmandResult {
    println!("Unwrapping {}", address);
    println!(
        "  Amount: {} SOL\n  Recipient: {}",
        lamports_to_sol(
            config
                .rpc_client
                .get_balance_with_commitment(&address, config.commitment_config)?
                .value
        ),
        config.owner.pubkey()
    );

    let mut transaction = Transaction::new_with_payer(
        &[close_account(
            &spl_token::id(),
            &address,
            &config.owner.pubkey(),
            &config.owner.pubkey(),
            &[],
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_balance(config: &Config, address: Pubkey) -> CommmandResult {
    let balance = config
        .rpc_client
        .get_token_account_balance_with_commitment(&address, config.commitment_config)?
        .value;

    println!("ui amount: {}", balance.ui_amount);
    println!("decimals: {}", balance.decimals);
    println!("amount: {}", balance.amount);

    Ok(None)
}

fn command_supply(config: &Config, address: Pubkey) -> CommmandResult {
    let supply = config
        .rpc_client
        .get_token_supply_with_commitment(&address, config.commitment_config)?
        .value;

    println!("{}", supply.ui_amount);
    Ok(None)
}

fn command_accounts(config: &Config, token: Option<Pubkey>) -> CommmandResult {
    let accounts = config
        .rpc_client
        .get_token_accounts_by_owner_with_commitment(
            &config.owner.pubkey(),
            match token {
                Some(token) => TokenAccountsFilter::Mint(token),
                None => TokenAccountsFilter::ProgramId(spl_token::id()),
            },
            config.commitment_config,
        )?
        .value;
    if accounts.is_empty() {
        println!("None");
    }

    println!("Account                                      Token                                        Balance");
    println!("-------------------------------------------------------------------------------------------------");
    for keyed_account in accounts {
        let address = keyed_account.pubkey;

        if let UiAccountData::Json(parsed_account) = keyed_account.account.data {
            if parsed_account.program != "spl-token" {
                println!(
                    "{:<44} Unsupported account program: {}",
                    address, parsed_account.program
                );
            } else {
                match serde_json::from_value(parsed_account.parsed) {
                    Ok(TokenAccountType::Account(ui_token_account)) => println!(
                        "{:<44} {:<44} {}",
                        address, ui_token_account.mint, ui_token_account.token_amount.ui_amount
                    ),
                    Ok(_) => println!("{:<44} Unsupported token account", address),
                    Err(err) => println!("{:<44} Account parse failure: {}", address, err),
                }
            }
        } else {
            println!("{:<44} Unsupported account data format", address);
        }
    }
    Ok(None)
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
                .help("JSON RPC URL for the cluster.  Default from the configuration file."),
        )
        .arg(
            Arg::with_name("owner")
                .long("owner")
                .value_name("KEYPAIR")
                .validator(is_keypair)
                .takes_value(true)
                .help(
                    "Specify the token owner account. \
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
        .subcommand(SubCommand::with_name("create-token").about("Create a new token")
            .arg(
                Arg::with_name("decimals")
                    .long("decimals")
                    .validator(|s| {
                        s.parse::<u8>().map_err(|e| format!("{}", e))?;
                        Ok(())
                    })
                    .value_name("DECIMALS")
                    .takes_value(true)
                    .default_value(&default_decimals)
                    .help("Number of base 10 digits to the right of the decimal place"),
            )
        )
        .subcommand(
            SubCommand::with_name("create-account")
                .about("Create a new token account")
                .arg(
                    Arg::with_name("token")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token that the account will hold"),
                ),
        )
        .subcommand(
            SubCommand::with_name("assign")
                .about("Assign a token or token account to a new owner")
                .arg(
                    Arg::with_name("address")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The address of the token account"),
                )
                .arg(
                    Arg::with_name("new_owner")
                        .validator(is_pubkey_or_keypair)
                        .value_name("OWNER_ADDRESS")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("The address of the new owner"),
                ),
        )
        .subcommand(
            SubCommand::with_name("transfer")
                .about("Transfer tokens between accounts")
                .arg(
                    Arg::with_name("sender")
                        .validator(is_pubkey_or_keypair)
                        .value_name("SENDER_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token account address of the sender"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("TOKEN_AMOUNT")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Amount to send, in tokens"),
                )
                .arg(
                    Arg::with_name("recipient")
                        .validator(is_pubkey_or_keypair)
                        .value_name("RECIPIENT_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("The token account address of recipient"),
                ),
        )
        .subcommand(
            SubCommand::with_name("approve")
                .about("Approve token sprending")
                .arg(
                    Arg::with_name("sender")
                        .validator(is_pubkey_or_keypair)
                        .value_name("SENDER_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token account address of the sender"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("TOKEN_AMOUNT")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Amount to send, in tokens"),
                )
                .arg(
                    Arg::with_name("recipient")
                        .validator(is_pubkey_or_keypair)
                        .value_name("RECIPIENT_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("The token account address of recipient"),
                ),
        )
        .subcommand(
            SubCommand::with_name("burn")
                .about("Burn tokens from an account")
                .arg(
                    Arg::with_name("source")
                        .validator(is_pubkey_or_keypair)
                        .value_name("SOURCE_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token account address to burn from"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("TOKEN_AMOUNT")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Amount to burn, in tokens"),
                ),
        )
        .subcommand(
            SubCommand::with_name("mint")
                .about("Mint new tokens")
                .arg(
                    Arg::with_name("token")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token to mint"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("TOKEN_AMOUNT")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Amount to mint, in tokens"),
                )
                .arg(
                    Arg::with_name("recipient")
                        .validator(is_pubkey_or_keypair)
                        .value_name("RECIPIENT_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("The token account address of recipient"),
                ),
        )
        .subcommand(
            SubCommand::with_name("balance")
                .about("Get token account balance")
                .arg(
                    Arg::with_name("address")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token account address"),
                ),
        )
        .subcommand(
            SubCommand::with_name("supply")
                .about("Get token supply")
                .arg(
                    Arg::with_name("address")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The token address"),
                ),
        )
        .subcommand(
            SubCommand::with_name("accounts")
                .about("List all token accounts by owner")
                .arg(
                    Arg::with_name("token")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .help("Limit results to the given token. [Default: list accounts for all tokens]"),
                ),
        )
        .subcommand(
            SubCommand::with_name("wrap")
                .about("Wrap native SOL in a SOL token account")
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("AMOUNT")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("Amount of SOL to wrap"),
                ),
        )
        .subcommand(
            SubCommand::with_name("unwrap")
                .about("Unwrap a SOL token account")
                .arg(
                    Arg::with_name("address")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help("The address of the token account to unwrap"),
                ),
        )
        .subcommand(SubCommand::with_name("airdrop")
            .arg(
                Arg::with_name("faucet_url")
                    .value_name("FAUCET_URL")
                    .takes_value(true)
                    .index(1)
                    .required(true)
                    .help("The address of the faucet"),
            )
            .about("Request an airdrop of 100 SOL"))
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
                        "Specify the bridge program public key"
                    ),
            )
            .arg(
                Arg::with_name("guardian")
                    .validator(is_hex)
                    .value_name("GUADIAN_ADDRESS")
                    .takes_value(true)
                    .index(2)
                    .required(true)
                    .help("Address of the initial guardian"),
            ))
        .subcommand(
            SubCommand::with_name("lock")
                .about("Transfer tokens to another chain")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help(
                            "Specify the bridge program public key"
                        ),
                )
                .arg(
                    Arg::with_name("sender")
                        .validator(is_pubkey_or_keypair)
                        .value_name("SENDER_TOKEN_ACCOUNT_ADDRESS")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("The token account address of the sender"),
                )
                .arg(
                    Arg::with_name("token")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("The mint address"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("AMOUNT")
                        .takes_value(true)
                        .index(4)
                        .required(true)
                        .help("Amount to transfer out"),
                )
                .arg(
                    Arg::with_name("chain")
                        .validator(is_u8)
                        .value_name("CHAIN")
                        .takes_value(true)
                        .index(5)
                        .required(true)
                        .help("Chain to transfer to"),
                )
                .arg(
                    Arg::with_name("nonce")
                        .validator(is_u32)
                        .value_name("NONCE")
                        .takes_value(true)
                        .index(6)
                        .required(true)
                        .help("Nonce of the transfer"),
                ),
        )
        .subcommand(
            SubCommand::with_name("postvaa")
                .about("Submit a VAA to the chain")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help(
                            "Specify the bridge program public key"
                        ),
                )
                .arg(
                    Arg::with_name("vaa")
                        .validator(is_hex)
                        .value_name("HEX_VAA")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("The vaa to be posted"),
                )
        )
        .subcommand(
            SubCommand::with_name("create-wrapped")
                .about("Create new wrapped asset and token account")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help(
                            "Specify the bridge program public key"
                        ),
                )
                .arg(
                    Arg::with_name("chain")
                        .validator(is_u8)
                        .value_name("CHAIN")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Chain ID of the asset"),
                )
                .arg(
                    Arg::with_name("token")
                        .validator(is_hex)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Token address of the asset"),
                )
        )
        .subcommand(
            SubCommand::with_name("wrapped-address")
                .about("Derive wrapped asset address")
                .arg(
                    Arg::with_name("bridge")
                        .long("bridge")
                        .value_name("BRIDGE_KEY")
                        .validator(is_pubkey_or_keypair)
                        .takes_value(true)
                        .index(1)
                        .required(true)
                        .help(
                            "Specify the bridge program public key"
                        ),
                )
                .arg(
                    Arg::with_name("chain")
                        .validator(is_u8)
                        .value_name("CHAIN")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("Chain ID of the asset"),
                )
                .arg(
                    Arg::with_name("token")
                        .validator(is_hex)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Token address of the asset"),
                )
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
            commitment_config: CommitmentConfig::max(),
        }
    };

    solana_logger::setup_with_default("solana=info");

    let _ = match matches.subcommand() {
        ("create-token", Some(arg_matches)) => {
            let decimals = value_t_or_exit!(arg_matches, "decimals", u8);
            command_create_token(&config, decimals)
        }
        ("create-account", Some(arg_matches)) => {
            let token = pubkey_of(arg_matches, "token").unwrap();
            command_create_account(&config, token)
        }
        ("assign", Some(arg_matches)) => {
            let address = pubkey_of(arg_matches, "address").unwrap();
            let new_owner = pubkey_of(arg_matches, "new_owner").unwrap();
            command_assign(&config, address, new_owner)
        }
        ("transfer", Some(arg_matches)) => {
            let sender = pubkey_of(arg_matches, "sender").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", f64);
            let recipient = pubkey_of(arg_matches, "recipient").unwrap();
            command_transfer(&config, sender, amount, recipient)
        }
        ("burn", Some(arg_matches)) => {
            let source = pubkey_of(arg_matches, "source").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", f64);
            command_burn(&config, source, amount)
        }
        ("mint", Some(arg_matches)) => {
            let token = pubkey_of(arg_matches, "token").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", f64);
            let recipient = pubkey_of(arg_matches, "recipient").unwrap();
            command_mint(&config, token, amount, recipient)
        }
        ("wrap", Some(arg_matches)) => {
            let amount = value_t_or_exit!(arg_matches, "amount", f64);
            command_wrap(&config, amount)
        }
        ("unwrap", Some(arg_matches)) => {
            let address = pubkey_of(arg_matches, "address").unwrap();
            command_unwrap(&config, address)
        }
        ("balance", Some(arg_matches)) => {
            let address = pubkey_of(arg_matches, "address").unwrap();
            command_balance(&config, address)
        }
        ("supply", Some(arg_matches)) => {
            let address = pubkey_of(arg_matches, "address").unwrap();
            command_supply(&config, address)
        }
        ("accounts", Some(arg_matches)) => {
            let token = pubkey_of(arg_matches, "token");
            command_accounts(&config, token)
        }
        ("airdrop", Some(arg_matches)) => {
            let faucet_addr = value_t_or_exit!(arg_matches, "faucet_url", String);
            request_and_confirm_airdrop(
                &config.rpc_client,
                &faucet_addr.to_socket_addrs().unwrap().next().unwrap(),
                &config.owner.pubkey(),
                100 * LAMPORTS_PER_SOL,
            )
        }
        ("create-bridge", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let initial_guardian: String = value_of(arg_matches, "guardian").unwrap();
            let initial_data = hex::decode(initial_guardian).unwrap();

            let mut guardian = [0u8; 20];
            guardian.copy_from_slice(&initial_data);
            command_deploy_bridge(&config, &bridge, vec![guardian])
        }
        ("lock", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let account = pubkey_of(arg_matches, "sender").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", u64);
            let nonce = value_t_or_exit!(arg_matches, "nonce", u32);
            let chain = value_t_or_exit!(arg_matches, "chain", u8);
            let token = pubkey_of(arg_matches, "token").unwrap();
            command_lock_tokens(
                &config, &bridge, account, token, amount, chain, [0; 32], nonce,
            )
        }
        ("postvaa", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let vaa_string: String = value_of(arg_matches, "vaa").unwrap();
            let vaa = hex::decode(vaa_string).unwrap();
            command_submit_vaa(&config, &bridge, vaa.as_slice())
        }
        ("create-wrapped", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let chain = value_t_or_exit!(arg_matches, "chain", u8);
            let addr_string: String = value_of(arg_matches, "token").unwrap();
            let addr_data = hex::decode(addr_string).unwrap();

            let mut token_addr = [0u8; 32];
            token_addr.copy_from_slice(addr_data.as_slice());

            command_create_wrapped_asset(
                &config,
                &bridge,
                AssetMeta {
                    chain,
                    address: token_addr,
                },
            )
        }
        ("wrapped-address", Some(arg_matches)) => {
            let bridge = pubkey_of(arg_matches, "bridge").unwrap();
            let chain = value_t_or_exit!(arg_matches, "chain", u8);
            let addr_string: String = value_of(arg_matches, "token").unwrap();
            let addr_data = hex::decode(addr_string).unwrap();

            let mut token_addr = [0u8; 32];
            token_addr.copy_from_slice(addr_data.as_slice());

            let bridge_key = Bridge::derive_bridge_id(&bridge).unwrap();
            let wrapped_key =
                Bridge::derive_wrapped_asset_id(&bridge, &bridge_key, chain, token_addr).unwrap();
            println!("Wrapped address: {}", wrapped_key);
            return;
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
                .send_and_confirm_transaction_with_spinner_and_commitment(
                    &transaction,
                    config.commitment_config,
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

pub fn is_hex<T>(value: T) -> Result<(), String>
where
    T: AsRef<str> + Display,
{
    hex::decode(value.to_string())
        .map(|_| ())
        .map_err(|e| format!("{}", e))
}
