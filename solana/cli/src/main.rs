use std::fmt::Display;
use std::net::{IpAddr, SocketAddr};
use std::str::FromStr;
use std::thread::sleep;
use std::time::Duration;
use std::{mem::size_of, process::exit};

use clap::{
    crate_description, crate_name, crate_version, value_t, value_t_or_exit, App, AppSettings, Arg,
    SubCommand,
};
use primitive_types::U256;
use solana_clap_utils::{
    input_parsers::{keypair_of, pubkey_of},
    input_validators::{is_amount, is_keypair, is_pubkey_or_keypair, is_url},
};
use solana_client::rpc_client::RpcClient;
use solana_client::rpc_request::TokenAccountsFilter;
use solana_faucet::faucet::request_airdrop_transaction;
use solana_sdk::hash::Hash;
use solana_sdk::system_instruction::create_account;
use solana_sdk::{
    native_token::*,
    pubkey::Pubkey,
    signature::{read_keypair_file, Keypair, Signer},
    system_instruction,
    transaction::Transaction,
};
use spl_token::{
    self,
    instruction::*,
    state::{Account, Mint},
};

use spl_bridge::instruction::{
    initialize, post_vaa, transfer_out, BridgeInstruction, ForeignAddress, TransferOutPayload,
    VAAData, CHAIN_ID_SOLANA,
};
use spl_bridge::state::{
    AssetMeta, Bridge, BridgeConfig, ClaimedVAA, GuardianSet, TransferOutProposal,
};
use spl_bridge::syscalls::RawKey;

struct Config {
    rpc_client: RpcClient,
    owner: Keypair,
    fee_payer: Keypair,
}

type Error = Box<dyn std::error::Error>;
type CommmandResult = Result<Option<Transaction>, Error>;

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

fn command_deploy_bridge(config: &Config) -> CommmandResult {
    println!("Deploying bridge program");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let p = Pubkey::from_str("7AeSppn3AjaeYScZsnRf1ZRQvtyo4Ke5gx7PAJ3r7BFp").unwrap();
    let ix = initialize(
        &p,
        &config.owner.pubkey(),
        RawKey {
            x: [8; 32],
            y_parity: true,
        },
        &BridgeConfig {
            vaa_expiration_time: 200000000,
            token_program: spl_token::id(),
        },
    )?;
    println!("bridge: {}, ", ix.accounts[2].pubkey.to_string());
    println!("payer: {}, ", ix.accounts[3].pubkey.to_string());

    let mut ix_c = create_account(
        &config.owner.pubkey(),
        &ix.accounts[2].pubkey,
        100000000,
        size_of::<Bridge>() as u64,
        &p,
    );
    ix_c.accounts[1].is_signer = false;
    let mut ix_c2 = create_account(
        &config.owner.pubkey(),
        &ix.accounts[3].pubkey,
        100000000,
        size_of::<GuardianSet>() as u64,
        &p,
    );
    ix_c2.accounts[1].is_signer = false;
    let mut transaction =
        Transaction::new_with_payer(&[ix_c, ix_c2, ix], Some(&config.fee_payer.pubkey()));

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

    let p = Pubkey::from_str("7AeSppn3AjaeYScZsnRf1ZRQvtyo4Ke5gx7PAJ3r7BFp").unwrap();
    let ix = transfer_out(
        &p,
        &config.owner.pubkey(),
        &account,
        &token,
        &TransferOutPayload {
            amount: U256::from(amount),
            chain_id: to_chain,
            asset: AssetMeta {
                address: token.to_bytes(), // TODO fetch from WASSET (if WASSET)
                chain: CHAIN_ID_SOLANA,    //TODO fetch from WASSET (if WASSET)
            },
            target,
            nonce,
        },
    )?;
    println!("custody: {}, ", ix.accounts[7].pubkey.to_string());

    // Approve tokens
    let mut ix_a = approve(
        &spl_token::id(),
        &account,
        &ix.accounts[3].pubkey,
        &config.owner.pubkey(),
        &[],
        U256::from(amount),
    );

    // TODO remove create calls
    let mut ix_c = create_account(
        &config.owner.pubkey(),
        &ix.accounts[4].pubkey,
        100000000,
        size_of::<TransferOutProposal>() as u64,
        &p,
    );
    ix_c.accounts[1].is_signer = false;
    let mut ix_c2 = create_account(
        &config.owner.pubkey(),
        &ix.accounts[7].pubkey,
        100000000,
        size_of::<Account>() as u64,
        &spl_token::id(),
    );
    ix_c2.accounts[1].is_signer = false;

    let mut transaction =
        Transaction::new_with_payer(&[ix_a, ix_c, ix_c2, ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_submit_vaa(config: &Config, vaa: &[u8]) -> CommmandResult {
    println!("Submitting VAA");

    let minimum_balance_for_rent_exemption = config
        .rpc_client
        .get_minimum_balance_for_rent_exemption(size_of::<Mint>())?;

    let p = Pubkey::from_str("7AeSppn3AjaeYScZsnRf1ZRQvtyo4Ke5gx7PAJ3r7BFp").unwrap();
    let ix = post_vaa(&p, &config.owner.pubkey(), vaa)?;

    let mut transaction = Transaction::new_with_payer(&[ix], Some(&config.fee_payer.pubkey()));

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(
        config,
        minimum_balance_for_rent_exemption + fee_calculator.calculate_fee(&transaction.message()),
    )?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_create_token(config: &Config) -> CommmandResult {
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
                U256::from(0),
                9, // hard code 9 decimal places to match the sol/lamports relationship
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
    let amount = sol_to_lamports(ui_amount);

    let mut transaction = Transaction::new_with_payer(
        &[transfer(
            &spl_token::id(),
            &sender,
            &recipient,
            &config.owner.pubkey(),
            &[],
            U256::from(amount),
        )?],
        Some(&config.fee_payer.pubkey()),
    );

    let (recent_blockhash, fee_calculator) = config.rpc_client.get_recent_blockhash()?;
    check_fee_payer_balance(config, fee_calculator.calculate_fee(&transaction.message()))?;
    transaction.sign(&[&config.fee_payer, &config.owner], recent_blockhash);
    Ok(Some(transaction))
}

fn command_approve(
    config: &Config,
    sender: Pubkey,
    ui_amount: f64,
    recipient: Pubkey,
) -> CommmandResult {
    println!(
        "Approve {} tokens\n  Sender: {}\n  Recipient: {}",
        ui_amount, sender, recipient
    );
    let amount = sol_to_lamports(ui_amount);

    let mut transaction = Transaction::new_with_payer(
        &[approve(
            &spl_token::id(),
            &sender,
            &recipient,
            &config.owner.pubkey(),
            &[],
            U256::from(amount),
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
    let amount = sol_to_lamports(ui_amount);

    let mut transaction = Transaction::new_with_payer(
        &[burn(
            &spl_token::id(),
            &source,
            &config.owner.pubkey(),
            &[],
            U256::from(amount),
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
        "Mint {} tokens\n  Token: {}\n  Recipient: {}",
        ui_amount, token, recipient
    );
    let amount = sol_to_lamports(ui_amount);

    let mut transaction = Transaction::new_with_payer(
        &[mint_to(
            &spl_token::id(),
            &token,
            &recipient,
            &config.owner.pubkey(),
            &[],
            U256::from(amount),
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
                &spl_token::native_mint::id(),
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
        lamports_to_sol(config.rpc_client.get_balance(&address)?),
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
    let balance = config.rpc_client.get_token_account_balance(&address)?;
    println!("{}", balance);
    Ok(None)
}

fn command_supply(_config: &Config, address: Pubkey) -> CommmandResult {
    println!("supply {}", address);
    Ok(None)
}

fn command_accounts(config: &Config, token: Option<Pubkey>) -> CommmandResult {
    let accounts = config.rpc_client.get_token_accounts_by_owner(
        &config.owner.pubkey(),
        match token {
            Some(token) => TokenAccountsFilter::Mint(token),
            None => TokenAccountsFilter::ProgramId(spl_token::id()),
        },
    )?;
    if accounts.is_empty() {
        println!("None");
    }

    println!("Account                                      Token                                        Balance");
    println!("-------------------------------------------------------------------------------------------------");
    for (address, account) in accounts {
        let balance = match config.rpc_client.get_token_account_balance(&address) {
            Ok(response) => response,
            Err(err) => 0,
        };

        println!("{:<44} {:<44} {}", address, account.lamports, balance);
    }
    Ok(None)
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
        .subcommand(SubCommand::with_name("create-token").about("Create a new token"))
        .subcommand(SubCommand::with_name("create-bridge").about("Create a new bridge"))
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
            SubCommand::with_name("lock")
                .about("Transfer tokens to another chain")
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
                    Arg::with_name("token")
                        .validator(is_pubkey_or_keypair)
                        .value_name("TOKEN_ADDRESS")
                        .takes_value(true)
                        .index(2)
                        .required(true)
                        .help("The mint address"),
                )
                .arg(
                    Arg::with_name("amount")
                        .validator(is_amount)
                        .value_name("AMOUNT")
                        .takes_value(true)
                        .index(3)
                        .required(true)
                        .help("Amount to transfer out"),
                )
                .arg(
                    Arg::with_name("chain")
                        .validator(is_u8)
                        .value_name("CHAIN")
                        .takes_value(true)
                        .index(4)
                        .required(true)
                        .help("Chain to transfer to"),
                )
                .arg(
                    Arg::with_name("nonce")
                        .validator(is_u8)
                        .value_name("NONCE")
                        .takes_value(true)
                        .index(5)
                        .required(true)
                        .help("Nonce of the transfer"),
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
        }
    };

    solana_logger::setup_with_default("solana=info");

    let _ = match matches.subcommand() {
        ("create-token", Some(_arg_matches)) => command_create_token(&config),
        ("create-bridge", Some(_arg_matches)) => command_deploy_bridge(&config),
        ("lock", Some(arg_matches)) => {
            let account = pubkey_of(arg_matches, "sender").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", u64);
            let nonce = value_t_or_exit!(arg_matches, "nonce", u32);
            let chain = value_t_or_exit!(arg_matches, "chain", u8);
            let token = pubkey_of(arg_matches, "token").unwrap();
            command_lock_tokens(&config, account, token, amount, chain, [0; 32], nonce)
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
        ("approve", Some(arg_matches)) => {
            let sender = pubkey_of(arg_matches, "sender").unwrap();
            let amount = value_t_or_exit!(arg_matches, "amount", f64);
            let recipient = pubkey_of(arg_matches, "recipient").unwrap();
            command_approve(&config, sender, amount, recipient)
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
        _ => unreachable!(),
    }
    .and_then(|transaction| {
        if let Some(transaction) = transaction {
            // TODO: Upgrade to solana-client 1.3 and
            // `send_and_confirm_transaction_with_spinner_and_commitment()` with single
            // confirmation by default for better UX
            let signature = config
                .rpc_client
                .send_and_confirm_transaction_with_spinner(&transaction)?;
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
