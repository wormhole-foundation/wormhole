//! On-chain state for the pyth2wormhole SOL contract.
//!
//! Important: A config init/update should be performed on every
//! deployment/upgrade of this Solana program. Doing so prevents
//! problems related to max batch size mismatches between config and
//! contract logic. See attest.rs for details.

use borsh::{BorshDeserialize, BorshSerialize};
use solana_program::pubkey::Pubkey;
use solitaire::{processors::seeded::AccountOwner, AccountState, Data, Derive, Owned};

#[derive(Default, BorshDeserialize, BorshSerialize)]
pub struct Pyth2WormholeConfig {
    ///  Authority owning this contract
    pub owner: Pubkey,
    /// Wormhole bridge program
    pub wh_prog: Pubkey,
    /// Authority owning Pyth price data
    pub pyth_owner: Pubkey,
    /// How many product/price pairs can be sent and attested at once
    ///
    /// Important: Whenever the corresponding logic in attest.rs
    /// changes its expected number of symbols per batch, this config
    /// must be updated accordingly on-chain.
    pub max_batch_size: u16,
}

impl Owned for Pyth2WormholeConfig {
    fn owner(&self) -> AccountOwner {
        AccountOwner::This
    }
}

pub type P2WConfigAccount<'b, const IsInitialized: AccountState> =
    Derive<Data<'b, Pyth2WormholeConfig, { IsInitialized }>, "pyth2wormhole-config">;
