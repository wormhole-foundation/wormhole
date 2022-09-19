//! Helpers to derive and read Wormhole accounts.

use solana_program::pubkey::Pubkey;

mod config;
mod emitter;
mod fee_collector;
mod sequence;
mod vaa;

pub use {
    config::Config,
    emitter::Emitter,
    fee_collector::FeeCollector,
    sequence::Sequence,
    vaa::VAA,
};
use {
    solana_program::account_info::AccountInfo,
    wormhole::WormholeError,
};

// Account is a trait for deriving and/or reading from Wormhole accounts.
pub trait Account: Sized {
    type Seeds;
    type Output;

    fn key(account: &Pubkey, seeds: Self::Seeds) -> Self::Output;
    fn get(account: &AccountInfo) -> Result<Self, WormholeError>;
}
