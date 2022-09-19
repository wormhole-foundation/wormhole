use {
    solana_program::{
        account_info::AccountInfo,
        pubkey::Pubkey,
    },
    wormhole::WormholeError,
};

mod claim;
mod config;
mod emitter;
mod fee_collector;
mod guardian_set;
mod sequence;
mod vaa;

pub use {
    claim::{
        Claim,
        ClaimSeeds,
    },
    config::Config,
    emitter::Emitter,
    fee_collector::FeeCollector,
    guardian_set::GuardianSet,
    sequence::Sequence,
    vaa::VAA,
};

// Account provides helpers for deriving keys and reading data from Wormhole accounts.
pub trait Account: Sized {
    type Seeds;
    type Output;

    fn key(account: &Pubkey, seeds: Self::Seeds) -> Self::Output;
    fn get(account: &AccountInfo) -> Result<Self, WormholeError>;
}
