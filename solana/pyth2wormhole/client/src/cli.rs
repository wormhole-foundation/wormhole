//! CLI options

use solana_program::pubkey::Pubkey;

use clap::Clap;
#[derive(Clap)]
#[clap(
    about = "A client for the pyth2wormhole Solana program",
    author = "The Wormhole Project"
)]
pub struct Cli {
    #[clap(
        short,
        long,
        default_value = "3",
        about = "Logging level, where 0..=1 RUST_LOG=error and 5.. is RUST_LOG=trace"
    )]
    pub log_level: u32,
    #[clap(
        long,
        about = "Identity JSON file for the entity meant to cover transaction costs",
        default_value = "~/.config/solana/id.json"
    )]
    pub payer: String,
    #[clap(long, default_value = "http://localhost:8899")]
    pub rpc_url: String,
    pub p2w_addr: Pubkey,
    #[clap(subcommand)]
    pub action: Action,
}

#[derive(Clap)]
pub enum Action {
    #[clap(about = "Initialize a pyth2wormhole program freshly deployed under <p2w_addr>")]
    Init {
        #[clap(long = "owner")]
        new_owner_addr: Pubkey,
        #[clap(long = "wormhole")]
        wormhole_addr: Pubkey,
        #[clap(long = "pyth")]
        pyth_owner_addr: Pubkey,
    },
    #[clap(
        about = "Use an existing pyth2wormhole program to forward product price information to another chain"
    )]
    Forward {
        #[clap(long = "product")]
        product_addr: Pubkey,
    },
}
