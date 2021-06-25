pub mod cli;

use borsh::BorshSerialize;
use clap::Clap;
use log::LevelFilter;
use solana_client::rpc_client::RpcClient;
use solana_program::{hash::Hash, pubkey::Pubkey};
use solana_sdk::{
    commitment_config::CommitmentConfig, signature::read_keypair_file, transaction::Transaction,
};
use solitaire::{AccountState, processors::seeded::Seeded};
use solitaire_client::{AccEntry, Keypair, SolSigner, ToInstruction};

use cli::{Action, Cli};

use pyth2wormhole::{config::P2WConfigAccount, initialize::InitializeAccounts, Pyth2WormholeConfig};

pub type ErrBox = Box<dyn std::error::Error>;

fn main() -> Result<(), ErrBox> {
    let cli = Cli::parse();
    init_logging(cli.log_level);

    let payer = read_keypair_file(&*shellexpand::tilde(&cli.payer))?;
    let rpc_client = RpcClient::new_with_commitment(cli.rpc_url, CommitmentConfig::processed());

    let p2w_addr = cli.p2w_addr;

    let (recent_blockhash, _) = rpc_client.get_recent_blockhash()?;

    let tx = match cli.action {
        Action::Init {
            new_owner_addr,
            wormhole_addr,
            pyth_owner_addr,
        } => handle_init(
            payer,
            p2w_addr,
            new_owner_addr,
            wormhole_addr,
            pyth_owner_addr,
            recent_blockhash,
        )?,
        Action::Forward { product_addr: _ } => {
            todo!()
        }
    };

    rpc_client.send_and_confirm_transaction_with_spinner(&tx)?;

    Ok(())
}

fn handle_init(
    payer: Keypair,
    p2w_addr: Pubkey,
    new_owner_addr: Pubkey,
    wormhole_addr: Pubkey,
    pyth_owner_addr: Pubkey,
    recent_blockhash: Hash,
) -> Result<Transaction, ErrBox> {
    use AccEntry::*;

    let payer_pubkey = payer.pubkey();

    let accs = InitializeAccounts {
        payer: Signer(payer),
        new_config: Unprivileged(<P2WConfigAccount<'_, {AccountState::Uninitialized}>>::key(None, &p2w_addr)),
    };

    let config = Pyth2WormholeConfig {
        owner: new_owner_addr,
        wormhole_program_addr: wormhole_addr,
        pyth_owner: pyth_owner_addr,
    };
    let ix_data = pyth2wormhole::instruction::Instruction::Initialize(config);

    let (ix, signers) = accs.gen_client_ix(p2w_addr, ix_data.try_to_vec()?.as_slice())?;

    let tx_signed = Transaction::new_signed_with_payer::<Vec<&Keypair>>(
        &[ix],
        Some(&payer_pubkey),
        signers.iter().collect::<Vec<_>>().as_ref(),
        recent_blockhash,
    );
    Ok(tx_signed)
}

fn init_logging(verbosity: u32) {
    use LevelFilter::*;
    let filter = match verbosity {
        0..=1 => Error,
        2 => Warn,
        3 => Info,
        4 => Debug,
        _other => Trace,
    };

    env_logger::builder().filter_level(filter).init();
}
