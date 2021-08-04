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
    #[clap(long)]
    pub p2w_addr: Pubkey,
    #[clap(subcommand)]
    pub action: Action,
}

#[derive(Clap)]
pub enum Action {
    #[clap(about = "Initialize a pyth2wormhole program freshly deployed under <p2w_addr>")]
    Init {
	/// The bridge program account
	#[clap(long = "wh-prog")]
	wh_prog: Pubkey,
	#[clap(long = "owner")]
        owner_addr: Pubkey,
        #[clap(long = "pyth-owner")]
        pyth_owner_addr: Pubkey,
    },
    #[clap(
        about = "Use an existing pyth2wormhole program to attest product price information to another chain"
    )]
    Attest {
        #[clap(long = "product")]
        product_addr: Pubkey,
        #[clap(long = "price")]
        price_addr: Pubkey,
        #[clap(long)]
	nonce: u32,
    },
    #[clap(about = "Update an existing pyth2wormhole program's settings (currently set owner only)")]
    SetConfig {
	/// Current owner keypair path
        #[clap(long = "owner", default_value = "~/.config/solana/id.json")]
	owner: String,
	/// New owner to set 
        #[clap(long = "new-owner")]
	new_owner_addr: Pubkey,
        #[clap(long = "new-wh-prog")]
	new_wh_prog: Pubkey,
        #[clap(long = "new-pyth-owner")]
	new_pyth_owner_addr: Pubkey,
    },
}
